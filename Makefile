.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  start           Start control-plane and worker"
	@echo "  install         Install dependencies (bun + go modules)"
	@echo "  build           Build control-plane and worker binaries"
	@echo "  test            Run tests"
	@echo "  test_coverage   Run tests with coverage"
	@echo "  lint            Run go vet"
	@echo "  lint_text       Run textlint"
	@echo "  format          Format Go code"
	@echo "  format_check    Check Go code formatting"
	@echo "  clean           Clean build artifacts"
	@echo "  before-commit   Run lint, format check, test, and build"
	@echo "  run_control_plane  Run control-plane only"
	@echo "  run_worker      Run worker only"
	@echo "  sqlc            Generate sqlc code"

.PHONY: install
install:
	bun install
	cd control-plane && go mod download
	cd worker && go mod download

.PHONY: install_ci
install_ci:
	bun install --frozen-lockfile
	cd control-plane && go mod download
	cd worker && go mod download

.PHONY: build
build:
	cd control-plane && go build -o akigura-srv ./cmd/srv
	cd worker && go build -o akigura-worker ./cmd/worker

.PHONY: clean
clean:
	cd control-plane && go clean
	cd worker && go clean

.PHONY: test
test:
	cd control-plane && go test ./...
	cd worker && go test ./...

.PHONY: test_coverage
test_coverage:
	cd control-plane && go test -coverprofile=coverage.out ./...
	cd worker && go test -coverprofile=coverage.out ./...

.PHONY: lint
lint:
	cd control-plane && go vet ./...
	cd worker && go vet ./...

.PHONY: lint_text
lint_text:
	bun run lint:text

.PHONY: format
format:
	cd control-plane && go fmt ./...
	cd worker && go fmt ./...

.PHONY: format_check
format_check:
	@cd control-plane && test -z "$$(gofmt -l .)" || (echo "control-plane: Files need formatting:" && gofmt -l . && exit 1)
	@cd worker && test -z "$$(gofmt -l .)" || (echo "worker: Files need formatting:" && gofmt -l . && exit 1)

.PHONY: before-commit
before-commit: lint_text format_check test build

.PHONY: start
start:
	@echo "Starting AkiGura..."
	@echo ""
	@echo "  API Server:  http://localhost:8000"
	@echo "  Health:      http://localhost:8000/health"
	@echo ""
	@cd control-plane && go run ./cmd/srv &
	@cd worker && go run ./cmd/worker

.PHONY: run_control_plane
run_control_plane:
	cd control-plane && go run ./cmd/srv

.PHONY: run_worker
run_worker:
	cd worker && go run ./cmd/worker

.PHONY: sqlc
sqlc:
	cd control-plane && sqlc generate
