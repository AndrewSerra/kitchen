package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenStatus string

const (
	TokenStatusActive   TokenStatus = "active"
	TokenStatusInactive TokenStatus = "inactive"
)

type RefreshTokenRecord struct {
	ID        int         `json:"id"`
	Token     string      `json:"token"`
	TokenID   uuid.UUID   `json:"token_id"`
	UserID    int         `json:"user_id"`
	Status    TokenStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type RefreshToken = jwt.Token
type AccessToken = jwt.Token

type TokenSet struct {
	RefreshToken *RefreshToken
	AccessToken  *AccessToken
}

type SignedTokenSet struct {
	RefreshToken string
	AccessToken  string
}
