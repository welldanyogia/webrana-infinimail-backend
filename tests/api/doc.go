// Package api contains tests that run against a real backend server.
//
// These tests require the backend server to be running before execution.
// They test all API endpoints defined in the OpenAPI specification.
//
// Usage:
//
//	# Start the backend server first
//	go run cmd/server/main.go
//
//	# Then run the API tests
//	go test -tags=api ./tests/api/... -v
//
// Environment Variables:
//
//	API_BASE_URL - Base URL of the API server (default: http://localhost:8080)
//	API_KEY      - API key for authentication (default: test-api-key-for-development-only-32chars)
package api
