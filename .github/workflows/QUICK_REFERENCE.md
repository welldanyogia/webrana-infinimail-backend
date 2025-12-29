# CI/CD Quick Reference Guide

## Workflow Triggers Cheat Sheet

| Workflow | Auto Trigger | Manual | Cron |
|----------|-------------|--------|------|
| **backend-ci.yml** | Push to main/develop, PRs to main | ❌ | ❌ |
| **backend-cd.yml** | Push to main, tags v*.*.* | ✅ | ❌ |
| **backend-security.yml** | Push to main/develop, PRs to main | ✅ | ✅ Daily 2AM UTC |

## Common Commands

### Local Development
```bash
# Pre-commit checks
make check

# Full test suite
make test

# Coverage report
make test-coverage

# Install tools
make install-tools

# Build binary
make build

# Docker operations
make docker-build
make docker-run
make docker-stop
```

### GitHub CLI
```bash
# Trigger CD workflow manually
gh workflow run backend-cd.yml

# Trigger security scan
gh workflow run backend-security.yml

# View workflow status
gh run list

# Watch workflow
gh run watch

# View logs
gh run view <run-id> --log
```

### Docker Operations
```bash
# Pull latest image
docker pull ghcr.io/<username>/<repo>/infinimail-backend:latest

# Pull specific version
docker pull ghcr.io/<username>/<repo>/infinimail-backend:v1.0.0

# Pull by commit
docker pull ghcr.io/<username>/<repo>/infinimail-backend:sha-abc1234

# Run container
docker run -p 8080:8080 -p 2525:2525 \
  -e DATABASE_URL=postgres://user:pass@host:5432/db \
  ghcr.io/<username>/<repo>/infinimail-backend:latest
```

## Release Process

### Create New Release
```bash
# 1. Ensure main branch is clean
git checkout main
git pull origin main

# 2. Run tests locally
make check
make test-coverage

# 3. Create and push tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 4. Monitor workflow
gh run watch
```

### Rollback Release
```bash
# 1. Revert to previous image
docker pull ghcr.io/<username>/<repo>/infinimail-backend:v0.9.0

# 2. Update docker-compose.yml
# Change image tag to v0.9.0

# 3. Restart services
docker-compose up -d

# 4. Verify
docker-compose logs -f backend
```

## Troubleshooting Quick Fixes

### CI Failures

**Lint Error:**
```bash
make fmt
make lint
git add .
git commit -m "fix: linting"
git push
```

**Test Failure:**
```bash
# Run specific test
go test ./internal/package -v -run TestName

# Check race conditions
make test-race

# Fix and commit
git add .
git commit -m "fix: test failure"
git push
```

**Build Failure:**
```bash
# Test build locally
make build

# Check dependencies
go mod tidy
go mod verify

# Commit fix
git add go.mod go.sum
git commit -m "fix: dependencies"
git push
```

### CD Failures

**Docker Build Error:**
```bash
# Test Dockerfile locally
docker build -t test-backend .

# Check for syntax errors
docker build --no-cache -t test-backend .
```

**GHCR Push Error:**
```bash
# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Manual push
docker tag test-backend ghcr.io/<username>/<repo>/infinimail-backend:test
docker push ghcr.io/<username>/<repo>/infinimail-backend:test
```

### Security Scan Failures

**Gosec Issues:**
```bash
# Run locally
gosec ./...

# Check specific file
gosec ./internal/handlers/

# Review and fix
# G101: Hardcoded credentials → Use env vars
# G104: Unchecked errors → Add error handling
# G304: File path injection → Validate paths
```

**Dependency Vulnerabilities:**
```bash
# Run govulncheck
govulncheck ./...

# Update vulnerable dependency
go get -u github.com/vulnerable/package@latest
go mod tidy

# Verify fix
govulncheck ./...
```

## Workflow Status Badges

Add to README.md:

```markdown
![Backend CI](https://github.com/<username>/<repo>/workflows/Backend%20CI/badge.svg)
![Backend CD](https://github.com/<username>/<repo>/workflows/Backend%20CD/badge.svg)
![Security Scan](https://github.com/<username>/<repo>/workflows/Backend%20Security%20Scanning/badge.svg)
```

## Environment Variables

### CI/CD Workflows
```yaml
GO_VERSION: '1.24.x'              # Go version for tests/builds
REGISTRY: ghcr.io                 # Container registry
IMAGE_NAME: ${{ github.repository }}/infinimail-backend
```

### Local Development
```bash
export DATABASE_URL=postgres://user:pass@localhost:5432/infinimail
export API_PORT=8080
export SMTP_PORT=2525
export LOG_LEVEL=debug
```

## Coverage Requirements

| Type | Minimum | Target |
|------|---------|--------|
| Unit Tests | 70% | 80%+ |
| Integration Tests | 60% | 75%+ |
| Combined | 75% | 85%+ |

## Performance Benchmarks

| Metric | Target | Max |
|--------|--------|-----|
| CI Workflow | 5 min | 10 min |
| CD Workflow | 8 min | 15 min |
| Security Scan | 10 min | 20 min |
| Docker Build | 3 min | 5 min |

## Security Severity Handling

| Severity | Action | Timeline |
|----------|--------|----------|
| CRITICAL | Block merge, immediate fix | 24 hours |
| HIGH | Block merge, priority fix | 3 days |
| MEDIUM | Warning, scheduled fix | 1 week |
| LOW | Track, fix when convenient | 1 month |

## Artifact Retention

| Artifact | Retention | Purpose |
|----------|-----------|---------|
| coverage-unit | 5 days | Quick review |
| coverage-report | 30 days | Trending analysis |
| backend-binary | 5 days | Testing |
| gosec-report | 30 days | Security audit |
| govulncheck-report | 30 days | Security audit |

## Useful Links

- **Workflows:** `.github/workflows/`
- **Makefile:** `./Makefile`
- **Dockerfile:** `./Dockerfile`
- **Docker Compose:** `./docker-compose.yml`
- **GitHub Actions Docs:** https://docs.github.com/actions
- **Go Security:** https://go.dev/security/vuln/
- **GHCR Docs:** https://docs.github.com/packages

---

**Quick Help:**
```bash
make help              # Show all Makefile targets
gh workflow list       # List all workflows
gh run list            # List recent runs
```

**Emergency Contacts:**
- **CI/CD Issues:** ATLAS (Team Lead)
- **Security Issues:** SENTINEL (Security Lead)
- **Critical Incidents:** NEXUS (Chief Orchestrator)

---

Last Updated: 2025-12-29 | Maintained by: ATLAS
