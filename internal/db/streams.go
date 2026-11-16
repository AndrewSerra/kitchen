package db

import (
	"context"
	"errors"

	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (dbService *DBService) CreateStream(streamName string, userID int) (*models.Stream, error) {
	if dbService.conn == nil {
		return nil, DatabaseConnectionError
	}

	query := `
	INSERT INTO streams (name, user_id)
	VALUES ($1, $2)
	RETURNING id, created_at, updated_at
	`
	stream := models.Stream{
		Name:   streamName,
		UserID: userID,
	}
	if err := dbService.conn.QueryRow(context.Background(), query, streamName, userID).Scan(&stream.ID, &stream.CreatedAt, &stream.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, StreamExistsError
		}
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			return nil, StreamUserNotFoundError
		}
		return nil, err
	}
	return &stream, nil
}

func (dbService *DBService) GetStream(userID int, streamID int) (*models.Stream, error) {
	if dbService.conn == nil {
		return nil, DatabaseConnectionError
	}

	query := `
	SELECT id, name, user_id, created_at, updated_at
	FROM streams
	WHERE id = $1 AND user_id = $2
	`

	var stream models.Stream
	if err := dbService.conn.QueryRow(context.Background(), query, streamID, userID).Scan(&stream.ID, &stream.Name, &stream.UserID, &stream.CreatedAt, &stream.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, StreamNotFoundError
		}
		return nil, err
	}
	return &stream, nil
}

func (dbService *DBService) GetAllStreamsForUser(userID int) ([]*models.Stream, error) {
	if dbService.conn == nil {
		return nil, DatabaseConnectionError
	}

	query := `
	SELECT id, name, user_id, created_at, updated_at
	FROM streams
	WHERE user_id = $1
	`

	streams := make([]*models.Stream, 0)
	rows, err := dbService.conn.Query(context.Background(), query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stream models.Stream
		if err := rows.Scan(&stream.ID, &stream.Name, &stream.UserID, &stream.CreatedAt, &stream.UpdatedAt); err != nil {
			return nil, err
		}
		streams = append(streams, &stream)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return streams, nil
}

func (dbService *DBService) DeleteStream(userID int, streamID int) error {
	if dbService.conn == nil {
		return DatabaseConnectionError
	}

	query := `
	DELETE FROM streams
	WHERE id = $1 AND user_id = $2
	`
	result, err := dbService.conn.Exec(context.Background(), query, streamID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return StreamNotFoundError
	}

	return nil
}
