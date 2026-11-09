package utils

import (
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/AndrewSerra/kitchen/internal/models"
)

// REQUEST BODY VALIDATION
// ========================================================
func ValidateRegisterRequest(request *models.RegisterRequest) error {
	if request.Email == "" {
		return errors.New("email is required")
	}
	if request.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func ValidateLoginRequest(request *models.LoginRequest) error {
	if request.Email == "" {
		return errors.New("email is required")
	}
	if request.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func ValidateAccessTokenPayload(payload AccessTokenPayload) error {
	if payload.UserID <= 0 {
		return errors.New("user_id is required")
	}
	return nil
}

func ValidateRefreshTokenPayload(payload RefreshTokenPayload) error {
	if uuid.MustParse(payload.TokenID.String()) == uuid.Nil {
		return errors.New("token_id is required")
	}
	return nil
}

// CONFIG VALIDATION
// ========================================================
func ValidateServerURL(url string) (string, error) {
	if url == "" {
		return "", errors.New("url is required")
	}

	urlSections := strings.Split(url, "://")
	protocol := urlSections[0]
	url = urlSections[1]

	if len(urlSections) != 2 {
		return "", errors.New("invalid url")
	}

	if protocol != "http" && protocol != "https" {
		return "", errors.New("invalid protocol")
	}

	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}

	return protocol + "://" + url, nil
}
