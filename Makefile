# Makefile for Infinimail Backend Testing

.PHONY: test test-unit test-integration test-e2e test-coverage test-race clean-test build run lint fmt tidy check ci-test ci-build install-tools quality-gate check-coverage

# Default test target - runs all tests
test:
	go test ./... -v

# Run only unit tests (fast, no external dependencies)
test-unit:
	go test ./internal/... -v -short

# Run integration tests (requires Docker for testcontainers)
test-integration:
	go test ./tests/integration/... -v -tags=integration

# Run end-to-end tests (requires full backend running)
test-e2e:
	go test ./tests/e2e/... -v -tags=e2e

# Run API endpoint tests against real backend (requires backend running)
test-api:
	@echo "Make sure backend is running on localhost:8080"
	go test ./tests/api/... -v -tags=api

# Run tests with coverage report
test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
	go test ./... -race -v

# Clean test artifacts
clean-test:
	rm -f coverage.out coverage.html
	go clean -testcache

# Build the application
build:
	go build -o bin/server ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Run all checks before commit
check: fmt lint test-unit
	@echo "All checks passed!"

# Run quality gate checks locally (mimics CI)
quality-gate: fmt lint test test-coverage check-coverage
	@echo "============================================"
	@echo "Quality Gate: ALL CHECKS PASSED"
	@echo "============================================"

# Check if coverage meets threshold
check-coverage:
	@echo "Checking coverage threshold..."
	@if [ ! -f coverage.out ]; then \
		echo "ERROR: coverage.out not found. Run 'make test-coverage' first."; \
		exit 1; \
	fi
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	THRESHOLD=70; \
	echo "Total coverage: $${COVERAGE}%"; \
	echo "Threshold: $${THRESHOLD}%"; \
	if [ $$(echo "$${COVERAGE} < $${THRESHOLD}" | bc -l) -eq 1 ]; then \
		echo "ERROR: Coverage $${COVERAGE}% is below threshold $${THRESHOLD}%"; \
		exit 1; \
	fi; \
	echo "SUCCESS: Coverage meets threshold"

# Install required tools for CI/development
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Tools installed successfully!"

# CI-friendly test target (no verbose, with race detection)
ci-test:
	go test ./internal/... -short -race -coverprofile=coverage.out -covermode=atomic

# CI-friendly build target (with version info)
ci-build:
	@echo "Building for production..."
	CGO_ENABLED=0 go build -ldflags="-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$(shell git rev-parse --short HEAD)" -o bin/server ./cmd/server
	@echo "Build complete: bin/server"

# Docker build
docker-build:
	docker build -t infinimail-backend:latest .

# Docker run locally
docker-run:
	docker-compose up -d

# Docker stop
docker-stop:
	docker-compose down

# Clean all artifacts
clean-all: clean-test
	rm -rf bin/
	rm -f gosec-report.json govulncheck-results.json
	docker-compose down -v 2>/dev/null || true

# Help target
help:
	@echo "Available targets:"
	@echo "  make test              - Run all tests"
	@echo "  make test-unit         - Run unit tests only"
	@echo "  make test-integration  - Run integration tests"
	@echo "  make test-e2e          - Run end-to-end tests"
	@echo "  make test-coverage     - Generate coverage report"
	@echo "  make test-race         - Run tests with race detector"
	@echo "  make check-coverage    - Verify coverage meets 70% threshold"
	@echo "  make quality-gate      - Run all quality gate checks (CI simulation)"
	@echo "  make lint              - Run linter"
	@echo "  make fmt               - Format code"
	@echo "  make check             - Quick pre-commit checks"
	@echo "  make build             - Build binary"
	@echo "  make ci-test           - CI-friendly test run"
	@echo "  make ci-build          - CI-friendly build with version info"
	@echo "  make install-tools     - Install development tools"
	@echo "  make docker-build      - Build Docker image"
	@echo "  make docker-run        - Run with docker-compose"
	@echo "  make clean-all         - Clean all artifacts"
