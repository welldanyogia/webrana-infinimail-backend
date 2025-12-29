# Quality Gates Configuration

Ship with confidence. This document describes all quality gates enforced in the Infinimail backend CI/CD pipeline.

## Overview

Quality gates are automated checks that must pass before code can be merged to the main branch or released. These gates ensure code quality, test coverage, and security standards are maintained.

## Quality Gate Matrix

| Gate | Type | Threshold | Blocking | Workflow |
|------|------|-----------|----------|----------|
| Linting | Static Analysis | 0 errors | Yes | test.yml |
| Unit Tests | Testing | 100% pass | Yes | test.yml |
| Integration Tests | Testing | 100% pass | Yes | test.yml |
| E2E Tests | Testing | 100% pass | Yes | test.yml |
| Unit Coverage | Coverage | 75% | Yes | codecov.yml |
| Integration Coverage | Coverage | 65% | Yes | codecov.yml |
| Project Coverage | Coverage | 70% | Yes | codecov.yml |
| Patch Coverage | Coverage | 70% | Yes | codecov.yml |
| Race Detection | Concurrency | 0 races | Yes | test.yml |
| Code Review | Human | 1 approval | Yes | Branch Protection |

## Gate Details

### 1. Linting (golangci-lint)

**Purpose:** Ensure code follows Go best practices and style guidelines

**Tool:** `golangci-lint`

**Checks:**
- Code formatting (gofmt)
- Common mistakes (govet)
- Unused code (deadcode, unused)
- Error handling (errcheck)
- Security issues (gosec)
- Complexity (gocyclo)
- And 50+ other linters

**Run locally:**
```bash
make lint
```

**Fix automatically:**
```bash
golangci-lint run --fix
```

**Failures indicate:**
- Code style violations
- Potential bugs
- Security vulnerabilities
- Code smells

### 2. Unit Tests

**Purpose:** Verify individual components work correctly in isolation

**Coverage Target:** 75%

**Run locally:**
```bash
make test-unit
```

**Includes:**
- Package-level tests (*_test.go)
- Fast tests (< 1 second each)
- No external dependencies
- Mock database and services
- Race detection enabled

**Failures indicate:**
- Breaking changes in logic
- Regression in functionality
- Race conditions

### 3. Integration Tests

**Purpose:** Verify components work together with real dependencies

**Coverage Target:** 65%

**Run locally:**
```bash
make test-integration
```

**Includes:**
- Real PostgreSQL database (testcontainers)
- Database migrations
- Repository layer tests
- Service integration tests
- API integration tests

**Failures indicate:**
- Database query issues
- Migration problems
- Integration contract violations
- Resource cleanup issues

### 4. E2E Tests

**Purpose:** Verify complete user workflows work end-to-end

**Coverage Target:** 60%

**Run locally:**
```bash
make test-e2e
```

**Includes:**
- Full application stack
- Real database
- HTTP API tests
- WebSocket tests
- Complete user scenarios

**Failures indicate:**
- API contract violations
- Workflow regressions
- Missing error handling
- Integration issues

### 5. Code Coverage

**Purpose:** Ensure sufficient test coverage of codebase

**Thresholds:**
- **Project Coverage:** 70% (overall codebase)
- **Patch Coverage:** 70% (new code in PR)
- **Unit Tests:** 75% (internal packages)
- **Integration Tests:** 65% (integration layer)
- **E2E Tests:** 60% (end-to-end scenarios)

**Configuration:** `codecov.yml`

**Run locally:**
```bash
make test-coverage
```

**View report:**
```bash
# Generates coverage.html
open coverage.html  # macOS
xdg-open coverage.html  # Linux
start coverage.html  # Windows
```

**Coverage violations:**
- Total coverage < 70% - Build fails
- New code < 70% coverage - Build fails
- Coverage decreases > 1% - Build fails

### 6. Race Detection

**Purpose:** Detect data race conditions in concurrent code

**Tool:** Go race detector (`-race` flag)

**Run locally:**
```bash
make test-race
```

**Failures indicate:**
- Concurrent access to shared data without synchronization
- Potential crashes in production
- Non-deterministic behavior

**Common causes:**
- Unprotected map access
- Missing mutex locks
- Goroutine synchronization issues
- Shared state without channels

### 7. Code Review

**Purpose:** Human verification of code quality and design

**Requirements:**
- At least 1 approval from team member
- All comments resolved
- No requested changes pending

**Review checklist:**
- Code readability
- Design patterns
- Test quality
- Security considerations
- Documentation
- Performance implications

## CI/CD Pipeline Flow

```
┌─────────────────────────────────────────────────────────┐
│                    Push / Pull Request                   │
└────────────────────┬────────────────────────────────────┘
                     │
          ┌──────────┴──────────┐
          ▼                     ▼
    ┌──────────┐          ┌──────────┐
    │   Lint   │          │  Format  │
    └─────┬────┘          └─────┬────┘
          │                     │
          └──────────┬──────────┘
                     ▼
          ┌──────────────────┐
          │   Unit Tests      │
          │  (Race Detector)  │
          │  Coverage: 75%    │
          └─────────┬─────────┘
                    │
          ┌─────────┴──────────┐
          ▼                    ▼
    ┌──────────┐         ┌──────────┐
    │Integration│        │ E2E Tests│
    │   Tests   │        │ Coverage │
    │Coverage 65%│       │  60%     │
    └─────┬─────┘        └─────┬────┘
          │                    │
          └────────┬───────────┘
                   ▼
        ┌────────────────────┐
        │ Coverage Report     │
        │ Threshold: 70%      │
        └──────────┬─────────┘
                   │
                   ▼
        ┌────────────────────┐
        │  Quality Gate       │
        │  Status Check       │
        └──────────┬─────────┘
                   │
        ┌──────────┴──────────┐
        │  All Checks Pass?   │
        └──────────┬──────────┘
                   │
        ┌──────────┴──────────┐
        ▼                     ▼
    ┌──────┐              ┌──────┐
    │ Pass │              │ Fail │
    └──┬───┘              └───┬──┘
       │                      │
       ▼                      ▼
┌──────────────┐      ┌──────────────┐
│ Ready to     │      │ Block Merge  │
│ Merge        │      │ Fix Issues   │
└──────────────┘      └──────────────┘
```

## Running All Quality Gates Locally

### Pre-Commit Check

Run before every commit:

```bash
make check
```

This runs:
- `go fmt ./...` - Format code
- `golangci-lint run` - Lint code
- `go test -short ./internal/...` - Unit tests

### Pre-Push Check

Run before pushing:

```bash
# Format and lint
make fmt lint

# All tests with coverage
make test-coverage

# Check coverage threshold
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE < 70" | bc -l) )); then
  echo "ERROR: Coverage ${COVERAGE}% is below 70%"
  exit 1
fi
```

### Pre-PR Check

Run before creating PR:

```bash
# Full test suite
make test

# Integration tests
make test-integration

# E2E tests (requires backend running)
make test-e2e

# Race detection
make test-race

# Lint with auto-fix
golangci-lint run --fix
```

## Quality Gate Failures

### How to Fix

1. **Linting Failures**
   ```bash
   # View errors
   make lint

   # Auto-fix where possible
   golangci-lint run --fix

   # Manual fixes required for complex issues
   ```

2. **Test Failures**
   ```bash
   # Run specific test
   go test -v ./internal/service -run TestEmailService

   # Run with verbose output
   make test-unit

   # Debug with race detector
   make test-race
   ```

3. **Coverage Below Threshold**
   ```bash
   # Generate coverage report
   make test-coverage

   # View HTML report
   open coverage.html

   # Identify untested code (red in HTML)
   # Add tests for critical paths
   ```

4. **Race Conditions**
   ```bash
   # Run race detector
   make test-race

   # Review race reports
   # Add proper synchronization (mutex, channels)
   # Verify fix with race detector
   ```

## Bypassing Quality Gates (Emergency Only)

Quality gates should NEVER be bypassed except in critical emergencies.

### When Bypass is Acceptable

- **Critical production outage** - Site down, data loss
- **Security vulnerability** - Active exploit in the wild
- **Legal compliance** - Regulatory deadline

### Bypass Procedure

1. **Document the reason** in PR description
2. **Get approval** from team lead (ATLAS)
3. **Create follow-up issue** to add proper tests
4. **Conduct post-incident review**
5. **Administrator override** only (branch protection)

### Never Bypass For

- Feature deadlines
- Demo preparation
- "Tests are slow"
- "Coverage is arbitrary"
- Convenience

## Monitoring Quality Metrics

### GitHub Actions Dashboard

View workflow runs: `https://github.com/<org>/<repo>/actions`

### Codecov Dashboard

View coverage trends: `https://app.codecov.io/gh/<org>/<repo>`

**Key metrics to monitor:**
- Coverage trend over time
- Hotspots (low coverage files)
- Coverage by flags (unit/integration/e2e)
- Patch coverage on PRs

### Quality Trends

Track these metrics over time:
- Average test runtime
- Flaky test rate
- Coverage percentage
- Lint violation count
- PR merge time
- Failed CI runs percentage

## Best Practices

### Writing Tests

1. **Follow AAA pattern** - Arrange, Act, Assert
2. **One assertion per test** - Clear failure messages
3. **Use table-driven tests** - Cover multiple scenarios
4. **Mock external dependencies** - Unit tests should be fast
5. **Test error cases** - Not just happy path

Example:
```go
func TestEmailService_Send(t *testing.T) {
    tests := []struct {
        name    string
        email   *Email
        wantErr bool
    }{
        {
            name:    "valid email",
            email:   &Email{To: "test@example.com"},
            wantErr: false,
        },
        {
            name:    "invalid email",
            email:   &Email{To: "invalid"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            service := NewEmailService(mockRepo)

            // Act
            err := service.Send(tt.email)

            // Assert
            if (err != nil) != tt.wantErr {
                t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Improving Coverage

1. **Focus on critical paths** - Not 100% coverage goal
2. **Test edge cases** - Boundary conditions, errors
3. **Use coverage reports** - Identify gaps visually
4. **Write meaningful tests** - Not just for coverage number
5. **Exclude test files** - Don't test tests

### Managing Flaky Tests

1. **Identify flaky tests** - Track failures over time
2. **Fix immediately** - Don't let them accumulate
3. **Use proper timeouts** - Avoid timing issues
4. **Clean up resources** - Prevent test interference
5. **Isolate test data** - Each test should be independent

## Troubleshooting

### GitHub Actions Failing but Local Passes

**Common causes:**
- Different Go version (check `GO_VERSION` in workflow)
- Missing environment variables
- Cached dependencies
- Different test execution order (race condition)

**Solutions:**
```bash
# Use same Go version as CI
go version  # Check current version
gvm install go1.24  # Install matching version
gvm use go1.24

# Clean cache
go clean -testcache
go clean -modcache

# Run with race detector
go test -race ./...
```

### Coverage Differs Locally vs CI

**Common causes:**
- Different test execution
- Cached coverage files
- Build tags not used

**Solutions:**
```bash
# Clean coverage artifacts
rm -f coverage*.out coverage.html

# Run same command as CI
go test -short -coverprofile=coverage.out -covermode=atomic ./...

# Check coverage
go tool cover -func=coverage.out | grep total
```

### Tests Timeout in CI

**Common causes:**
- Slow database operations
- Missing timeouts in code
- Resource constraints

**Solutions:**
```bash
# Add timeouts to tests
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Use ctx in test
}

# Optimize slow queries
# Add indexes to database
# Use parallel tests where appropriate
```

## Configuration Files

### codecov.yml
Location: `webrana-infinimail-backend/codecov.yml`
Purpose: Configure coverage requirements and reporting

### .github/workflows/test.yml
Location: `webrana-infinimail-backend/.github/workflows/test.yml`
Purpose: CI workflow for quality gates

### .github/workflows/release.yml
Location: `webrana-infinimail-backend/.github/workflows/release.yml`
Purpose: Automated release workflow

### .golangci.yml (Future)
Location: `webrana-infinimail-backend/.golangci.yml` (to be created)
Purpose: Configure linter rules

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [golangci-lint Documentation](https://golangci-lint.run/)
- [Codecov Documentation](https://docs.codecov.com/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go Race Detector](https://go.dev/doc/articles/race_detector)

---

**Last Updated:** 2025-12-29
**Owner:** VALIDATOR (QA & Release Engineer)
