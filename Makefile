# Makefile for Infinimail Backend Testing

.PHONY: test test-unit test-integration test-e2e test-coverage test-race clean-test build run lint

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
