# Security Scanning Quick Reference

This guide provides quick commands for running security scans on the Infinimail Backend.

## Table of Contents
- [Quick Start](#quick-start)
- [CI/CD Pipeline](#cicd-pipeline)
- [Local Scanning](#local-scanning)
- [Tool Installation](#tool-installation)
- [Common Issues](#common-issues)

## Quick Start

### Run All Security Scans
```bash
# Using Makefile
make -f Makefile.security security-scan

# Or run individual scans
make -f Makefile.security gosec-scan
make -f Makefile.security trivy-scan
make -f Makefile.security gitleaks-scan
```

### Install Security Tools
```bash
# All tools at once
make -f Makefile.security security-install

# Or install individually (see Tool Installation section)
```

### Install Pre-commit Hooks
```bash
# Install git hooks for automatic security checks
chmod +x scripts/install-hooks.sh
./scripts/install-hooks.sh
```

## CI/CD Pipeline

### Automated Security Scanning

The `.github/workflows/security.yml` workflow runs automatically on:
- Every push to `main` or `develop` branches
- Every pull request
- Daily at 2 AM UTC (scheduled)
- Manual trigger via GitHub Actions UI

### Scanners in CI/CD

| Scanner | What it Scans | Severity |
|---------|---------------|----------|
| **gosec** | Go source code for security issues | All |
| **CodeQL** | Advanced semantic analysis | High/Critical |
| **Trivy (Container)** | Docker image vulnerabilities | High/Critical/Medium |
| **Trivy (Filesystem)** | Dependencies in go.mod, secrets, misconfig | High/Critical/Medium |
| **Dependency Review** | New vulnerabilities in PRs | High+ |
| **Gitleaks** | Hardcoded secrets in code/history | All |
| **Nancy** | Go dependency vulnerabilities (OSS Index) | All |

### Viewing Results

1. **GitHub Security Tab**:
   - Navigate to: Repository → Security → Code scanning
   - View all alerts from gosec, CodeQL, and Trivy

2. **Pull Request Checks**:
   - See pass/fail status for each scanner
   - Click "Details" to view specific findings

3. **Workflow Artifacts**:
   - Download SARIF reports from Actions → Security Scanning → Artifacts

## Local Scanning

### 1. gosec - Go Security Scanner

Detects: SQL injection, command injection, path traversal, weak crypto, hardcoded secrets

```bash
# Scan all packages
gosec ./...

# Exclude test directories
gosec -exclude-dir=tests ./...

# JSON output for parsing
gosec -fmt=json -out=gosec-report.json ./...

# Scan specific security rules
gosec -include=G201,G202,G204,G304 ./...

# Check specific file
gosec internal/storage/file_storage.go
```

**Key Security Rules**:
- `G101`: Hardcoded credentials
- `G201/G202`: SQL injection
- `G204`: Command injection
- `G304`: Path traversal
- `G401/G505`: Weak crypto (MD5, SHA1)
- `G404`: Insecure random number generator

### 2. Trivy - Vulnerability Scanner

Detects: CVEs in dependencies, container images, secrets, misconfigurations

```bash
# Scan filesystem (go.mod dependencies)
trivy fs .

# Only high and critical
trivy fs --severity HIGH,CRITICAL .

# Scan for secrets and misconfigurations
trivy fs --scanners vuln,secret,misconfig .

# Scan Docker image
docker build -t infinimail-backend:test .
trivy image infinimail-backend:test

# Scan with JSON output
trivy fs --format json --output trivy-report.json .
```

**What Trivy Finds**:
- CVEs in Go dependencies (go.mod)
- CVEs in Alpine base image
- Hardcoded secrets (API keys, passwords)
- Kubernetes misconfigurations
- Dockerfile security issues

### 3. Gitleaks - Secret Scanner

Detects: API keys, passwords, tokens, private keys in code and git history

```bash
# Scan current files
gitleaks detect --source . -v

# Scan only staged files (pre-commit)
gitleaks protect --staged -v

# Scan entire git history
gitleaks detect --source . --log-opts="--all" -v

# Generate JSON report
gitleaks detect --source . --report-path gitleaks-report.json

# Use custom rules
gitleaks detect --source . --config .gitleaks.toml
```

**What Gitleaks Finds**:
- AWS credentials
- API keys (generic, Stripe, etc.)
- Database passwords
- Private SSH/PGP keys
- JWT tokens
- OAuth tokens

### 4. golangci-lint - Comprehensive Linter

Detects: Security issues, code quality problems, suspicious patterns

```bash
# Run with security config
golangci-lint run --config=.golangci.yml

# Only security linters
golangci-lint run --enable=gosec,errcheck,govet,staticcheck

# Auto-fix issues
golangci-lint run --fix

# Run on changed files only
golangci-lint run --new-from-rev=HEAD~1

# Verbose output
golangci-lint run -v
```

**Security Linters Enabled**:
- `gosec`: Security vulnerabilities
- `errcheck`: Unchecked errors (can cause panics)
- `govet`: Suspicious constructs
- `staticcheck`: Advanced static analysis
- `bodyclose`: HTTP response body leaks
- `sqlclosecheck`: Database connection leaks
- `noctx`: HTTP requests without context

### 5. Nancy - Dependency Scanner

Detects: Known vulnerabilities in Go dependencies (OSS Index)

```bash
# Scan dependencies
go list -json -deps ./... | nancy sleuth

# Skip update check (faster)
go list -json -deps ./... | nancy sleuth --skip-update-check

# Exclude dev dependencies
go list -json -deps -tags=!integration,!e2e ./... | nancy sleuth
```

### 6. govulncheck - Official Go Vulnerability Scanner

Detects: Vulnerabilities from Go's official vulnerability database

```bash
# Scan project
govulncheck ./...

# JSON output
govulncheck -json ./... > govulncheck-results.json

# Scan specific package
govulncheck ./internal/...
```

## Tool Installation

### macOS (Homebrew)

```bash
# gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# golangci-lint
brew install golangci-lint

# Trivy
brew install aquasecurity/trivy/trivy

# Gitleaks
brew install gitleaks

# Nancy
go install github.com/sonatype-nexus-community/nancy@latest

# govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# pre-commit framework (optional)
brew install pre-commit
```

### Linux (Ubuntu/Debian)

```bash
# gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Trivy
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
sudo apt-get update
sudo apt-get install trivy

# Gitleaks
# Download latest release from: https://github.com/gitleaks/gitleaks/releases
wget https://github.com/gitleaks/gitleaks/releases/download/v8.18.1/gitleaks_8.18.1_linux_x64.tar.gz
tar -xzf gitleaks_8.18.1_linux_x64.tar.gz
sudo mv gitleaks /usr/local/bin/

# Nancy
go install github.com/sonatype-nexus-community/nancy@latest

# govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### Windows

```bash
# gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# golangci-lint
# Download from: https://github.com/golangci/golangci-lint/releases

# Trivy (Chocolatey)
choco install trivy

# Gitleaks
# Download from: https://github.com/gitleaks/gitleaks/releases

# Nancy
go install github.com/sonatype-nexus-community/nancy@latest

# govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### Verify Installation

```bash
# Check all tools are installed
gosec --version
golangci-lint --version
trivy --version
gitleaks version
nancy --version
govulncheck -version
```

## Common Issues

### False Positives

If a scanner reports a false positive:

1. **For gosec**: Add `// #nosec` comment with explanation
   ```go
   // #nosec G304 -- file path is validated by validatePath()
   file, err := os.Open(filePath)
   ```

2. **For Trivy**: Add to `.trivyignore`
   ```
   # CVE-2024-12345 - False positive, not exploitable in our context
   CVE-2024-12345
   ```

3. **For Gitleaks**: Add to `.gitleaksignore`
   ```
   # Test credentials in example file
   abc123:.env.example:generic-api-key
   ```

### Performance Issues

```bash
# Run only critical scans
make -f Makefile.security security-quick  # gosec + gitleaks only

# Exclude test files
gosec -exclude-dir=tests ./...

# Limit Trivy severity
trivy fs --severity CRITICAL .
```

### CI/CD Failures

If security workflow fails:

1. **Check Security Tab**: View specific findings
2. **Run Locally**: Reproduce the issue
   ```bash
   make -f Makefile.security security-scan
   ```
3. **Review Artifacts**: Download SARIF reports from workflow
4. **Fix or Suppress**: Fix the issue or add to ignore files (with justification)

### Updating Vulnerability Databases

```bash
# Update Trivy database
trivy image --download-db-only

# Update Nancy cache
nancy --clean-cache

# Update govulncheck database
govulncheck -version  # Auto-updates on run
```

## Best Practices

1. **Run Before Committing**
   ```bash
   # Install pre-commit hooks
   ./scripts/install-hooks.sh

   # Or run manually
   make -f Makefile.security security-quick
   ```

2. **Review Findings Regularly**
   - Check GitHub Security tab weekly
   - Review daily scheduled scan results
   - Monitor Dependabot alerts

3. **Update Dependencies**
   ```bash
   # Update Go dependencies
   go get -u ./...
   go mod tidy

   # Then scan
   make -f Makefile.security security-scan
   ```

4. **Never Commit Secrets**
   - Use environment variables
   - Add secrets to `.gitignore`
   - Rotate exposed secrets immediately

5. **Document Suppressions**
   - Always add comments explaining why
   - Set expiration dates for temporary suppressions
   - Review ignore files quarterly

## Quick Reference Commands

```bash
# Complete security scan
make -f Makefile.security security-scan

# Quick scan (fast)
make -f Makefile.security security-quick

# CI-friendly (fails on findings)
make -f Makefile.security security-ci

# Clean artifacts
make -f Makefile.security security-clean

# Install tools
make -f Makefile.security security-install

# Install pre-commit hooks
./scripts/install-hooks.sh

# Help
make -f Makefile.security security-help
```

## Resources

- [gosec Rules](https://github.com/securego/gosec#available-rules)
- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
- [Gitleaks Documentation](https://github.com/gitleaks/gitleaks)
- [golangci-lint Linters](https://golangci-lint.run/usage/linters/)
- [Go Security Best Practices](https://github.com/OWASP/Go-SCP)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)

---

**For detailed security policy and vulnerability reporting, see [SECURITY.md](SECURITY.md)**
