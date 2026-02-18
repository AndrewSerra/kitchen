include .env
export

BINARY := bin/server
MIGRATE := $(shell which migrate 2>/dev/null || echo "migrate")

.PHONY: run build test migrate-up migrate-down lint

run:
	go run ./cmd/server

build:
	go build -o $(BINARY) ./cmd/server

test:
	go test ./...

migrate-up:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down 1

lint:
	golangci-lint run ./...
