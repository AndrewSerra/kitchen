package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/AndrewSerra/kitchen/internal/db"
	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/AndrewSerra/kitchen/internal/utils"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthRoute struct {
	db *db.DBService
}

func NewAuthRoute(db *db.DBService) *AuthRoute {
	return &AuthRoute{
		db: db,
	}
}

func (h *AuthRoute) Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Post("/login", h.Login)
	router.Post("/register", h.Register)
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
