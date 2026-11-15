package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AndrewSerra/kitchen/internal/logger"
	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/AndrewSerra/kitchen/internal/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (dbService *DBService) createRefreshTokenTx(tx pgx.Tx, signedRefreshToken string, refreshToken jwt.Token, userID int) (*models.RefreshTokenRecord, error) {
	if dbService.conn == nil {
		return nil, DatabaseConnectionError
	}

	err := dbService.disableActiveRefreshTokensTx(tx, userID)
	if err != nil {
		return nil, fmt.Errorf("disabling active refresh tokens: %w", err)
	}

	query := `
		INSERT INTO refresh_tokens (token, token_id, user_id) VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	refreshTokenHash := utils.HashToken(signedRefreshToken)

	jtiClaim := refreshToken.Claims.(jwt.MapClaims)["jti"]
	tokenID, err := dbService.tokenGenerator.ParseTokenID(jtiClaim)
	if err != nil {
		return nil, err
	}

	var refreshTokenRecord int
	var createdAt time.Time
	var updatedAt time.Time
	err = tx.QueryRow(context.Background(), query, refreshTokenHash, tokenID, userID).Scan(&refreshTokenRecord, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating refresh token: %w", err)
	}

	return &models.RefreshTokenRecord{
		ID:        refreshTokenRecord,
		Token:     refreshTokenHash,
		TokenID:   tokenID,
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

	tokenSet, err := dbService.createTokenSetTx(tx, userID)
	if err != nil {
		return nil, fmt.Errorf("creating token set: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return tokenSet, nil
}

func (dbService *DBService) createTokenSetTx(tx pgx.Tx, userID int) (*models.SignedTokenSet, error) {
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

	_, err = dbService.createRefreshTokenTx(tx, refreshToken, *tokenSet.RefreshToken, userID)
	if err != nil {
		return nil, fmt.Errorf("creating refresh token: %w", err)
	}

	return &models.SignedTokenSet{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, nil
}

func (dbService *DBService) RefreshTokens(refreshToken string) (*models.SignedTokenSet, error) {
	if dbService.conn == nil {
		return nil, DatabaseConnectionError
	}

	parsedToken, err := dbService.tokenGenerator.VerifyToken(refreshToken)
	if err != nil {
		logger.Warn("refresh token verification failed", err)
		return nil, err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		logger.Warn("refresh token claim parsing failed", "err", utils.TokenClaimParsingError)
		return nil, utils.TokenClaimParsingError
	}

	tokenID, err := dbService.tokenGenerator.ParseTokenID(claims["jti"])
	if err != nil {
		logger.Warn("refresh token jti mismatch", "err", err)
		return nil, err
	}

	refreshTokenRecord, err := dbService.GetRefreshToken(tokenID)
	if err != nil {
		if errors.Is(err, TokenNotExistsError) {
			logger.Warn("refresh token not found", "err", err)
			return nil, err
		}
		logger.Error("getting refresh token failed", "err", err)
		return nil, err
	}

	if refreshTokenRecord.Status != models.TokenStatusActive {
		logger.Warn("refresh token inactive", "err", utils.TokenInactiveError, "user_id", refreshTokenRecord.UserID)
		return nil, utils.TokenInactiveError
	}

	if !utils.CompareTokenToHash(refreshToken, refreshTokenRecord.Token) {
		logger.Warn("refresh token hash mismatch", "err", TokenHashMismatchError, "user_id", refreshTokenRecord.UserID)
		return nil, TokenHashMismatchError
	}

	tx, err := dbService.conn.Begin(context.Background())
	if err != nil {
		logger.Error("beginning refresh token transaction failed", "err", err)
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	err = dbService.disableActiveRefreshTokensTx(tx, refreshTokenRecord.UserID)
	if err != nil {
		logger.Error("disabling active refresh tokens failed", "err", err, "user_id", refreshTokenRecord.UserID)
		return nil, fmt.Errorf("disabling active refresh tokens: %w", err)
	}

	tokenSet, err := dbService.createTokenSetTx(tx, refreshTokenRecord.UserID)
	if err != nil {
		logger.Error("creating token set failed", "err", err, "user_id", refreshTokenRecord.UserID)
		return nil, fmt.Errorf("creating token set: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		logger.Error("committing refresh token transaction failed", "err", err, "user_id", refreshTokenRecord.UserID)
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	logger.Info("refresh tokens issued", "user_id", refreshTokenRecord.UserID)
	return tokenSet, nil
}

func (dbService *DBService) GetRefreshToken(tokenID uuid.UUID) (*models.RefreshTokenRecord, error) {
	if dbService.conn == nil {
		return nil, DatabaseConnectionError
	}

	query := `
		SELECT id, token, token_id, user_id, status, created_at, updated_at
		FROM refresh_tokens WHERE token_id = $1
	`

	var refreshTokenRecord models.RefreshTokenRecord
	if err := dbService.conn.QueryRow(context.Background(), query, tokenID).
		Scan(&refreshTokenRecord.ID,
			&refreshTokenRecord.Token,
			&refreshTokenRecord.TokenID,
			&refreshTokenRecord.UserID,
			&refreshTokenRecord.Status,
			&refreshTokenRecord.CreatedAt,
			&refreshTokenRecord.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, TokenNotExistsError
		}
		return nil, fmt.Errorf("getting refresh token: %w", err)
	}

	return &refreshTokenRecord, nil
}

func (dbService *DBService) DisableActiveRefreshTokens(userID int) error {
	if dbService.conn == nil {
		return DatabaseConnectionError
	}

	tx, err := dbService.conn.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	err = dbService.disableActiveRefreshTokensTx(tx, userID)
	if err != nil {
		return fmt.Errorf("disabling active refresh tokens: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

func (dbService *DBService) disableActiveRefreshTokensTx(tx pgx.Tx, userID int) error {
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

func (dbService *DBService) DisableRefreshToken(tokenID uuid.UUID, token string) error {
	if dbService.conn == nil {
		return DatabaseConnectionError
	}

	tx, err := dbService.conn.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	err = dbService.disableRefreshTokenTx(tx, tokenID, token)
	if err != nil {
		return fmt.Errorf("disabling refresh token transaction: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("committing refresh token transaction: %w", err)
	}

	return nil
}

func (dbService *DBService) disableRefreshTokenTx(tx pgx.Tx, tokenID uuid.UUID, token string) error {
	if dbService.conn == nil {
		return DatabaseConnectionError
	}

	query := `
		UPDATE refresh_tokens SET status = $1 WHERE token_id = $2 AND token = $3
	`

	rows, err := tx.Exec(context.Background(), query, models.TokenStatusInactive, tokenID, utils.HashToken(token))
	if err != nil {
		return fmt.Errorf("disabling refresh token: %w", err)
	}

	if rows.RowsAffected() == 0 {
		return TokenNotExistsError
	}

	return nil
}
