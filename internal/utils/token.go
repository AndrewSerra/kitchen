package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/AndrewSerra/kitchen/internal/logger"
	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

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

func HashToken(token string) (string, error) {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:]), nil
}

func CompareTokenToHash(token string, hash string) (bool, error) {
	tokenHash, err := HashToken(token)
	if err != nil {
		return false, fmt.Errorf("hashing token: %w", err)
	}
	return tokenHash == hash, nil
}
