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

.PHONY: run_control_plane
run_control_plane:
	cd control-plane && go run ./cmd/srv

.PHONY: run_worker
run_worker:
	cd worker && go run ./cmd/worker
