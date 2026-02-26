# digital-twin-community backend Makefile

.PHONY: help dev build test test-unit test-integration test-cover lint fmt vet docker-build migrate seed clean

BINARY_NAME=server
BUILD_DIR=./bin
MAIN_PKG=./cmd/server
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-w -s -X main.version=$(VERSION)"

# Colours
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)

## help: Show this help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## dev: Run the server in development mode (hot reload with air)
dev:
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air -c .air.toml

## build: Build the binary
build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PKG)
	@echo "$(GREEN)Built: $(BUILD_DIR)/$(BINARY_NAME)$(RESET)"

## test: Run all tests
test: test-unit test-integration

## test-unit: Run unit tests with race detector
test-unit:
	go test -v -race -run Unit ./internal/...

## test-integration: Run integration tests (requires Docker)
test-integration:
	go test -v -tags=integration -timeout=10m ./internal/...

## test-cover: Run tests with coverage report
test-cover:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1
	@echo "$(GREEN)Coverage report: coverage.html$(RESET)"

## lint: Run golangci-lint
lint:
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.57.0
	golangci-lint run --timeout=5m ./...

## fmt: Format Go code
fmt:
	gofmt -w -s .
	goimports -w .

## vet: Run go vet
vet:
	go vet ./...

## migrate-up: Run database migrations
migrate-up:
	@which migrate > /dev/null || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path ./db/migrations -database "$(DATABASE_URL)" up

## migrate-down: Rollback last migration
migrate-down:
	migrate -path ./db/migrations -database "$(DATABASE_URL)" down 1

## migrate-status: Show migration status
migrate-status:
	migrate -path ./db/migrations -database "$(DATABASE_URL)" version

## docker-build: Build Docker image
docker-build:
	docker build -t digital-twin-backend:$(VERSION) -f Dockerfile .

## docker-run: Run backend with Docker Compose
docker-run:
	cd .. && docker compose up -d

## docker-stop: Stop Docker Compose services
docker-stop:
	cd .. && docker compose down

## sqlc: Generate type-safe SQL code
sqlc:
	@which sqlc > /dev/null || go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	sqlc generate

## seed: Run database seed (local dev only)
seed:
	go run ./cmd/seed/...

## tidy: Tidy go modules
tidy:
	go mod tidy
	go mod verify

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
