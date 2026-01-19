.PHONY: help fmt lint test cover build run clean security migrate-up migrate-down migrate-create migrate-force migrator-build migrator-run generate generate-install deps-outdated depguard

# Default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := hateblog-backend
CMD_DIR := ./cmd/app
BUILD_DIR := ./bin
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
	@$(MAKE) depguard
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
	gosec -quiet -exclude-dir=internal/infra/handler/openapi ./...
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
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -extldflags "-static"' \
		-o $(BUILD_DIR)/fetcher \
		./cmd/fetcher
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -extldflags "-static"' \
		-o $(BUILD_DIR)/updater \
		./cmd/updater
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -extldflags "-static"' \
		-o $(BUILD_DIR)/admin \
		./cmd/admin
	go build -o $(BUILD_DIR)/migrator ./cmd/migrator
	@echo "✓ Binaries built: $(BUILD_DIR)/$(APP_NAME), $(BUILD_DIR)/fetcher, $(BUILD_DIR)/updater, $(BUILD_DIR)/admin, $(BUILD_DIR)/migrator"

## run: Run the application
run:
	@echo "==> Running application..."
	go run $(CMD_DIR)/main.go

## deps-outdated: List outdated Go module dependencies
deps-outdated:
	@echo "==> Checking outdated dependencies..."
	go run ./cmd/tools/depsoutdated
	@echo "✓ Dependency check complete"

## depguard: Enforce dependency boundaries
depguard:
	@echo "==> Checking dependency boundaries..."
	CGO_ENABLED=0 go run github.com/OpenPeeDeeP/depguard/cmd/depguard@latest -c depguard.json ./...
	@echo "✓ Dependency boundaries respected"

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

## migrator-build: Build the data migrator tool
migrator-build:
	@echo "==> Building migrator..."
	go build -o $(BUILD_DIR)/migrator ./cmd/migrator
	@echo "✓ Migrator built: $(BUILD_DIR)/migrator"

## migrator-run: Run the data migration from MySQL to PostgreSQL
migrator-run: migrator-build
	@echo "==> Running data migration..."
	@if [ ! -f "$(BUILD_DIR)/migrator" ]; then \
		echo "Error: migrator binary not found. Run 'make migrator-build' first"; \
		exit 1; \
	fi
	$(BUILD_DIR)/migrator
	@echo "✓ Migration complete"

## ci: Run all CI checks (fmt, lint, depguard, test, security)
ci: fmt lint depguard test security
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
	go install github.com/cweill/gotests/gotests@latest
	go install github.com/josharian/impl@latest
	go install github.com/haya14busa/goplay/cmd/goplay@latest
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	go install github.com/OpenPeeDeeP/depguard/cmd/depguard@latest
	@echo "✓ Development environment ready"
