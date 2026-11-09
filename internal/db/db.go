package db

import (
	"context"
	"errors"
	"strings"

	"github.com/AndrewSerra/kitchen/internal/utils"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
)

type DBService struct {
	url            string
	conn           *pgx.Conn
	tokenGenerator *utils.TokenGenerator
}

func NewDBService(dbURL string) (*DBService, error) {
	if dbURL == "" {
		return nil, errors.New("DB_URL is not set")
	}

	dbService := &DBService{
		url:            dbURL,
		tokenGenerator: utils.NewTokenGenerator(),
	}

	return dbService, nil
}

func (dbService *DBService) InitDB() error {
	var err error

	dbService.conn, err = pgx.Connect(context.Background(), dbService.url)
	if err != nil {
		return errors.New("Error connecting to database " + err.Error())
	}
	return nil
}

func (dbService *DBService) Close() error {
	if dbService.conn != nil {
		return dbService.conn.Close(context.Background())
	}
	return nil
}

func (dbService *DBService) RunMigrations() error {
	if dbService.conn == nil {
		return errors.New("Database connection not established")
	}

	m, err := migrate.New(
		"file:///Users/andrewserra/GithubProjects/kitchen/database/migrations",
		toPgx5MigrateURL(dbService.url),
	)
	if err != nil {
		return errors.New("Error creating migration " + err.Error())
	}
	defer m.Close()

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return errors.New("Error running migrations " + err.Error())
	}
	return nil
}

// migrate's pgx/v5 driver registers as "pgx5"; pgx.Connect still wants postgres://.
func toPgx5MigrateURL(dbURL string) string {
	switch {
	case strings.HasPrefix(dbURL, "postgres://"):
		return "pgx5://" + strings.TrimPrefix(dbURL, "postgres://")
	case strings.HasPrefix(dbURL, "postgresql://"):
		return "pgx5://" + strings.TrimPrefix(dbURL, "postgresql://")
	default:
		return dbURL
	}
}
