package db

import (
	"context"
	"fmt"
	"time"

	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/AndrewSerra/kitchen/internal/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (dbService *DBService) CreateRefreshTokenTx(tx pgx.Tx, signedRefreshToken string, refreshToken jwt.Token, userID int) (*models.RefreshTokenRecord, error) {
	if dbService.conn == nil {
		return nil, DatabaseConnectionError
	}

	err := dbService.DisableActiveRefreshTokensTx(tx, userID)
	if err != nil {
		return nil, fmt.Errorf("disabling active refresh tokens: %w", err)
	}

	query := `
		INSERT INTO refresh_tokens (token, token_id, user_id) VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	refreshTokenHash, err := utils.HashToken(signedRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("hashing refresh token: %w", err)
	}

	var refreshTokenRecord int
	var createdAt time.Time
	var updatedAt time.Time
	err = tx.QueryRow(context.Background(), query, refreshTokenHash, refreshToken.Claims.(jwt.MapClaims)["jti"], userID).Scan(&refreshTokenRecord, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating refresh token: %w", err)
	}

	return &models.RefreshTokenRecord{
		ID:        refreshTokenRecord,
		Token:     signedRefreshToken,
		TokenID:   refreshToken.Claims.(jwt.MapClaims)["jti"].(uuid.UUID),
		UserID:    userID,
		Status:    models.TokenStatusActive,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (dbService *DBService) CreateTokenSet(userID int) (*models.SignedTokenSet, error) {
	if dbService.conn == nil {
		return nil, DatabaseConnectionError
	}

	tx, err := dbService.conn.Begin(context.Background())
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	tokenSet, err := dbService.CreateTokenSetTx(tx, userID)
	if err != nil {
		return nil, fmt.Errorf("creating token set: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return tokenSet, nil
}

func (dbService *DBService) CreateTokenSetTx(tx pgx.Tx, userID int) (*models.SignedTokenSet, error) {
	if dbService.conn == nil {
		return nil, DatabaseConnectionError
	}

	tokenId := uuid.NewString()
	tokenSet, err := dbService.tokenGenerator.GenerateTokenSet(userID, tokenId)
	if err != nil {
		return nil, fmt.Errorf("generating token set: %w", err)
	}

	refreshToken, err := dbService.tokenGenerator.SignToken(tokenSet.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("signing refresh token: %w", err)
	}

	accessToken, err := dbService.tokenGenerator.SignToken(tokenSet.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	_, err = dbService.CreateRefreshTokenTx(tx, refreshToken, *tokenSet.RefreshToken, userID)
	if err != nil {
		return nil, fmt.Errorf("creating refresh token: %w", err)
	}

	return &models.SignedTokenSet{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, nil
}

func (dbService *DBService) GetRefreshToken(token string) (*models.RefreshTokenRecord, error) {
	return nil, nil
}

func (dbService *DBService) DisableActiveRefreshTokens(userID int) error {
	if dbService.conn == nil {
		return DatabaseConnectionError
	}

	query := `
		UPDATE refresh_tokens SET status = $1 WHERE user_id = $2 AND status = $3
	`

	_, err := dbService.conn.Exec(context.Background(), query, models.TokenStatusInactive, userID, models.TokenStatusActive)
	if err != nil {
		return fmt.Errorf("disabling active refresh tokens: %w", err)
	}

	return nil
}

func (dbService *DBService) DisableActiveRefreshTokensTx(tx pgx.Tx, userID int) error {
	if dbService.conn == nil {
		return DatabaseConnectionError
	}

	query := `
		UPDATE refresh_tokens SET status = $1 WHERE user_id = $2 AND status = $3
	`

	_, err := tx.Exec(context.Background(), query, models.TokenStatusInactive, userID, models.TokenStatusActive)
	if err != nil {
		return fmt.Errorf("disabling active refresh tokens: %w", err)
	}

	return nil
}
