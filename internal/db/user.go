package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

func (dbService *DBService) CreateUser(registerRequest *models.RegisterRequest) (*models.User, *models.SignedTokenSet, error) {
	if dbService.conn == nil {
		return nil, nil, DatabaseConnectionError
	}

	query := `
		INSERT INTO users (email, password) VALUES ($1, $2)
		RETURNING id, email, created_at, updated_at
	`

	var user models.User

	tx, err := dbService.conn.Begin(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	err = tx.QueryRow(
		context.Background(),
		query,
		registerRequest.Email,
		registerRequest.Password,
	).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, nil, UserExistsError
		}
		return nil, nil, fmt.Errorf("creating user: %w", err)
	}

	tokenSet, err := dbService.createTokenSetTx(tx, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("creating token set: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("committing transaction: %w", err)
	}

	return &user, tokenSet, nil
}

func (dbService *DBService) ValidateUser(email string, password string) (*models.User, *models.SignedTokenSet, error) {
	if dbService.conn == nil {
		return nil, nil, DatabaseConnectionError
	}

	user, err := dbService.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, UserNotFoundError) {
			return nil, nil, UserNotFoundError
		}
		return nil, nil, fmt.Errorf("getting user by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, nil, InvalidCredentialsError
	}

	tokenSet, err := dbService.CreateTokenSet(user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("creating token set: %w", err)
	}

	return user, tokenSet, nil
}

func (dbService *DBService) GetUserByEmail(email string) (*models.User, error) {
	if dbService.conn == nil {
		return nil, DatabaseConnectionError
	}

	query := `
		SELECT id, email, password, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := dbService.conn.QueryRow(context.Background(), query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, UserNotFoundError
		}
		return nil, fmt.Errorf("getting user by email: %w", err)
	}
	return &user, nil
}
