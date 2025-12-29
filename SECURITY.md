# Security Policy

## Overview

Security is a top priority for Infinimail Backend. This document outlines our security practices, how to run security scans locally, and how to report vulnerabilities.

## Table of Contents

- [Security Scanning in CI/CD](#security-scanning-in-cicd)
- [Running Security Scans Locally](#running-security-scans-locally)
- [Security Best Practices](#security-best-practices)
- [Vulnerability Reporting](#vulnerability-reporting)
- [Security Features](#security-features)
- [Known Security Considerations](#known-security-considerations)

## Security Scanning in CI/CD

Our GitHub Actions workflow automatically runs comprehensive security scans on every push and pull request:

### Automated Scanners

1. **gosec** - Go source code security scanner
   - Detects: SQL injection, command injection, path traversal, weak crypto
   - Runs on: Every commit
   - Results: Available in Security tab

2. **CodeQL** - Advanced semantic analysis
   - Detects: Complex vulnerabilities, data flow issues, injection attacks
   - Runs on: Every commit + scheduled daily
   - Results: GitHub Security tab

3. **Trivy** - Container vulnerability scanner
   - Scans: Docker images for CVEs in Alpine base + dependencies
   - Severity: CRITICAL, HIGH, MEDIUM
   - Runs on: Every commit

4. **Trivy Filesystem** - Dependency vulnerability scanner
   - Scans: go.mod, go.sum for known CVEs
   - Detects: Vulnerable dependencies, secrets in code, misconfigurations
   - Runs on: Every commit

5. **Dependency Review** - PR-only scanner
   - Detects: New vulnerable dependencies introduced in PRs
   - Blocks: GPL-2.0, GPL-3.0, AGPL-3.0 licenses
   - Runs on: Pull requests only

6. **Gitleaks** - Secret scanning
   - Detects: API keys, passwords, tokens in code/history
   - Runs on: Every commit + full history scan
   - Prevention: Pre-commit hook recommended

7. **Nancy** - Go dependency scanner
   - OSS Index integration for vulnerability intelligence
   - Runs on: Every commit

### Viewing Security Results

Security scan results are available in multiple places:

1. **GitHub Security Tab**: `https://github.com/YOUR_ORG/webrana-infinimail-backend/security`
   - Code scanning alerts (gosec, CodeQL, Trivy)
   - Dependabot alerts
   - Secret scanning alerts

2. **Pull Request Checks**: Each PR shows pass/fail status for all scanners

3. **Actions Artifacts**: Download detailed SARIF reports from workflow runs

## Running Security Scans Locally

### Prerequisites

Install required tools:

```bash
# gosec - Go security scanner
go install github.com/securego/gosec/v2/cmd/gosec@latest

# golangci-lint - Multi-purpose linter with security checks
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Trivy - Container and dependency scanner
# macOS
brew install aquasecurity/trivy/trivy

# Linux
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
sudo apt-get update
sudo apt-get install trivy

# Windows
choco install trivy

# Gitleaks - Secret scanner
# macOS
brew install gitleaks

# Linux/Windows - Download from https://github.com/gitleaks/gitleaks/releases

# Nancy - Dependency scanner
go install github.com/sonatype-nexus-community/nancy@latest
```

### Quick Security Scan

Run all security checks with one command:

```bash
make security-scan
```

Or add this to your `Makefile`:

```makefile
.PHONY: security-scan
security-scan: gosec-scan trivy-scan gitleaks-scan nancy-scan
	@echo "✅ All security scans complete!"

.PHONY: gosec-scan
gosec-scan:
	@echo "Running gosec..."
	gosec -exclude-dir=tests ./...

.PHONY: trivy-scan
trivy-scan:
	@echo "Running Trivy filesystem scan..."
	trivy fs --severity HIGH,CRITICAL .

.PHONY: gitleaks-scan
gitleaks-scan:
	@echo "Running Gitleaks..."
	gitleaks detect --source . -v

.PHONY: nancy-scan
nancy-scan:
	@echo "Running Nancy..."
	go list -json -deps ./... | nancy sleuth
```

### Individual Security Scans

#### 1. gosec - Source Code Security

```bash
# Scan all packages
gosec ./...

# Scan with detailed output
gosec -fmt=json -out=gosec-report.json ./...

# Scan excluding test files
gosec -exclude-dir=tests ./...

# Check specific security rules
gosec -include=G201,G202,G204 ./...  # SQL injection, command injection
```

**Key checks:**
- G201/G202: SQL injection via string concatenation
- G204: Command injection via os/exec
- G304: Path traversal
- G401/G505: Weak cryptographic algorithms
- G101: Hardcoded credentials

#### 2. golangci-lint - Comprehensive Linting

```bash
# Run with security-focused config
golangci-lint run --config=.golangci.yml

# Run only security linters
golangci-lint run --enable=gosec,errcheck,govet,staticcheck

# Fix auto-fixable issues
golangci-lint run --fix

# Verbose output
golangci-lint run -v
```

**Security linters enabled:**
- gosec: Security issues
- errcheck: Unchecked errors (potential panics)
- govet: Suspicious constructs
- staticcheck: Advanced static analysis
- bodyclose: HTTP response body leaks
- sqlclosecheck: Database connection leaks

#### 3. Trivy - Container & Dependency Scanning

```bash
# Scan Docker image
docker build -t infinimail-backend:test .
trivy image infinimail-backend:test

# Scan only HIGH and CRITICAL vulnerabilities
trivy image --severity HIGH,CRITICAL infinimail-backend:test

# Scan filesystem (dependencies in go.mod)
trivy fs .

# Scan for secrets and misconfigurations
trivy fs --scanners vuln,secret,misconfig .

# Generate detailed report
trivy image --format json --output trivy-report.json infinimail-backend:test
```

**What Trivy detects:**
- CVEs in dependencies (go.mod)
- CVEs in base Docker image (Alpine)
- Secrets in code/config files
- Kubernetes misconfigurations
- IaC security issues

#### 4. Gitleaks - Secret Scanning

```bash
# Scan current files
gitleaks detect --source . -v

# Scan entire git history
gitleaks detect --source . --log-opts="--all" -v

# Generate report
gitleaks detect --source . --report-path gitleaks-report.json
```

**What Gitleaks detects:**
- API keys and tokens
- Passwords
- Private keys (SSH, PGP, etc.)
- AWS credentials
- Database connection strings

#### 5. Nancy - Dependency Vulnerabilities

```bash
# Scan dependencies
go list -json -deps ./... | nancy sleuth

# Skip update check (for CI)
go list -json -deps ./... | nancy sleuth --skip-update-check
```

### Pre-commit Security Checks

Prevent security issues before committing:

#### Option 1: Manual Pre-commit Script

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash

echo "Running pre-commit security checks..."

# Run gitleaks
echo "🔍 Scanning for secrets..."
if ! gitleaks protect --staged -v; then
    echo "❌ Gitleaks found potential secrets. Commit blocked."
    exit 1
fi

# Run gosec on changed files
echo "🔍 Running gosec..."
if ! gosec ./...; then
    echo "⚠️  gosec found security issues. Review before committing."
    # Uncomment to block commit:
    # exit 1
fi

# Run golangci-lint
echo "🔍 Running golangci-lint..."
if ! golangci-lint run --new-from-rev=HEAD~1; then
    echo "⚠️  Linter found issues in new code. Review before committing."
    # Uncomment to block commit:
    # exit 1
fi

echo "✅ Pre-commit checks passed!"
exit 0
```

Make it executable:

```bash
chmod +x .git/hooks/pre-commit
```

#### Option 2: pre-commit Framework (Recommended)

Install pre-commit framework:

```bash
# macOS
brew install pre-commit

# Linux
pip install pre-commit
```

Create `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.18.1
    hooks:
      - id: gitleaks

  - repo: https://github.com/golangci/golangci-lint
    rev: v1.55.2
    hooks:
      - id: golangci-lint
        args: [--config=.golangci.yml]

  - repo: local
    hooks:
      - id: gosec
        name: gosec
        entry: gosec
        args: [./...]
        language: system
        pass_filenames: false
```

Install hooks:

```bash
pre-commit install
```

## Security Best Practices

### Code Security

1. **Input Validation**
   - Validate all user inputs
   - Use GORM parameterized queries (NEVER string concatenation)
   - Sanitize file paths before filesystem operations

   ```go
   // BAD - SQL injection risk
   db.Raw("SELECT * FROM users WHERE email = '" + userEmail + "'")

   // GOOD - Parameterized query
   db.Where("email = ?", userEmail).Find(&users)
   ```

2. **Path Traversal Prevention**
   - Always use `filepath.Clean()` and `filepath.Join()`
   - Validate paths are within expected directories
   - See `internal/storage/file_storage.go` for reference implementation

   ```go
   // BAD - Path traversal risk
   filePath := basePath + "/" + userInput

   // GOOD - Safe path handling
   filePath := filepath.Join(basePath, filepath.Clean(userInput))
   if !strings.HasPrefix(filePath, basePath) {
       return ErrPathTraversal
   }
   ```

3. **Secret Management**
   - NEVER hardcode secrets in code
   - Use environment variables for all credentials
   - Use `crypto/rand` instead of `math/rand` for security
   - Use `subtle.ConstantTimeCompare()` for secret comparison

   ```go
   // BAD - Timing attack vulnerable
   if apiKey == userKey {
       // ...
   }

   // GOOD - Constant time comparison
   if subtle.ConstantTimeCompare([]byte(apiKey), []byte(userKey)) == 1 {
       // ...
   }
   ```

4. **Error Handling**
   - Always check errors (use errcheck linter)
   - Don't expose internal errors to users
   - Log security events without sensitive data

   ```go
   // BAD - Unchecked error
   file.Close()

   // GOOD - Checked error
   if err := file.Close(); err != nil {
       logger.Error("failed to close file", "error", err)
   }
   ```

### Dependency Security

1. **Regular Updates**
   ```bash
   # Update dependencies
   go get -u ./...
   go mod tidy

   # Check for vulnerabilities
   go list -json -deps ./... | nancy sleuth
   ```

2. **Minimal Dependencies**
   - Only add dependencies when necessary
   - Review security of new dependencies
   - Check license compatibility

3. **Dependency Pinning**
   - Use exact versions in go.mod
   - Review go.sum for integrity

### Container Security

1. **Minimal Base Image**
   - Use Alpine Linux (current)
   - Consider distroless for production

2. **Non-root User**
   - Run container as non-root user (TODO)
   - Set read-only filesystem where possible

3. **Vulnerability Scanning**
   ```bash
   docker build -t infinimail-backend .
   trivy image infinimail-backend
   ```

### Production Security Checklist

- [ ] `API_KEY` set to strong random value (32+ chars)
- [ ] `DATABASE_URL` uses `sslmode=verify-full`
- [ ] `ALLOWED_ORIGINS` set to specific domains (no wildcards)
- [ ] `APP_ENV=production`
- [ ] `SMTP_ALLOW_INSECURE=false`
- [ ] TLS certificates configured for SMTP
- [ ] Rate limiting enabled and tuned
- [ ] Log aggregation configured (no secrets in logs)
- [ ] Regular security updates scheduled
- [ ] Database backups encrypted
- [ ] File permissions: `.env` is 600
- [ ] Firewall configured (only 8080, 25/2525 exposed)

## Vulnerability Reporting

### Supported Versions

We actively support security updates for:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | ✅ Yes             |
| < 1.0   | ❌ No              |

### Reporting a Vulnerability

**DO NOT** create public GitHub issues for security vulnerabilities.

Instead, please report security vulnerabilities to:

**Email**: security@webrana.com (or your security contact)

**What to include:**

1. **Description**: Clear description of the vulnerability
2. **Impact**: Potential security impact (CVSS score if available)
3. **Reproduction**: Step-by-step instructions to reproduce
4. **Environment**: Go version, OS, deployment method
5. **Proof of Concept**: Code or curl commands demonstrating the issue
6. **Suggested Fix**: If you have a patch or mitigation

### Response Timeline

- **Initial Response**: Within 48 hours
- **Triage**: Within 7 days
- **Fix Development**: Depends on severity
  - Critical: 7 days
  - High: 14 days
  - Medium: 30 days
  - Low: Next release
- **Public Disclosure**: After fix is released

### Security Advisory Process

1. **Receive Report**: Acknowledge receipt within 48 hours
2. **Triage**: Assess severity and impact
3. **Develop Fix**: Create patch and test thoroughly
4. **Notify Reporter**: Share fix for validation
5. **Release**: Deploy security update
6. **Publish Advisory**: GitHub Security Advisory after fix is deployed
7. **Credit Reporter**: Public acknowledgment (if desired)

## Security Features

### Current Security Features

1. **Authentication**
   - API key authentication (production)
   - Constant-time comparison (prevents timing attacks)
   - Configurable auth bypass for development

2. **Input Validation**
   - Email address validation
   - File extension blocklist
   - File size limits (25 MB)
   - Path traversal prevention

3. **CORS Protection**
   - Configurable origin whitelist
   - No wildcard (*) in production

4. **Rate Limiting**
   - Per-IP rate limiting
   - Configurable limits (10 req/s default)
   - Burst protection

5. **Security Headers**
   - X-Content-Type-Options: nosniff
   - X-Frame-Options: DENY
   - X-XSS-Protection: 1; mode=block
   - Content-Security-Policy
   - Strict-Transport-Security (HTTPS)

6. **Database Security**
   - GORM parameterized queries
   - SQL injection prevention
   - Connection pooling
   - SSL/TLS support

7. **File Storage Security**
   - Path traversal prevention
   - Extension blocklist (.exe, .bat, .sh, etc.)
   - Size limits
   - Unique filename generation (UUID)

8. **SMTP Security**
   - TLS/SSL support
   - Message size limits
   - Recipient limits
   - Configurable timeouts

### Planned Security Enhancements

- [ ] mTLS for API authentication
- [ ] JWT token support
- [ ] Enhanced logging (SIEM integration)
- [ ] Honeypot endpoints for intrusion detection
- [ ] Automated security testing in CI/CD
- [ ] Container runtime security (Falco)
- [ ] Network policies (Kubernetes)

## Known Security Considerations

### Current Limitations

1. **Email Delivery**: No SPF/DKIM validation (by design for temp mail)
2. **Storage**: Local filesystem (consider S3 for production)
3. **Rate Limiting**: In-memory (not distributed)
4. **Session Management**: Stateless API (no session fixation risk)

### Accepted Risks

1. **Auto-provisioning**: Mailboxes auto-created (feature, not bug)
2. **Public Email Access**: Anyone can receive to any address (intended)
3. **No Email Encryption**: Emails stored in plain text (consider pgp)

### Mitigations

- Regular security scanning (automated)
- Dependency updates (Dependabot)
- Container scanning (Trivy)
- Code review required for all changes
- Penetration testing (recommended annually)

## Security Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://github.com/OWASP/Go-SCP)
- [CWE Top 25](https://cwe.mitre.org/top25/)
- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
- [gosec Rules](https://github.com/securego/gosec#available-rules)

## Contact

For security questions or concerns:
- Email: security@webrana.com
- GitHub Security Tab: [Security Advisories](https://github.com/YOUR_ORG/webrana-infinimail-backend/security)

---

**Last Updated**: 2025-12-29
**Review Schedule**: Quarterly
**Next Review**: 2025-03-29
