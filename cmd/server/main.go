package main

import (
	"log"

	"github.com/andrewserra/kitchen/internal/config"
	"github.com/andrewserra/kitchen/internal/db"
	"github.com/andrewserra/kitchen/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	sqlDB, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer sqlDB.Close()

	srv := server.New(cfg, sqlDB)
	if err := srv.Run(cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
