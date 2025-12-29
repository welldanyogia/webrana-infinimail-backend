# Infinimail Backend CI/CD Pipeline

This directory contains GitHub Actions workflows for the Infinimail backend continuous integration and deployment pipeline.

## Workflows Overview

### 1. Backend CI (`backend-ci.yml`)

**Purpose:** Continuous Integration - validates code quality, runs tests, and generates coverage reports.

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` branch
- Changes in backend code or workflow files

**Jobs:**

#### Lint Code
- Runs `golangci-lint` with 5-minute timeout
- Checks code formatting with `gofmt`
- Uses Go module caching for faster runs

#### Unit Tests
- **Matrix Strategy:** Tests against Go 1.24.0 and 1.24.x
- Runs `make test-unit` (fast unit tests with `-short` flag)
- Generates coverage report for latest Go version
- Uploads coverage to Codecov and artifacts
- Uses Go module caching

#### Race Condition Tests
- Runs `make test-race` to detect data races
- Critical for concurrent code safety
- Uses `-race` flag to enable race detector

#### Integration Tests
- Spins up PostgreSQL 16 container
- Runs `make test-integration` against real database
- Generates integration coverage report
- Uploads coverage to Codecov

#### Build Test
- Runs `make build` to verify binary compilation
- Verifies binary with `file` command
- Uploads binary as artifact (5-day retention)
- Depends on: lint, test-unit, test-race

#### Coverage Report
- **Runs only on main branch pushes**
- Generates combined coverage with `make test-coverage`
- Uploads HTML coverage report (30-day retention)
- Creates coverage summary in GitHub Step Summary
- Uploads to Codecov with `combined` flag

**Usage:**
```bash
# Triggered automatically on push/PR
# To test locally:
make check           # Runs fmt, lint, test-unit
make test-coverage   # Generates full coverage report
```

---

### 2. Backend CD (`backend-cd.yml`)

**Purpose:** Continuous Deployment - builds and publishes Docker images to GitHub Container Registry.

**Triggers:**
- Push to `main` branch (after CI passes)
- Git tags matching `v*.*.*` (semantic versioning)
- Manual workflow dispatch

**Jobs:**

#### Wait for CI
- Waits for CI workflow to complete successfully
- Prevents deploying broken code
- Uses `lewagon/wait-on-check-action`

#### Build and Push Docker Image
- **Multi-platform builds:** `linux/amd64`, `linux/arm64`
- **Registry:** `ghcr.io` (GitHub Container Registry)
- **Caching:** GitHub Actions cache for faster builds
- **Tag Strategy:**
  - `latest` - Latest main branch build
  - `sha-<short>` - Short commit SHA
  - `sha-<long>` - Full commit SHA
  - `v1.2.3` - Semantic version (on tags)
  - `v1.2` - Major.minor (on tags)
  - `v1` - Major version (on tags)
  - Custom tags via workflow dispatch

**Build Arguments:**
- `BUILD_DATE` - Repository update timestamp
- `VCS_REF` - Git commit SHA
- `VERSION` - Git ref name

**Metadata Labels:**
- `org.opencontainers.image.title`
- `org.opencontainers.image.description`
- `org.opencontainers.image.vendor`
- `maintainer`

#### Scan Image
- Runs Trivy vulnerability scanner
- Scans for CRITICAL and HIGH vulnerabilities
- Uploads SARIF results to GitHub Security tab
- Generates table output for review

#### Verify Image
- Pulls the built image from GHCR
- Inspects image metadata
- Tests container startup
- Ensures container runs successfully for 5 seconds
- Checks logs for errors

#### Create Release Notes
- **Runs only on version tags** (e.g., `v1.2.3`)
- Generates changelog from git commits
- Compares with previous tag
- Creates GitHub Release
- Marks as prerelease for alpha/beta/rc tags

#### Notify Success
- Creates deployment summary
- Provides next steps for production deployment
- Lists post-deployment actions

**Usage:**
```bash
# Automatic on main branch push:
git push origin main

# Manual trigger:
gh workflow run backend-cd.yml

# Create release:
git tag v1.0.0
git push origin v1.0.0

# Pull image:
docker pull ghcr.io/welldanyogia/webrana-infinimail/infinimail-backend:latest
```

---

### 3. Backend Security (`backend-security.yml`)

**Purpose:** Security scanning - identifies vulnerabilities, secrets, and compliance issues.

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` branch
- **Daily schedule:** 2 AM UTC (cron: `0 2 * * *`)
- Manual workflow dispatch

**Jobs:**

#### Gosec Security Scan
- Runs `gosec` static security analyzer
- Scans for Go-specific security issues
- Generates SARIF for GitHub Security tab
- Generates JSON report for artifacts
- Checks for:
  - SQL injection vulnerabilities
  - Command injection
  - Hardcoded credentials
  - Weak cryptography
  - Unsafe file operations

#### Trivy Filesystem Scan
- Scans filesystem for vulnerabilities
- Checks dependencies and code
- Severity levels: CRITICAL, HIGH, MEDIUM
- Uploads SARIF to GitHub Security

#### Dependency Vulnerability Check
- Runs `govulncheck` from Go vulnerability database
- Identifies known vulnerabilities in dependencies
- Generates JSON report
- Provides detailed vulnerability information
- Parses and displays results in summary

#### Secrets Scan
- Uses Gitleaks to detect secrets
- Scans entire git history
- Identifies:
  - API keys
  - Passwords
  - Tokens
  - Private keys
  - AWS credentials

#### Advanced Security Scan (Placeholder)
- **Reserved for SENTINEL**
- SAST (Static Application Security Testing)
- DAST (Dynamic Application Security Testing)
- License compliance checking
- Container hardening verification
- API security testing
- OWASP Top 10 checks

#### Compliance Check (Placeholder)
- **Reserved for SENTINEL**
- CIS Benchmark compliance
- PCI-DSS requirements
- GDPR compliance
- Security policy enforcement
- Audit logging verification

#### Security Summary
- Aggregates all scan results
- Provides recommendations
- Links to detailed findings
- Notes for SENTINEL completion

**Usage:**
```bash
# Manual security scan:
gh workflow run backend-security.yml

# Local security checks:
make install-tools
gosec ./...
govulncheck ./...
```

---

## Setup Requirements

### GitHub Secrets

Configure the following secrets in your repository settings:

**Optional (for enhanced features):**
- `CODECOV_TOKEN` - Codecov integration for coverage reports
- `GITLEAKS_LICENSE` - Gitleaks Pro license (optional)

**Automatic:**
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

### GitHub Container Registry (GHCR)

GHCR is automatically available for GitHub repositories. Images are pushed to:
```
ghcr.io/<username>/<repository>/infinimail-backend
```

**Permissions:**
- Workflow has `packages: write` permission
- Images are private by default
- Configure package visibility in repository settings

### Repository Settings

**Enable:**
1. GitHub Actions in repository settings
2. Read and write permissions for workflows
3. Allow GitHub Actions to create and approve pull requests (optional)

**Branch Protection (Recommended):**
- Require status checks to pass before merging
- Required checks:
  - Lint Code
  - Unit Tests
  - Race Condition Tests
  - Integration Tests
- Require pull request reviews

---

## Workflow Dependencies

### Makefile Targets

All workflows leverage existing Makefile targets:

```makefile
make lint              # Run golangci-lint
make test-unit         # Run unit tests
make test-integration  # Run integration tests
make test-coverage     # Generate coverage report
make test-race         # Run race detector
make build             # Build binary
make ci-test           # CI-friendly test
make ci-build          # CI-friendly build with version info
make install-tools     # Install development tools
```

### Go Version

- **Primary:** Go 1.24.x (latest patch)
- **Matrix:** Go 1.24.0, 1.24.x
- Defined in `env.GO_VERSION`

---

## CI/CD Pipeline Flow

### Pull Request Flow
```
1. Developer creates PR to main
2. Trigger: backend-ci.yml
   ├── Lint Code (parallel)
   ├── Unit Tests Go 1.24.0 (parallel)
   ├── Unit Tests Go 1.24.x (parallel)
   ├── Race Condition Tests (parallel)
   └── Integration Tests (parallel)
3. Trigger: backend-security.yml
   ├── Gosec Scan (parallel)
   ├── Trivy Filesystem Scan (parallel)
   ├── Dependency Check (parallel)
   └── Secrets Scan (parallel)
4. Build Test (after tests pass)
5. PR status checks updated
6. Code review and approval
7. Merge to main
```

### Main Branch Flow
```
1. Code merged to main
2. Trigger: backend-ci.yml
   ├── All CI jobs run
   └── Coverage Report generated
3. Wait for CI completion
4. Trigger: backend-cd.yml
   ├── Build Docker image (multi-platform)
   ├── Push to GHCR with tags
   ├── Scan image with Trivy
   ├── Verify image startup
   └── Generate deployment summary
5. Image ready for deployment
```

### Release Flow
```
1. Developer creates version tag
   git tag v1.0.0
   git push origin v1.0.0
2. Trigger: backend-cd.yml
   ├── Build and push with version tags
   ├── Scan and verify
   ├── Generate changelog
   └── Create GitHub Release
3. Release published with Docker pull command
```

---

## Monitoring and Debugging

### Workflow Logs

View workflow runs:
```bash
# List workflow runs
gh run list --workflow=backend-ci.yml

# View specific run
gh run view <run-id>

# Download logs
gh run download <run-id>
```

### Artifacts

Generated artifacts (available for download):
- **coverage-unit** (5 days) - Unit test coverage
- **coverage-report** (30 days) - Combined coverage HTML
- **infinimail-backend-binary** (5 days) - Compiled binary
- **gosec-report** (30 days) - Security scan results
- **govulncheck-report** (30 days) - Vulnerability check results

### GitHub Security Tab

Security findings are uploaded to:
- Repository → Security → Code scanning alerts
- Categories: gosec, trivy-fs, trivy-image

---

## Performance Optimization

### Caching Strategy

**Go Module Cache:**
- Uses `actions/setup-go@v5` built-in caching
- Cache key: `go.sum` hash
- Shared across jobs

**Docker Build Cache:**
- Uses GitHub Actions cache
- Mode: `max` for maximum layer caching
- Speeds up subsequent builds significantly

### Parallel Execution

Jobs run in parallel when possible:
- Lint, Unit Tests, Race Tests (CI)
- Gosec, Trivy, Dependency Check (Security)

### Matrix Strategy

Unit tests run against multiple Go versions in parallel:
- Go 1.24.0 (specific version)
- Go 1.24.x (latest patch)

---

## Rollback Strategy

### Image Rollback

Previous images are always available:
```bash
# List all tags
docker pull ghcr.io/welldanyogia/webrana-infinimail/infinimail-backend:sha-<commit>

# Rollback to previous version
docker pull ghcr.io/welldanyogia/webrana-infinimail/infinimail-backend:v1.0.0
```

### Deployment Rollback

```bash
# Update docker-compose.yml with previous tag
image: ghcr.io/welldanyogia/webrana-infinimail/infinimail-backend:v1.0.0

# Redeploy
docker-compose up -d
```

---

## Best Practices

### For Developers

1. **Run checks locally before pushing:**
   ```bash
   make check           # Runs fmt, lint, test-unit
   make test-coverage   # Verify coverage
   ```

2. **Write tests for all new features**
3. **Keep coverage above 70%**
4. **Fix security findings immediately**
5. **Use semantic versioning for tags**

### For DevOps

1. **Monitor daily security scans**
2. **Update dependencies regularly**
3. **Review and triage security alerts**
4. **Keep CI/CD workflows updated**
5. **Monitor workflow execution times**
6. **Optimize caching strategies**

---

## Troubleshooting

### Common Issues

**Issue: Lint failures**
```bash
# Fix locally
make fmt
make lint
git commit -am "fix: linting issues"
```

**Issue: Test failures**
```bash
# Run specific tests
go test ./internal/... -v -run TestName

# Check race conditions
make test-race
```

**Issue: Docker build failures**
```bash
# Build locally
make docker-build

# Check Dockerfile syntax
docker build -t test .
```

**Issue: GHCR authentication**
```bash
# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```

---

## Future Enhancements

### Planned (by SENTINEL)

- [ ] Advanced SAST/DAST scanning
- [ ] API security testing
- [ ] Performance testing integration
- [ ] Load testing automation
- [ ] Blue-green deployment support
- [ ] Canary deployment automation
- [ ] Automated rollback on failure
- [ ] Enhanced compliance checks

### Monitoring Integration

- [ ] Prometheus metrics export
- [ ] Grafana dashboard provisioning
- [ ] Alert manager configuration
- [ ] Log aggregation (ELK/Loki)

---

## Support and Escalation

**Team Lead:** ATLAS (Team Beta)
**Security Lead:** SENTINEL
**Plugin Development:** CIPHER
**Quality Assurance:** VALIDATOR
**Chief Orchestrator:** NEXUS

For issues or questions:
1. Check workflow logs
2. Review this documentation
3. Consult Makefile targets
4. Escalate to ATLAS
5. Critical security issues → SENTINEL

---

**Last Updated:** 2025-12-29
**Maintained by:** ATLAS - Team Lead & Senior DevOps Engineer
**Status:** Production Ready ✓
