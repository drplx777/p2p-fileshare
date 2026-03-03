.PHONY: tidy test build run migrate-up migrate-down docker-build docker-up docker-down migrate-docker

COMPOSE = docker compose --env-file .env

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

# Docker: сборка и запуск (локально и на сервере)
docker-build:
	$(COMPOSE) build api

docker-up:
	$(COMPOSE) up -d api db

docker-down:
	$(COMPOSE) down

# Миграции через контейнер
migrate-docker:
	$(COMPOSE) run --rm migrate

