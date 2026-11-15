package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/AndrewSerra/kitchen/internal/logger"
	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var TokenUnexpectedSigningMethodError = errors.New("unexpected signing method")
var TokenExpiredError = errors.New("token expired")
var TokenInvalidError = errors.New("token invalid")
var TokenNotYetValidError = errors.New("token not yet valid")
var TokenParsingError = errors.New("token parsing error")
var TokenClaimParsingError = errors.New("token claim parsing error")
var TokenIDMismatchError = errors.New("token ID mismatch")
var TokenInactiveError = errors.New("token inactive")
var TokenUserIDNotFoundError = errors.New("user ID not found in token claims")
var TokenTokenIDNotFoundError = errors.New("token ID not found in token claims")

type AccessTokenPayload struct {
	UserID int `json:"user_id"`
}

type RefreshTokenPayload struct {
	TokenID uuid.UUID `json:"token_id"`
}

type TokenGenerator struct {
	issuer   string
	audience string
	subject  string
	method   jwt.SigningMethod

	refreshTokenExpiresIn time.Duration
	accessTokenExpiresIn  time.Duration

	namespace string
}

func NewTokenGenerator() *TokenGenerator {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		logger.Error("JWT_SECRET is not set in the environment variables")
		os.Exit(1)
	}

	namespace := "kitchen"

	return &TokenGenerator{
		issuer:                namespace,
		audience:              namespace,
		subject:               namespace,
		method:                jwt.SigningMethodHS256,
		refreshTokenExpiresIn: 24 * time.Hour,
		accessTokenExpiresIn:  15 * time.Minute,
		namespace:             namespace,
	}
}

func (t *TokenGenerator) getTokenSecret() string {
	return os.Getenv("JWT_SECRET")
}

func (t *TokenGenerator) GenerateAccessToken(payload AccessTokenPayload) (*models.AccessToken, error) {
	now := time.Now()
	token := jwt.NewWithClaims(t.method, jwt.MapClaims{
		"iss": t.issuer,
		"aud": t.audience,
		"sub": t.subject,
		"exp": jwt.NewNumericDate(now.Add(t.accessTokenExpiresIn)),
		"iat": jwt.NewNumericDate(now),
		"nbf": jwt.NewNumericDate(now),
		fmt.Sprintf("%s/%s", t.namespace, "user_id"): payload.UserID,
	})
	return token, nil
}

func (t *TokenGenerator) GenerateRefreshToken(payload RefreshTokenPayload) (*models.RefreshToken, error) {
	now := time.Now()
	token := jwt.NewWithClaims(t.method, jwt.MapClaims{
		"iss": t.issuer,
		"aud": t.audience,
		"sub": t.subject,
		"exp": jwt.NewNumericDate(now.Add(t.refreshTokenExpiresIn)),
		"iat": jwt.NewNumericDate(now),
		"nbf": jwt.NewNumericDate(now),
		"jti": payload.TokenID,
	})
	return token, nil
}

func (t *TokenGenerator) GenerateTokenSet(userID int, tokenID string) (*models.TokenSet, error) {

	refreshToken, err := t.GenerateRefreshToken(RefreshTokenPayload{
		TokenID: uuid.MustParse(tokenID),
	})
	if err != nil {
		return nil, err
	}

	accessToken, err := t.GenerateAccessToken(AccessTokenPayload{
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}
	return &models.TokenSet{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, nil
}

func (t *TokenGenerator) SignToken(token *jwt.Token) (string, error) {
	secret := t.getTokenSecret()
	return token.SignedString([]byte(secret))
}

func (t *TokenGenerator) VerifyToken(token string) (*jwt.Token, error) {
	secret := t.getTokenSecret()
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, TokenUnexpectedSigningMethodError
		}
		return []byte(secret), nil
	},
		jwt.WithAudience(t.audience),
		jwt.WithSubject(t.subject),
		jwt.WithIssuer(t.issuer),
		jwt.WithNotBeforeRequired(),
		jwt.WithExpirationRequired())
	if err != nil {
		return nil, t.mapVerifyTokenError(err)
	}

	return parsedToken, nil
}

// GetUserIDFromToken extracts user_id from an access token (strict expiry).
func (t *TokenGenerator) GetUserIDFromToken(token string) (int, error) {
	verifiedToken, err := t.VerifyToken(token)
	if err != nil {
		return 0, err
	}

	claims, ok := verifiedToken.Claims.(jwt.MapClaims)
	if !ok {
		return 0, TokenClaimParsingError
	}
	raw, ok := claims[fmt.Sprintf("%s/%s", t.namespace, "user_id")]
	if !ok {
		return 0, TokenUserIDNotFoundError
	}
	return parseUserID(raw)
}

func (t *TokenGenerator) GetTokenIDFromToken(token string) (uuid.UUID, error) {
	verifiedToken, err := t.VerifyToken(token)
	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := verifiedToken.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, TokenClaimParsingError
	}
	if _, ok := claims["jti"]; !ok {
		return uuid.Nil, TokenTokenIDNotFoundError
	}
	return t.ParseTokenID(claims["jti"])
}

// GetTokenIDFromTokenAllowExpired extracts jti from a refresh token.
// Expiry is not required so logout can revoke expired sessions.
func (t *TokenGenerator) GetTokenIDFromTokenAllowExpired(token string) (uuid.UUID, error) {
	verifiedToken, err := t.verifyTokenAllowExpired(token)
	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := verifiedToken.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, TokenClaimParsingError
	}
	if _, ok := claims["jti"]; !ok {
		return uuid.Nil, TokenTokenIDNotFoundError
	}
	return t.ParseTokenID(claims["jti"])
}

func (t *TokenGenerator) verifyTokenAllowExpired(token string) (*jwt.Token, error) {
	secret := t.getTokenSecret()
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, TokenUnexpectedSigningMethodError
		}
		return []byte(secret), nil
	},
		jwt.WithAudience(t.audience),
		jwt.WithSubject(t.subject),
		jwt.WithIssuer(t.issuer))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			// Signature and other claims were validated; allow expired for logout.
			if parsedToken != nil {
				return parsedToken, nil
			}
		}
		return nil, t.mapVerifyTokenError(err)
	}
	return parsedToken, nil
}

func parseUserID(raw any) (int, error) {
	switch v := raw.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	default:
		return 0, TokenUserIDNotFoundError
	}
}

func (t *TokenGenerator) ParseTokenID(jtiClaim any) (uuid.UUID, error) {
	switch v := jtiClaim.(type) {
	case string:
		tokenID, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, TokenIDMismatchError
		}
		return tokenID, nil
	case uuid.UUID:
		return v, nil
	default:
		return uuid.Nil, TokenIDMismatchError
	}
}

func (t *TokenGenerator) mapVerifyTokenError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return TokenExpiredError
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return TokenNotYetValidError
	case errors.Is(err, TokenUnexpectedSigningMethodError):
		return TokenInvalidError
	case errors.Is(err, jwt.ErrTokenMalformed),
		errors.Is(err, jwt.ErrTokenSignatureInvalid),
		errors.Is(err, jwt.ErrTokenUnverifiable):
		return TokenParsingError
	default:
		return TokenParsingError
	}
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func CompareTokenToHash(token string, hash string) bool {
	return HashToken(token) == hash
}
