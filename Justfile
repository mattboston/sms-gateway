# Show available commands
default:
    @just --list

# Run Go backend and Vite frontend in development mode
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    trap 'kill 0' EXIT
    (cd src && go run ./cmd/sms-gateway serve --dev-mode) &
    (cd src/web && npm run dev) &
    wait

# Run just the Go backend in dev mode
dev-api:
    cd src && go run ./cmd/sms-gateway serve --dev-mode

# Run just the Vite dev server
dev-web:
    cd src/web && npm run dev

# Build frontend and backend
build: build-web build-api

# Build just the Go binary (assumes frontend already built)
build-api:
    cd src && CGO_ENABLED=0 go build -o ../bin/sms-gateway ./cmd/sms-gateway

# Build for Linux x86_64 (assumes frontend already built)
build-linux:
    cd src && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../bin/sms-gateway-linux-amd64 ./cmd/sms-gateway

# Build for Linux ARM (Raspberry Pi) (assumes frontend already built)
build-linux-arm:
    cd src && CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o ../bin/sms-gateway-linux-arm7 ./cmd/sms-gateway

# Build frontend + all Linux binaries
build-all: build-web build-api build-linux build-linux-arm

# Build just the React frontend
build-web:
    cd src/web && npm ci && npm run build

# Run all Go tests
test:
    cd src && go test ./...

# Run tests with verbose output
test-verbose:
    cd src && go test -v ./...

# Run all linters
lint:
    cd src && golangci-lint run ./...
    cd src/web && npm run lint

# Run all formatters
format:
    cd src && gofumpt -w .
    cd src/web && npm run format

# Regenerate Swagger API docs
swagger:
    cd src && swag init -g cmd/sms-gateway/main.go -o docs --parseDependency --parseInternal

# Run database migrations
migrate *ARGS:
    cd src && go run ./cmd/sms-gateway migrate up {{ ARGS }}

# Rollback last migration
migrate-down:
    cd src && go run ./cmd/sms-gateway migrate down

# Show migration status
migrate-status:
    cd src && go run ./cmd/sms-gateway migrate status

# Create a new migration file
migrate-new NAME:
    goose -dir src/migrations create {{ NAME }} sql

# Initialize development environment
init:
    cd src && go mod download
    cd src/web && npm install
    lefthook install

# Create a release by tagging and pushing (usage: just release 1.0.0)
release VERSION:
    #!/usr/bin/env bash
    set -euo pipefail
    TAG="{{ VERSION }}"
    TAG="${TAG#v}"
    git tag -a "$TAG" -m "Release $TAG"
    git push origin "$TAG"

# Preview what the next release would look like
release-dry-run:
    @echo "Current tags:"
    @git tag --sort=-v:refname | head -5
    @echo ""
    @echo "Commits since last tag:"
    @git log --oneline $(git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~10")..HEAD

# Remove build artifacts
clean:
    rm -rf bin/
    rm -rf src/web/dist/
    rm -rf src/web/node_modules/
