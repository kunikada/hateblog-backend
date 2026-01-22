# Work Rules

To avoid additional effort caused by rework, follow the rules below.
All responses must be written in Japanese.

## Documents

- Keep the content to the bare minimum required.
- Keep each file to about 200-400 lines, with a maximum of 500 lines.
- If you believe samples or examples are necessary, ask before adding them.
- Separate summary text from any sample or example files.
- Confirm that you are writing in the correct file and not duplicating content.

## Implementation

- If you must do something different from the instructions, explain and get consent first.
- When changing configurations or policies, explain and obtain consent.
- If another issue is found while working, pause and confirm before proceeding.
- Delete any temporary files created during the work.
- When adding debug messages, use debug level output.
- Structure directories and files with their functionality or domain in mind, not just by data type.
- Do not run format/lint/test and other commands after implementation - just report.

## Terminal

- Confirm beforehand before killing any process.

# Architecture

- **`Runtime`**: Go 1.25
- **`Database`**: PostgreSQL 18
- **`Cache`**: Redis
- **`API`**: OpenAPI 3.0 / oapi-codegen
- **`Testing`**: Go testing / GoMock
- **`Migration`**: golang-migrate v4
- **`Lint/Format`**: golangci-lint / gofmt

# Directory Structure

- **`.devcontainer/`**: Development environment definition for DevContainer (Docker + VS Code)
- **`cmd/app/`**: Main application
- **`cmd/admin/`**: Admin tools
- **`cmd/migrator/`**: Data migration execution tool
- **`cmd/fetcher/`**: Data fetching batch
- **`cmd/updater/`**: Data update batch
- **`docs/`**: Architecture, specifications, implementation plans, and other documentation
- **`internal/domain/`**: Domain models and entity definitions
- **`internal/infra/handler/`**: HTTP handlers
- **`internal/infra/postgres/`**: PostgreSQL repository implementation
- **`internal/infra/redis/`**: Redis cache implementation
- **`internal/infra/external/`**: External API integrations
- **`internal/pkg/`**: Application common utilities
- **`internal/platform/`**: Platform features (logging, cache, database, configuration, etc.)
- **`internal/usecase/`**: Business logic and orchestration layer
- **`migrations/`**: Database migration definitions (SQL)
- **`scripts/`**: Shell scripts and initialization scripts
- **`compose.yaml`**: Production Docker Compose configuration
- **`compose.dev.yaml`**: Development Docker Compose additional configuration
- **`Dockerfile`**: Production image definition
- **`Dockerfile.dev`**: Development image definition
- **`Dockerfile.postgres`**: PostgreSQL custom image definition
- **`Makefile`**: Build, run, test, and other command definitions
- **`go.mod`**: Go dependencies definition
- **`openapi.yaml`**: OpenAPI specification definition (source for type generation)
- **`oapi-codegen.yaml`**: oapi-codegen configuration
