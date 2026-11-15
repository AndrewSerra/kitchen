package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/AndrewSerra/kitchen/internal/db"
	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/AndrewSerra/kitchen/internal/utils"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthRoute struct {
	db             *db.DBService
	tokenGenerator *utils.TokenGenerator
}

func NewAuthRoute(db *db.DBService) *AuthRoute {
	return &AuthRoute{
		db:             db,
		tokenGenerator: utils.NewTokenGenerator(),
	}
}

func (h *AuthRoute) Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Post("/login", h.Login)
	router.Post("/register", h.Register)
	router.Post("/refresh", h.Refresh)
	router.Post("/logout", h.Logout)
	return router
}

func (h *AuthRoute) Login(w http.ResponseWriter, r *http.Request) {
	var loginData models.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		responseResourceBadRequest(w, map[string]string{"message": "invalid JSON"})
		return
	}

	if err := utils.ValidateLoginRequest(&loginData); err != nil {
		responseResourceBadRequest(w, map[string]string{"message": err.Error()})
		return
	}

	var loginResponse models.LoginResponse

	user, tokenSet, err := h.db.ValidateUser(loginData.Email, loginData.Password)
	if err != nil {
		if errors.Is(err, db.UserNotFoundError) || errors.Is(err, db.InvalidCredentialsError) {
			loginResponse.Success = false
			loginResponse.Message = "invalid credentials"

			responseResourceUnauthorized(w, loginResponse)
			return
		}

		loginResponse.Success = false
		loginResponse.Message = "could not validate user"

		responseResourceInternalServerError(w, err, loginResponse)
		return
	}

	user.Password = ""

	loginResponse.Success = true
	loginResponse.Message = "Login successful"
	loginResponse.User = user
	loginResponse.AccessToken = tokenSet.AccessToken
	loginResponse.RefreshToken = tokenSet.RefreshToken

	responseResourceOK(w, loginResponse)
}

func (h *AuthRoute) Register(w http.ResponseWriter, r *http.Request) {
	var registerData models.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&registerData); err != nil {
		responseResourceBadRequest(w, map[string]string{"message": "invalid JSON"})
		return
	}

	if err := utils.ValidateRegisterRequest(&registerData); err != nil {
		responseResourceBadRequest(w, map[string]string{"message": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(registerData.Password), bcrypt.DefaultCost)
	if err != nil {
		responseResourceInternalServerError(w, err, map[string]string{"message": "could not hash password"})
		return
	}
	registerData.Password = string(hash)

	var registerResponse models.RegisterResponse

	user, tokenSet, err := h.db.CreateUser(&registerData)
	if errors.Is(err, db.UserExistsError) {
		registerResponse.Success = false
		registerResponse.Message = "User already exists"
		registerResponse.User = user

		responseResourceConflict(w, registerResponse)
		return
	}
	if err != nil {
		registerResponse.Success = false
		registerResponse.Message = "could not create user"
		registerResponse.User = user

		responseResourceInternalServerError(w, err, registerResponse)
		return
	}

	registerResponse.Success = true
	registerResponse.Message = "User created successfully"
	registerResponse.User = user
	registerResponse.RefreshToken = tokenSet.RefreshToken
	registerResponse.AccessToken = tokenSet.AccessToken

	responseResourceCreated(w, registerResponse)
}

func (h *AuthRoute) Refresh(w http.ResponseWriter, r *http.Request) {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		responseResourceUnauthorized(w, models.RefreshResponse{
			Success: false,
			Message: "missing authorization header",
		})
		return
	}

	parts := strings.SplitN(authorizationHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		responseResourceUnauthorized(w, models.RefreshResponse{
			Success: false,
			Message: "invalid authorization header",
		})
		return
	}
	token := parts[1]

	tokenSet, err := h.db.RefreshTokens(token)
	if err != nil {
		if errors.Is(err, utils.TokenExpiredError) ||
			errors.Is(err, utils.TokenNotYetValidError) ||
			errors.Is(err, utils.TokenParsingError) ||
			errors.Is(err, utils.TokenInvalidError) ||
			errors.Is(err, utils.TokenClaimParsingError) ||
			errors.Is(err, utils.TokenIDMismatchError) ||
			errors.Is(err, db.TokenNotExistsError) ||
			errors.Is(err, utils.TokenInactiveError) ||
			errors.Is(err, db.TokenHashMismatchError) {
			responseResourceUnauthorized(w, models.RefreshResponse{
				Success: false,
				Message: "invalid refresh token",
			})
			return
		}

		responseResourceInternalServerError(w, err, models.RefreshResponse{
			Success: false,
			Message: "could not refresh tokens",
		})
		return
	}

	responseResourceOK(w, models.RefreshResponse{
		Success:      true,
		Message:      "tokens refreshed",
		AccessToken:  tokenSet.AccessToken,
		RefreshToken: tokenSet.RefreshToken,
	})
}

func (h *AuthRoute) Logout(w http.ResponseWriter, r *http.Request) {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		responseResourceUnauthorized(w, models.LogoutResponse{
			Success: false,
			Message: "missing authorization header",
		})
		return
	}

	headerParts := strings.SplitN(authorizationHeader, " ", 2)
	if len(headerParts) != 2 || !strings.EqualFold(headerParts[0], "Bearer") || headerParts[1] == "" {
		responseResourceUnauthorized(w, models.LogoutResponse{
			Success: false,
			Message: "invalid authorization header",
		})
		return
	}
	token := headerParts[1]

	tokenID, err := h.tokenGenerator.GetTokenIDFromTokenAllowExpired(token)
	if err != nil {
		responseResourceUnauthorized(w, models.LogoutResponse{
			Success: false,
			Message: "invalid token",
		})
		return
	}

	err = h.db.DisableRefreshToken(tokenID, token)
	if err == nil || errors.Is(err, db.TokenNotExistsError) {
		responseResourceOK(w, models.LogoutResponse{
			Success: true,
			Message: "logout successful",
		})
		return
	}

	if errors.Is(err, db.TokenHashMismatchError) {
		responseResourceUnauthorized(w, models.LogoutResponse{
			Success: false,
			Message: "invalid token",
		})
		return
	}

	responseResourceInternalServerError(w, err, models.LogoutResponse{
		Success: false,
		Message: "could not disable refresh token",
	})
}
