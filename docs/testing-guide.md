# Testing Guide

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Prerequisites](#prerequisites)
4. [Running Tests](#running-tests)
5. [Test Organization](#test-organization)
6. [Writing Tests](#writing-tests)
7. [Test Data Management](#test-data-management)
8. [CI/CD Integration](#cicd-integration)
9. [Troubleshooting](#troubleshooting)
10. [Best Practices](#best-practices)

---

## Overview

This project follows an **API-centric testing strategy** with three layers:

```
        ┌─────────────┐
        │  E2E Tests  │  Few (critical flows only)
        └─────────────┘
       ┌───────────────┐
       │  API Tests    │  Primary focus: endpoint behavior
       └───────────────┘
      ┌─────────────────┐
      │  Unit Tests     │  Minimal: core domain logic only
      └─────────────────┘
```

### Testing Philosophy

- **API Tests (Primary)**: Verify endpoints work correctly with real infrastructure
- **Unit Tests (Minimal)**: Test critical business logic in isolation
- **E2E Tests (Few)**: Validate main user scenarios end-to-end
- **Real Infrastructure**: Use testcontainers for PostgreSQL/Redis instead of mocks
- **Test Isolation**: Each test is independent and can run in parallel

---

## Quick Start

```bash
# Run all tests
make test

# Run tests with coverage report
make cover

# Run tests quickly (without race detector)
make test-short

# Run specific package tests
go test ./internal/domain/... -v

# Run specific test
go test ./internal/infra/handler -run TestCreateUser -v

# Run tests in watch mode (requires entr)
find . -name "*.go" | entr -c go test ./...
```

---

## Prerequisites

### Required Tools

1. **Go 1.25+**
   ```bash
   go version
   ```

2. **Docker** (for testcontainers)
   ```bash
   docker --version
   docker compose version
   ```

   Make sure Docker daemon is running:
   ```bash
   docker ps
   ```

3. **Make** (optional but recommended)
   ```bash
   make --version
   ```

### Install Development Tools

```bash
make dev
```

This installs:
- golangci-lint (linting)
- goimports (code formatting)
- govulncheck (security scanning)
- gosec (security analysis)
- migrate (database migrations)
- oapi-codegen (OpenAPI code generation)

### Environment Setup

1. Copy environment file:
   ```bash
   cp .env.example .env
   ```

2. Ensure Docker is running (required for testcontainers)

---

## Running Tests

### All Tests

```bash
# Standard test run with race detector
make test

# Equivalent to:
go test ./... -v -race -shuffle=on -timeout=5m
```

**Flags explained:**
- `-v`: Verbose output (shows test names)
- `-race`: Race condition detector
- `-shuffle=on`: Randomize test execution order
- `-timeout=5m`: Kill tests that run longer than 5 minutes

### Quick Tests (No Race Detector)

```bash
make test-short

# Equivalent to:
go test ./... -v -short -timeout=2m
```

Use this for faster feedback during development.

### Coverage Reports

```bash
make cover
```

This generates:
- `cover.out`: Coverage data file
- `cover.html`: Interactive HTML coverage report

Open the report:
```bash
open cover.html  # macOS
xdg-open cover.html  # Linux
```

### Run Specific Tests

```bash
# Test specific package
go test ./internal/domain/user -v

# Test specific function
go test ./internal/infra/handler -run TestCreateUser_Success -v

# Test with pattern matching
go test ./... -run "Test.*User.*" -v

# Run only unit tests (exclude integration tests)
go test -short ./...

# Run only integration tests
go test -run Integration ./...
```

### Parallel Test Execution

```bash
# Run tests in parallel (default: GOMAXPROCS)
go test ./... -parallel 4

# Run tests sequentially
go test ./... -parallel 1
```

### Verbose Output

```bash
# Show all test output
go test ./... -v

# Show only failures
go test ./...

# Show test names as they run
go test ./... -v | grep -E "^(PASS|FAIL|RUN)"
```

---

## Test Organization

### Directory Structure

```
hateblog-backend/
├── internal/
│   ├── domain/
│   │   ├── user/
│   │   │   ├── user.go
│   │   │   └── user_test.go          # Unit tests
│   │   ├── article/
│   │   │   ├── article.go
│   │   │   └── article_test.go       # Unit tests
│   │   └── comment/
│   │       ├── comment.go
│   │       └── comment_test.go       # Unit tests
│   ├── app/
│   │   ├── service/
│   │   │   ├── user_service.go
│   │   │   └── user_service_test.go  # Service unit tests
│   │   └── usecase/
│   │       ├── user_usecase.go
│   │       └── user_usecase_test.go  # Usecase tests
│   └── infra/
│       ├── handler/
│       │   ├── user_handler.go
│       │   ├── user_handler_test.go  # API integration tests
│       │   ├── article_handler.go
│       │   └── article_handler_test.go
│       ├── repository/
│       │   ├── postgres/
│       │   │   ├── user_repository.go
│       │   │   └── user_repository_test.go  # Repository integration tests
│       │   └── redis/
│       │       ├── cache.go
│       │       └── cache_test.go
│       └── middleware/
│           ├── auth.go
│           └── auth_test.go
├── test/
│   ├── e2e/
│   │   ├── user_flow_test.go         # E2E tests
│   │   └── article_flow_test.go
│   ├── fixtures/
│   │   ├── users.json                # Test data
│   │   └── articles.json
│   ├── helpers/
│   │   ├── testcontainer.go          # Shared test utilities
│   │   ├── testdata.go
│   │   └── assertions.go
│   └── testdata/
│       └── sample_data.sql           # SQL fixtures
└── docs/
    ├── TESTING.md                    # Testing strategy
    └── TESTING_GUIDE.md              # This guide
```

### Naming Conventions

**Test Files:**
- `*_test.go` - Must end with `_test.go`
- Located in the same package as the code being tested
- Example: `user.go` → `user_test.go`

**Test Functions:**
- `func Test<FunctionName>(t *testing.T)` - Basic tests
- `func Test<FunctionName>_<Scenario>(t *testing.T)` - Specific scenario
- Examples:
  - `TestCreateUser`
  - `TestCreateUser_Success`
  - `TestCreateUser_InvalidEmail`
  - `TestCreateUser_DuplicateEmail`

**Benchmark Tests:**
- `func Benchmark<FunctionName>(b *testing.B)`
- Example: `BenchmarkUserValidation`

**Example Tests (godoc):**
- `func Example<FunctionName>()`
- Example: `ExampleUser_Validate`

### Test Types by Location

| Location | Test Type | Infrastructure | Focus |
|----------|-----------|----------------|-------|
| `internal/domain/` | Unit | None | Business logic, validation |
| `internal/app/` | Unit/Integration | Mocked repos | Use case orchestration |
| `internal/infra/handler/` | API Integration | Testcontainers | HTTP endpoints |
| `internal/infra/repository/` | Integration | Testcontainers | Database operations |
| `test/e2e/` | E2E | Testcontainers | Complete user flows |

---

## Writing Tests

### Unit Tests (Domain Layer)

**Example: Domain Entity Validation**

```go
// internal/domain/user/user_test.go
package user_test

import (
    "testing"
    "hateblog/internal/domain/user"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUser_Validate(t *testing.T) {
    tests := []struct {
        name    string
        user    user.User
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid user",
            user: user.User{
                Name:  "Alice",
                Email: "alice@example.com",
            },
            wantErr: false,
        },
        {
            name: "empty name",
            user: user.User{
                Name:  "",
                Email: "alice@example.com",
            },
            wantErr: true,
            errMsg:  "name is required",
        },
        {
            name: "invalid email format",
            user: user.User{
                Name:  "Alice",
                Email: "invalid-email",
            },
            wantErr: true,
            errMsg:  "invalid email format",
        },
        {
            name: "email too long",
            user: user.User{
                Name:  "Alice",
                Email: strings.Repeat("a", 256) + "@example.com",
            },
            wantErr: true,
            errMsg:  "email too long",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.user.Validate()

            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestUser_IsActive(t *testing.T) {
    t.Run("active user", func(t *testing.T) {
        user := user.User{Status: user.StatusActive}
        assert.True(t, user.IsActive())
    })

    t.Run("inactive user", func(t *testing.T) {
        user := user.User{Status: user.StatusInactive}
        assert.False(t, user.IsActive())
    })
}
```

### Integration Tests (Repository Layer)

**Example: PostgreSQL Repository**

```go
// internal/infra/repository/postgres/user_repository_test.go
package postgres_test

import (
    "context"
    "testing"

    "hateblog/internal/domain/user"
    "hateblog/internal/infra/repository/postgres"
    "hateblog/test/helpers"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup testcontainer
    ctx := context.Background()
    container, connStr := helpers.SetupPostgresContainer(t, ctx)
    defer container.Terminate(ctx)

    // Initialize repository
    db := helpers.ConnectDB(t, connStr)
    defer db.Close()

    repo := postgres.NewUserRepository(db)

    t.Run("create user successfully", func(t *testing.T) {
        user := &user.User{
            Name:  "Alice",
            Email: "alice@example.com",
        }

        err := repo.Create(ctx, user)
        require.NoError(t, err)
        assert.NotEmpty(t, user.ID)
        assert.NotZero(t, user.CreatedAt)
    })

    t.Run("duplicate email error", func(t *testing.T) {
        user1 := &user.User{
            Name:  "Bob",
            Email: "bob@example.com",
        }
        err := repo.Create(ctx, user1)
        require.NoError(t, err)

        user2 := &user.User{
            Name:  "Bob Jr",
            Email: "bob@example.com", // duplicate
        }
        err = repo.Create(ctx, user2)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "duplicate")
    })
}

func TestUserRepository_FindByID(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    ctx := context.Background()
    container, connStr := helpers.SetupPostgresContainer(t, ctx)
    defer container.Terminate(ctx)

    db := helpers.ConnectDB(t, connStr)
    defer db.Close()

    repo := postgres.NewUserRepository(db)

    t.Run("find existing user", func(t *testing.T) {
        // Create test user
        created := &user.User{
            Name:  "Charlie",
            Email: "charlie@example.com",
        }
        require.NoError(t, repo.Create(ctx, created))

        // Find by ID
        found, err := repo.FindByID(ctx, created.ID)
        require.NoError(t, err)
        assert.Equal(t, created.ID, found.ID)
        assert.Equal(t, created.Email, found.Email)
    })

    t.Run("user not found", func(t *testing.T) {
        _, err := repo.FindByID(ctx, "nonexistent-id")
        assert.Error(t, err)
        assert.ErrorIs(t, err, user.ErrNotFound)
    })
}
```

### API Integration Tests (Handler Layer)

**Example: HTTP Handler**

```go
// internal/infra/handler/user_handler_test.go
package handler_test

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "hateblog/internal/infra/handler"
    "hateblog/test/helpers"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserHandler_CreateUser(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping API integration test")
    }

    // Setup test environment
    ctx := context.Background()
    pgContainer, pgConnStr := helpers.SetupPostgresContainer(t, ctx)
    defer pgContainer.Terminate(ctx)

    redisContainer, redisAddr := helpers.SetupRedisContainer(t, ctx)
    defer redisContainer.Terminate(ctx)

    // Initialize application
    app := helpers.SetupTestApp(t, pgConnStr, redisAddr)

    t.Run("create user successfully", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "name":  "Alice",
            "email": "alice@example.com",
        }
        body, _ := json.Marshal(reqBody)

        req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        rec := httptest.NewRecorder()

        app.ServeHTTP(rec, req)

        assert.Equal(t, http.StatusCreated, rec.Code)

        var response map[string]interface{}
        require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
        assert.NotEmpty(t, response["id"])
        assert.Equal(t, "Alice", response["name"])
        assert.Equal(t, "alice@example.com", response["email"])
    })

    t.Run("validation error - empty name", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "name":  "",
            "email": "alice@example.com",
        }
        body, _ := json.Marshal(reqBody)

        req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        rec := httptest.NewRecorder()

        app.ServeHTTP(rec, req)

        assert.Equal(t, http.StatusBadRequest, rec.Code)

        var response map[string]interface{}
        require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
        assert.Contains(t, response["error"], "name")
    })

    t.Run("validation error - invalid email", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "name":  "Alice",
            "email": "invalid-email",
        }
        body, _ := json.Marshal(reqBody)

        req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        rec := httptest.NewRecorder()

        app.ServeHTTP(rec, req)

        assert.Equal(t, http.StatusBadRequest, rec.Code)
    })

    t.Run("duplicate email error", func(t *testing.T) {
        // Create first user
        reqBody := map[string]interface{}{
            "name":  "Bob",
            "email": "bob@example.com",
        }
        body, _ := json.Marshal(reqBody)

        req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        rec := httptest.NewRecorder()
        app.ServeHTTP(rec, req)
        require.Equal(t, http.StatusCreated, rec.Code)

        // Try to create duplicate
        req = httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        rec = httptest.NewRecorder()
        app.ServeHTTP(rec, req)

        assert.Equal(t, http.StatusConflict, rec.Code)
    })
}

func TestUserHandler_GetUser(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping API integration test")
    }

    ctx := context.Background()
    pgContainer, pgConnStr := helpers.SetupPostgresContainer(t, ctx)
    defer pgContainer.Terminate(ctx)

    redisContainer, redisAddr := helpers.SetupRedisContainer(t, ctx)
    defer redisContainer.Terminate(ctx)

    app := helpers.SetupTestApp(t, pgConnStr, redisAddr)

    // Create test user
    userID := helpers.CreateTestUser(t, app, "Charlie", "charlie@example.com")

    t.Run("get existing user", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/users/"+userID, nil)
        rec := httptest.NewRecorder()

        app.ServeHTTP(rec, req)

        assert.Equal(t, http.StatusOK, rec.Code)

        var response map[string]interface{}
        require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
        assert.Equal(t, userID, response["id"])
        assert.Equal(t, "Charlie", response["name"])
    })

    t.Run("user not found", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/users/nonexistent-id", nil)
        rec := httptest.NewRecorder()

        app.ServeHTTP(rec, req)

        assert.Equal(t, http.StatusNotFound, rec.Code)
    })
}
```

### E2E Tests

**Example: Complete User Flow**

```go
// test/e2e/user_flow_test.go
package e2e_test

import (
    "context"
    "testing"

    "hateblog/test/helpers"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserRegistrationAndLoginFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test")
    }

    ctx := context.Background()
    env := helpers.SetupE2EEnvironment(t, ctx)
    defer env.Teardown()

    client := helpers.NewAPIClient(env.BaseURL)

    // Step 1: Register new user
    t.Log("Step 1: Register new user")
    registerResp, err := client.RegisterUser(ctx, map[string]string{
        "name":     "Alice",
        "email":    "alice@example.com",
        "password": "SecurePass123!",
    })
    require.NoError(t, err)
    assert.Equal(t, 201, registerResp.StatusCode)
    userID := registerResp.Data["id"].(string)

    // Step 2: Verify user can login
    t.Log("Step 2: Login with credentials")
    loginResp, err := client.Login(ctx, map[string]string{
        "email":    "alice@example.com",
        "password": "SecurePass123!",
    })
    require.NoError(t, err)
    assert.Equal(t, 200, loginResp.StatusCode)
    accessToken := loginResp.Data["access_token"].(string)
    assert.NotEmpty(t, accessToken)

    // Step 3: Access protected resource
    t.Log("Step 3: Access user profile with token")
    client.SetAuthToken(accessToken)
    profileResp, err := client.GetUserProfile(ctx, userID)
    require.NoError(t, err)
    assert.Equal(t, 200, profileResp.StatusCode)
    assert.Equal(t, "Alice", profileResp.Data["name"])

    // Step 4: Update profile
    t.Log("Step 4: Update user profile")
    updateResp, err := client.UpdateUser(ctx, userID, map[string]string{
        "name": "Alice Smith",
    })
    require.NoError(t, err)
    assert.Equal(t, 200, updateResp.StatusCode)
    assert.Equal(t, "Alice Smith", updateResp.Data["name"])
}
```

---

## Test Data Management

### Test Data Builders

Create reusable test data builders:

```go
// test/helpers/testdata.go
package helpers

import (
    "testing"
    "hateblog/internal/domain/user"
    "github.com/google/uuid"
)

type UserBuilder struct {
    user *user.User
}

func NewUserBuilder() *UserBuilder {
    return &UserBuilder{
        user: &user.User{
            ID:    uuid.New().String(),
            Name:  "Test User",
            Email: "test@example.com",
        },
    }
}

func (b *UserBuilder) WithName(name string) *UserBuilder {
    b.user.Name = name
    return b
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
    b.user.Email = email
    return b
}

func (b *UserBuilder) Build() *user.User {
    return b.user
}

// Usage in tests:
func TestExample(t *testing.T) {
    user := NewUserBuilder().
        WithName("Alice").
        WithEmail("alice@example.com").
        Build()
    // ... use user in test
}
```

### JSON Fixtures

```go
// test/fixtures/users.json
[
    {
        "id": "user-1",
        "name": "Alice",
        "email": "alice@example.com",
        "status": "active"
    },
    {
        "id": "user-2",
        "name": "Bob",
        "email": "bob@example.com",
        "status": "active"
    }
]
```

Load fixtures:

```go
// test/helpers/fixtures.go
package helpers

import (
    "encoding/json"
    "os"
    "testing"

    "hateblog/internal/domain/user"
)

func LoadUserFixtures(t *testing.T) []user.User {
    t.Helper()

    data, err := os.ReadFile("test/fixtures/users.json")
    if err != nil {
        t.Fatalf("failed to read fixtures: %v", err)
    }

    var users []user.User
    if err := json.Unmarshal(data, &users); err != nil {
        t.Fatalf("failed to unmarshal fixtures: %v", err)
    }

    return users
}
```

### Database Cleanup

```go
// test/helpers/cleanup.go
package helpers

import (
    "context"
    "database/sql"
    "testing"
)

func CleanupDatabase(t *testing.T, db *sql.DB) {
    t.Helper()

    ctx := context.Background()
    tables := []string{"comments", "articles", "users"}

    for _, table := range tables {
        _, err := db.ExecContext(ctx, "TRUNCATE TABLE "+table+" CASCADE")
        if err != nil {
            t.Logf("Warning: failed to truncate %s: %v", table, err)
        }
    }
}

// Alternative: Use transactions
func WithTransaction(t *testing.T, db *sql.DB, fn func(*sql.Tx)) {
    t.Helper()

    tx, err := db.Begin()
    if err != nil {
        t.Fatalf("failed to begin transaction: %v", err)
    }

    defer tx.Rollback() // Always rollback

    fn(tx)
}
```

---

## Test Data Management

### Testcontainers Setup

```go
// test/helpers/testcontainer.go
package helpers

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

func SetupPostgresContainer(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
    t.Helper()

    pgContainer, err := postgres.Run(ctx,
        "postgres:17-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second),
        ),
    )
    if err != nil {
        t.Fatalf("failed to start postgres container: %v", err)
    }

    host, err := pgContainer.Host(ctx)
    if err != nil {
        t.Fatalf("failed to get container host: %v", err)
    }

    port, err := pgContainer.MappedPort(ctx, "5432")
    if err != nil {
        t.Fatalf("failed to get container port: %v", err)
    }

    connStr := fmt.Sprintf(
        "postgresql://testuser:testpass@%s:%s/testdb?sslmode=disable",
        host, port.Port(),
    )

    return pgContainer, connStr
}

func SetupRedisContainer(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
    t.Helper()

    req := testcontainers.ContainerRequest{
        Image:        "redis:7-alpine",
        ExposedPorts: []string{"6379/tcp"},
        WaitingFor:   wait.ForLog("Ready to accept connections"),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Fatalf("failed to start redis container: %v", err)
    }

    host, err := container.Host(ctx)
    if err != nil {
        t.Fatalf("failed to get container host: %v", err)
    }

    port, err := container.MappedPort(ctx, "6379")
    if err != nil {
        t.Fatalf("failed to get container port: %v", err)
    }

    addr := fmt.Sprintf("%s:%s", host, port.Port())

    return container, addr
}
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Download dependencies
        run: go mod download

      - name: Run unit tests
        run: go test -short ./... -v

      - name: Run integration tests
        run: go test ./... -v -race -shuffle=on -timeout=10m

      - name: Generate coverage report
        run: go test ./... -coverprofile=coverage.out -covermode=atomic

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          flags: unittests

      - name: Run linter
        run: make lint

      - name: Run security checks
        run: make security

  build:
    needs: test
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Build binary
        run: make build
```

### GitLab CI Example

```yaml
# .gitlab-ci.yml
stages:
  - test
  - build

variables:
  DOCKER_DRIVER: overlay2

test:unit:
  stage: test
  image: golang:1.25-alpine
  services:
    - docker:dind
  variables:
    DOCKER_HOST: tcp://docker:2375
  before_script:
    - apk add --no-cache make git docker-cli
    - go mod download
  script:
    - go test -short ./... -v
  coverage: '/coverage: \d+.\d+% of statements/'

test:integration:
  stage: test
  image: golang:1.25-alpine
  services:
    - docker:dind
  variables:
    DOCKER_HOST: tcp://docker:2375
  before_script:
    - apk add --no-cache make git docker-cli
    - go mod download
  script:
    - go test ./... -v -race -coverprofile=coverage.out
    - go tool cover -func=coverage.out
  artifacts:
    paths:
      - coverage.out
    expire_in: 1 week

lint:
  stage: test
  image: golangci/golangci-lint:latest
  script:
    - golangci-lint run --config .golangci.yml

security:
  stage: test
  image: golang:1.25-alpine
  before_script:
    - go install github.com/securego/gosec/v2/cmd/gosec@latest
    - go install golang.org/x/vuln/cmd/govulncheck@latest
  script:
    - gosec ./...
    - govulncheck ./...

build:
  stage: build
  image: golang:1.25-alpine
  script:
    - make build
  artifacts:
    paths:
      - bin/
    expire_in: 1 week
```

### Local Pre-commit Hook

```bash
# .git/hooks/pre-commit
#!/bin/sh

echo "Running pre-commit checks..."

# Format code
make fmt

# Run linter
make lint
if [ $? -ne 0 ]; then
    echo "Linting failed"
    exit 1
fi

# Run tests
make test-short
if [ $? -ne 0 ]; then
    echo "Tests failed"
    exit 1
fi

echo "All checks passed!"
```

Make it executable:
```bash
chmod +x .git/hooks/pre-commit
```

---

## Troubleshooting

### Common Issues

#### 1. Docker Not Running

**Error:**
```
failed to start container: Cannot connect to the Docker daemon
```

**Solution:**
```bash
# Check if Docker is running
docker ps

# Start Docker (macOS)
open -a Docker

# Start Docker (Linux)
sudo systemctl start docker
```

#### 2. Port Already in Use

**Error:**
```
bind: address already in use
```

**Solution:**
```bash
# Find process using the port
lsof -i :5432
# or
netstat -tulpn | grep 5432

# Kill the process
kill -9 <PID>

# Or let testcontainers use a random port (default behavior)
```

#### 3. Container Startup Timeout

**Error:**
```
context deadline exceeded
```

**Solution:**
```go
// Increase timeout in wait strategy
wait.ForLog("ready to accept connections").
    WithOccurrence(2).
    WithStartupTimeout(60*time.Second)  // Increase from 30s
```

#### 4. Database Connection Pool Exhausted

**Error:**
```
pq: sorry, too many clients already
```

**Solution:**
```go
// Limit max connections in tests
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

#### 5. Tests Hang or Timeout

**Symptoms:** Tests never complete

**Solutions:**
```bash
# Run with verbose output to see which test hangs
go test -v -timeout=30s ./...

# Run tests one at a time
go test -parallel 1 ./...

# Check for goroutine leaks
go test -race ./...
```

#### 6. Flaky Tests

**Symptoms:** Tests pass sometimes, fail other times

**Common causes:**
- Race conditions (use `-race` flag)
- Timing dependencies (use channels/sync primitives)
- Shared state between tests
- External dependencies

**Solutions:**
```bash
# Run tests multiple times to identify flaky tests
go test -count=10 ./...

# Shuffle test execution order
go test -shuffle=on ./...

# Enable race detector
go test -race ./...
```

#### 7. Clean Up After Failed Tests

```bash
# Remove all test containers
docker ps -a | grep testcontainers | awk '{print $1}' | xargs docker rm -f

# Clean up Docker volumes
docker volume prune -f

# Clean up test cache
go clean -testcache
```

#### 8. Coverage Report Not Generated

**Error:**
```
coverage: [no statements]
```

**Solution:**
```bash
# Ensure you're running tests in the right directory
go test ./... -coverprofile=coverage.out

# Check if there are actually test files
find . -name "*_test.go"

# View coverage for specific package
go test ./internal/domain/user -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Best Practices

### 1. Test Independence

Each test should be completely independent:

```go
// ❌ BAD: Tests depend on execution order
var globalUser *User

func TestCreateUser(t *testing.T) {
    globalUser = &User{Name: "Alice"}
    // ...
}

func TestUpdateUser(t *testing.T) {
    globalUser.Name = "Bob"  // Depends on TestCreateUser
    // ...
}

// ✅ GOOD: Each test is independent
func TestCreateUser(t *testing.T) {
    user := &User{Name: "Alice"}
    // ... test in isolation
}

func TestUpdateUser(t *testing.T) {
    user := &User{Name: "Alice"}
    // ... create own test data
    user.Name = "Bob"
    // ...
}
```

### 2. Use Table-Driven Tests

For multiple similar test cases:

```go
// ✅ GOOD: Table-driven test
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"missing @", "userexample.com", true},
        {"missing domain", "user@", true},
        {"empty", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 3. Use Helper Functions

Mark helper functions with `t.Helper()`:

```go
func createTestUser(t *testing.T, name string) *User {
    t.Helper()  // Stack traces will point to caller, not this function

    user := &User{Name: name}
    if err := user.Validate(); err != nil {
        t.Fatalf("failed to create test user: %v", err)
    }
    return user
}
```

### 4. Cleanup with Defer

Always clean up resources:

```go
func TestWithDatabase(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()  // Always cleanup

    // ... test code
}
```

### 5. Use require vs assert

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Use require for critical assertions (stops test on failure)
require.NoError(t, err, "setup must succeed")

// Use assert for non-critical assertions (test continues)
assert.Equal(t, expected, actual)
assert.True(t, condition)
```

### 6. Test Error Cases

Always test both success and failure paths:

```go
func TestCreateUser(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        // Test happy path
    })

    t.Run("invalid email", func(t *testing.T) {
        // Test validation error
    })

    t.Run("duplicate email", func(t *testing.T) {
        // Test constraint violation
    })

    t.Run("database error", func(t *testing.T) {
        // Test infrastructure failure
    })
}
```

### 7. Avoid Testing Implementation Details

Test behavior, not implementation:

```go
// ❌ BAD: Testing internal implementation
func TestUserRepository_Create(t *testing.T) {
    repo := NewUserRepository(db)
    user := &User{Name: "Alice"}

    // Don't test that it calls the right SQL query
    assert.Contains(t, repo.lastQuery, "INSERT INTO users")
}

// ✅ GOOD: Test observable behavior
func TestUserRepository_Create(t *testing.T) {
    repo := NewUserRepository(db)
    user := &User{Name: "Alice"}

    err := repo.Create(ctx, user)
    require.NoError(t, err)

    // Verify the result by querying the database
    found, err := repo.FindByID(ctx, user.ID)
    require.NoError(t, err)
    assert.Equal(t, "Alice", found.Name)
}
```

### 8. Use Subtests for Organization

```go
func TestUserService(t *testing.T) {
    t.Run("Create", func(t *testing.T) {
        t.Run("Success", func(t *testing.T) { /* ... */ })
        t.Run("ValidationError", func(t *testing.T) { /* ... */ })
    })

    t.Run("Update", func(t *testing.T) {
        t.Run("Success", func(t *testing.T) { /* ... */ })
        t.Run("NotFound", func(t *testing.T) { /* ... */ })
    })
}
```

### 9. Document Complex Test Logic

```go
func TestComplexBusinessLogic(t *testing.T) {
    // GIVEN: A user with an active subscription
    user := createTestUser(t, "Alice")
    subscription := createActiveSubscription(t, user.ID)

    // WHEN: The subscription renewal fails
    err := service.RenewSubscription(ctx, subscription.ID)

    // THEN: The user should receive a notification
    // AND: The subscription status should be marked as "payment_failed"
    require.Error(t, err)

    notifications := getNotifications(t, user.ID)
    assert.Len(t, notifications, 1)
    assert.Equal(t, "payment_failed", notifications[0].Type)

    updated, _ := service.GetSubscription(ctx, subscription.ID)
    assert.Equal(t, "payment_failed", updated.Status)
}
```

### 10. Performance Testing

```go
func BenchmarkUserValidation(b *testing.B) {
    user := &User{
        Name:  "Alice",
        Email: "alice@example.com",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = user.Validate()
    }
}

// Run benchmarks:
// go test -bench=. -benchmem ./...
```

---

## Additional Resources

### Documentation
- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Testcontainers for Go](https://golang.testcontainers.org/)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)

### Books
- "Learn Go with Tests" by Chris James
- "Test-Driven Development in Go" by Adelina Simion

### Tools
- [gotestsum](https://github.com/gotestyourself/gotestsum) - Better test output
- [go-carpet](https://github.com/msoap/go-carpet) - Visual coverage in terminal
- [testify](https://github.com/stretchr/testify) - Testing toolkit

### Commands Cheatsheet

```bash
# Run tests
make test                          # All tests with race detector
make test-short                    # Fast tests without race detector
make cover                         # Tests with coverage report

# Specific tests
go test ./internal/domain/...      # Test specific package
go test -run TestCreateUser        # Test specific function
go test -short ./...               # Skip integration tests

# Coverage
go test -cover ./...               # Show coverage percentage
go test -coverprofile=c.out ./...  # Generate coverage file
go tool cover -html=c.out          # View coverage in browser

# Benchmarks
go test -bench=. ./...             # Run all benchmarks
go test -bench=BenchmarkUser -benchmem  # Specific benchmark with memory stats

# Debugging
go test -v ./...                   # Verbose output
go test -race ./...                # Race detector
go test -count=1 ./...             # Disable test cache
go test -failfast ./...            # Stop on first failure

# CI/CD
make ci                            # Run all checks (fmt, lint, test, security)
```

---

## Summary

This testing guide provides a complete reference for:

1. **Running tests** at different levels (unit, integration, E2E)
2. **Writing tests** following best practices and patterns
3. **Managing test data** with builders and fixtures
4. **Using testcontainers** for realistic integration tests
5. **CI/CD integration** for automated testing
6. **Troubleshooting** common issues

Remember the core philosophy:
- **API tests are the primary quality gate**
- **Unit tests for critical business logic only**
- **Use real infrastructure (testcontainers) instead of mocks**
- **Keep tests independent and fast**
- **Focus on behavior, not implementation**

For questions or issues, refer to the [TESTING.md](./TESTING.md) strategy document or consult the team.
