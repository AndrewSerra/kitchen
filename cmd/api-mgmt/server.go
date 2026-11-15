package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AndrewSerra/kitchen/internal/db"
	handler "github.com/AndrewSerra/kitchen/internal/http"
	"github.com/AndrewSerra/kitchen/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

type EnvType string

const (
	EnvDevelopment EnvType = "development"
	EnvProduction  EnvType = "production"
)

type Config struct {
	Env   EnvType
	Port  string
	DbURL string
}

func collectEnvVariables() (Config, error) {
	env := os.Getenv("ENV")
	if env == "" {
		env = string(EnvProduction)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Printf("DB_URL is not set in the environment variables\n")
		return Config{}, errors.New("DB_URL is not set in the environment variables")
	}

	return Config{Env: EnvType(env), Port: port, DbURL: dbURL}, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v\n", err)
		os.Exit(1)
	}

	config, err := collectEnvVariables()
	if err != nil {
		log.Printf("Error collecting environment variables: %v\n", err)
		os.Exit(1)
	}

	logger.Init(string(config.Env))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	dbService, err := db.NewDBService(config.DbURL)

	if err != nil {
		log.Printf("Error creating database service: %v\n", err)
		os.Exit(1)
	}

	kitchenMiddleware := handler.NewKitchenMiddleware(dbService)
	if err != nil {
		log.Printf("Error creating kitchen middleware: %v\n", err)
		os.Exit(1)
	}

	autHandlers := handler.NewAuthRoute(dbService)
	streamsHandlers := handler.NewStreamsRoute(dbService, kitchenMiddleware)

	httpHandlers := map[string]handler.HttpRoutes{
		"auth":    autHandlers,
		"streams": streamsHandlers,
	}

	for path, handler := range httpHandlers {
		r.Mount(fmt.Sprintf("/api/%s", path), handler.Routes())
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.Port),
		Handler: r,
	}

	err = dbService.InitDB()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		fmt.Println("Closing database connection...")
		err = dbService.Close()
		if err != nil {
			fmt.Printf("Error closing database connection: %v\n", err)
		}
		os.Exit(1)
	}

	err = dbService.RunMigrations()
	if err != nil {
		fmt.Printf("Error running migrations: %v\n", err)
		fmt.Println("Closing database connection...")
		err = dbService.Close()
		if err != nil {
			fmt.Printf("Error closing database connection: %v\n", err)
		}
		os.Exit(1)
	}

	go func() {
		fmt.Println("Starting server... Listening on port " + config.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Error starting server: %v\n", err)
			stop()
		}
	}()

	<-ctx.Done()
	stop()

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server: %v\n", err)
	}

	fmt.Println("Closing database connection...")
	err = dbService.Close()
	if err != nil {
		fmt.Printf("Error closing database connection: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server shut down gracefully")
}
