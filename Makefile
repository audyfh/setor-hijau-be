.PHONY: up down test lint sqlc migrate-new migrate-up migrate-down check

DATABASE_URL ?= postgres://postgres:postgrespassword@localhost:5432/setor_hijau?sslmode=disable

up:
	docker compose up -d

down:
	docker compose down

test:
	go test -v -race ./...

lint:
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not in PATH, ran go vet"

sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0 generate

migrate-up:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate up

migrate-down:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate down 1

migrate-new:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-new name=<migration_name>"; exit 1; fi
	@ts=$$(date +%Y%m%d%H%M%S); \
	touch "db/migrations/$${ts}_$(name).up.sql"; \
	touch "db/migrations/$${ts}_$(name).down.sql"; \
	echo "Created db/migrations/$${ts}_$(name).up.sql and down.sql"

check: lint test
	@echo "Checking sqlc diff..."
	@git diff --exit-code internal/platform/db/sqlc || (echo "Error: Uncommitted changes in generated sqlc files" && exit 1)
	@echo "All checks passed!"
