.PHONY: help fmt lint test cover build run clean docker-build docker-run security migrate-up migrate-down migrate-create generate generate-install deps-outdated

# Default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := hateblog-backend
CMD_DIR := ./cmd/app
BUILD_DIR := ./bin
DOCKER_IMAGE := $(APP_NAME)
DOCKER_TAG := latest
MIGRATE_DIR := ./migrations
OPENAPI_SPEC := ./openapi.yaml
OPENAPI_CONFIG := ./oapi-codegen.yaml
OPENAPI_OUTPUT := ./internal/infra/handler/openapi

## help: Display this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/##//' | column -t -s ':'

## fmt: Format Go code with gofmt and goimports
fmt:
	@echo "==> Formatting code..."
	gofmt -s -w .
	goimports -w -local hateblog .
	@echo "✓ Code formatted"

## lint: Run golangci-lint
lint:
	@echo "==> Running linter..."
	golangci-lint run --config .golangci.yml
	@echo "✓ Linting complete"

## test: Run tests with race detector
test:
	@echo "==> Running tests..."
	go test ./... -v -race -shuffle=on -timeout=5m
	@echo "✓ Tests complete"

## test-short: Run tests without race detector (faster)
test-short:
	@echo "==> Running tests (short)..."
	go test ./... -v -short -timeout=2m
	@echo "✓ Tests complete"

## cover: Run tests with coverage
cover:
	@echo "==> Running tests with coverage..."
	go test ./... -coverprofile=cover.out -covermode=atomic
	go tool cover -html=cover.out -o cover.html
	@echo "✓ Coverage report generated: cover.html"

## security: Run security checks (gosec + govulncheck)
security:
	@echo "==> Running security checks..."
	@echo "--> Running gosec..."
	gosec -quiet ./...
	@echo "--> Running govulncheck..."
	govulncheck ./...
	@echo "✓ Security checks complete"

## build: Build the application binary
build:
	@echo "==> Building application..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -extldflags "-static"' \
		-o $(BUILD_DIR)/$(APP_NAME) \
		$(CMD_DIR)
	@echo "✓ Binary built: $(BUILD_DIR)/$(APP_NAME)"

## run: Run the application
run:
	@echo "==> Running application..."
	go run $(CMD_DIR)/main.go

## deps-outdated: List outdated Go module dependencies
deps-outdated:
	@echo "==> Checking outdated dependencies..."
	go run ./cmd/tools/depsoutdated
	@echo "✓ Dependency check complete"

## clean: Clean build artifacts and caches
clean:
	@echo "==> Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f cover.out cover.html
	go clean -cache -testcache -modcache
	@echo "✓ Cleaned"

## deps: Download and tidy Go dependencies
deps:
	@echo "==> Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies updated"

## generate: Generate code from OpenAPI specification
generate:
	@echo "==> Generating code from OpenAPI spec..."
	@mkdir -p $(OPENAPI_OUTPUT)
	oapi-codegen -config $(OPENAPI_CONFIG) $(OPENAPI_SPEC)
	@echo "✓ Code generation complete"

## generate-install: Install oapi-codegen tool
generate-install:
	@echo "==> Installing oapi-codegen..."
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "✓ oapi-codegen installed"

## docker-build: Build Docker image
docker-build:
	@echo "==> Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "✓ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

## docker-run: Run Docker container locally
docker-run:
	@echo "==> Running Docker container..."
	docker run --rm -p 8080:8080 \
		--env-file .env \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

## compose-up: Start all services with Docker Compose
compose-up:
	@echo "==> Starting services..."
	docker compose up -d
	@echo "✓ Services started"

## compose-down: Stop all services
compose-down:
	@echo "==> Stopping services..."
	docker compose down
	@echo "✓ Services stopped"

## compose-logs: Show logs from all services
compose-logs:
	docker compose logs -f

## migrate-create: Create a new migration file (usage: make migrate-create name=create_users_table)
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: name parameter is required"; \
		echo "Usage: make migrate-create name=create_users_table"; \
		exit 1; \
	fi
	@echo "==> Creating migration: $(name)..."
	migrate create -ext sql -dir $(MIGRATE_DIR) -seq $(name)
	@echo "✓ Migration created"

## migrate-up: Run all pending migrations
migrate-up:
	@echo "==> Running migrations..."
	@if [ -z "$(DB_URL)" ]; then \
		DB_URL="postgresql://hateblog:changeme@localhost:5432/hateblog?sslmode=disable"; \
	fi; \
	migrate -path $(MIGRATE_DIR) -database "$$DB_URL" up
	@echo "✓ Migrations applied"

## migrate-down: Rollback the last migration
migrate-down:
	@echo "==> Rolling back migration..."
	@if [ -z "$(DB_URL)" ]; then \
		DB_URL="postgresql://hateblog:changeme@localhost:5432/hateblog?sslmode=disable"; \
	fi; \
	migrate -path $(MIGRATE_DIR) -database "$$DB_URL" down 1
	@echo "✓ Migration rolled back"

## migrate-force: Force migration version (usage: make migrate-force version=1)
migrate-force:
	@if [ -z "$(version)" ]; then \
		echo "Error: version parameter is required"; \
		echo "Usage: make migrate-force version=1"; \
		exit 1; \
	fi
	@echo "==> Forcing migration version to $(version)..."
	@if [ -z "$(DB_URL)" ]; then \
		DB_URL="postgresql://hateblog:changeme@localhost:5432/hateblog?sslmode=disable"; \
	fi; \
	migrate -path $(MIGRATE_DIR) -database "$$DB_URL" force $(version)
	@echo "✓ Migration version forced"

## ci: Run all CI checks (fmt, lint, test, security)
ci: fmt lint test security
	@echo "✓ All CI checks passed"

## dev: Setup development environment
dev: deps
	@echo "==> Setting up development environment..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✓ Created .env file from .env.example"; \
	fi
	@echo "==> Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "✓ Development environment ready"
