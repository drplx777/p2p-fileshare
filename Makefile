.PHONY: tidy test build run migrate-up migrate-down

tidy:
	go mod tidy

test:
	go test ./... -count=1

build:
	go build -o bin/p2p-fileshare-api ./cmd/api

run:
	go run ./cmd/api

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

