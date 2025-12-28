# Security Guidelines for Developers

This document provides security guidelines for developing and maintaining the Webrana Infinimail application.

---

## Table of Contents

1. [Authentication & Authorization](#1-authentication--authorization)
2. [Input Validation](#2-input-validation)
3. [File Handling](#3-file-handling)
4. [Database Security](#4-database-security)
5. [API Security](#5-api-security)
6. [SMTP Security](#6-smtp-security)
7. [WebSocket Security](#7-websocket-security)
8. [Logging & Monitoring](#8-logging--monitoring)
9. [Secret Management](#9-secret-management)
10. [Docker Security](#10-docker-security)
11. [Dependency Management](#11-dependency-management)
12. [Code Review Checklist](#12-code-review-checklist)

---

## 1. Authentication & Authorization

### Implementation Guide

```go
// internal/api/middleware/auth.go

package middleware

import (
    "crypto/subtle"
    "log/slog"
    "os"
    "strings"

    "github.com/labstack/echo/v4"
)

// APIKeyAuth validates API key from Authorization header
func APIKeyAuth(logger *slog.Logger) echo.MiddlewareFunc {
    validAPIKey := os.Getenv("API_KEY")
    if validAPIKey == "" {
        logger.Warn("API_KEY not set - API is UNSECURED")
    }

    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Skip auth for health endpoints
            if strings.HasPrefix(c.Path(), "/health") {
                return next(c)
            }

            authHeader := c.Request().Header.Get("Authorization")
            if authHeader == "" {
                return echo.NewHTTPError(401, "missing authorization header")
            }

            token := strings.TrimPrefix(authHeader, "Bearer ")
            token = strings.TrimSpace(token)

            // Use constant-time comparison to prevent timing attacks
            if subtle.ConstantTimeCompare([]byte(token), []byte(validAPIKey)) != 1 {
                logger.Warn("invalid API key attempt",
                    slog.String("ip", c.RealIP()),
                    slog.String("path", c.Path()))
                return echo.NewHTTPError(401, "invalid API key")
            }

            return next(c)
        }
    }
}
```

### Best Practices

- Always use constant-time comparison for secrets
- Never log authentication tokens
- Implement rate limiting on auth endpoints
- Use secure session management
- Implement proper logout/token revocation

---

## 2. Input Validation

### Validation Package

```go
// internal/validator/validator.go

package validator

import (
    "errors"
    "net/mail"
    "regexp"
    "strings"
    "unicode/utf8"
)

var (
    ErrInvalidEmail     = errors.New("invalid email format")
    ErrInvalidDomain    = errors.New("invalid domain format")
    ErrInputTooLong     = errors.New("input exceeds maximum length")
    ErrInvalidCharacter = errors.New("input contains invalid characters")
)

var (
    domainRegex    = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$`)
    localPartRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
    safeStringRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// ValidateEmail validates email address format
func ValidateEmail(email string) error {
    email = strings.TrimSpace(strings.ToLower(email))

    if utf8.RuneCountInString(email) > 254 {
        return ErrInputTooLong
    }

    if _, err := mail.ParseAddress(email); err != nil {
        return ErrInvalidEmail
    }

    return nil
}

// ValidateDomain validates domain name format
func ValidateDomain(domain string) error {
    domain = strings.TrimSpace(strings.ToLower(domain))

    if len(domain) == 0 || len(domain) > 253 {
        return ErrInputTooLong
    }

    if !domainRegex.MatchString(domain) {
        return ErrInvalidDomain
    }

    return nil
}

// ValidateLocalPart validates email local part
func ValidateLocalPart(localPart string) error {
    localPart = strings.TrimSpace(strings.ToLower(localPart))

    if len(localPart) == 0 || len(localPart) > 64 {
        return ErrInputTooLong
    }

    if !localPartRegex.MatchString(localPart) {
        return ErrInvalidCharacter
    }

    return nil
}

// ValidatePagination validates and sanitizes pagination parameters
func ValidatePagination(limit, offset int) (int, int) {
    const maxLimit = 100
    const defaultLimit = 20

    if limit <= 0 {
        limit = defaultLimit
    }
    if limit > maxLimit {
        limit = maxLimit
    }

    if offset < 0 {
        offset = 0
    }

    return limit, offset
}

// SanitizeFilename removes dangerous characters from filename
func SanitizeFilename(filename string) string {
    // Remove path separators
    filename = strings.ReplaceAll(filename, "/", "_")
    filename = strings.ReplaceAll(filename, "\\", "_")
    filename = strings.ReplaceAll(filename, "..", "_")

    // Remove control characters
    filename = strings.Map(func(r rune) rune {
        if r < 32 || r == 127 {
            return -1
        }
        return r
    }, filename)

    // Limit length
    if utf8.RuneCountInString(filename) > 255 {
        filename = string([]rune(filename)[:255])
    }

    // Fallback for empty filename
    if filename == "" {
        return "unnamed"
    }

    return filename
}

// SanitizeString removes potentially dangerous characters
func SanitizeString(input string, maxLength int) string {
    // Remove control characters
    input = strings.Map(func(r rune) rune {
        if r < 32 || r == 127 {
            return -1
        }
        return r
    }, input)

    input = strings.TrimSpace(input)

    if maxLength > 0 && utf8.RuneCountInString(input) > maxLength {
        input = string([]rune(input)[:maxLength])
    }

    return input
}
```

### Usage in Handlers

```go
func (h *DomainHandler) Create(c echo.Context) error {
    var req CreateDomainRequest
    if err := c.Bind(&req); err != nil {
        return response.BadRequest(c, "invalid request body")
    }

    // ALWAYS validate input
    if err := validator.ValidateDomain(req.Name); err != nil {
        return response.BadRequest(c, err.Error())
    }

    domain := &models.Domain{
        Name:     strings.ToLower(strings.TrimSpace(req.Name)),
        IsActive: true,
    }
    // ...
}
```

---

## 3. File Handling

### Secure File Storage

```go
// internal/storage/file_storage.go

package storage

import (
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
)

var (
    ErrPathTraversal = errors.New("path traversal detected")
    ErrFileNotFound  = errors.New("file not found")
    ErrFileTooLarge  = errors.New("file exceeds size limit")
)

const (
    MaxFileSize = 25 * 1024 * 1024 // 25 MB
)

// Allowed MIME types for attachments
var AllowedMIMETypes = map[string]bool{
    "application/pdf":    true,
    "image/jpeg":         true,
    "image/png":          true,
    "image/gif":          true,
    "text/plain":         true,
    "application/zip":    true,
    "application/msword": true,
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
    "application/vnd.ms-excel": true,
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
}

// Blocked file extensions
var BlockedExtensions = map[string]bool{
    ".exe":  true,
    ".bat":  true,
    ".cmd":  true,
    ".com":  true,
    ".pif":  true,
    ".scr":  true,
    ".vbs":  true,
    ".js":   true,
    ".jar":  true,
    ".ps1":  true,
    ".sh":   true,
    ".bash": true,
}

// validatePath ensures path is within basePath (prevents traversal)
func (s *localStorage) validatePath(filePath string) (string, error) {
    // Clean the path
    cleanPath := filepath.Clean(filePath)

    // Prevent absolute paths
    if filepath.IsAbs(cleanPath) {
        return "", ErrPathTraversal
    }

    // Prevent path traversal
    if strings.Contains(cleanPath, "..") {
        return "", ErrPathTraversal
    }

    // Build full path
    fullPath := filepath.Join(s.basePath, cleanPath)

    // Get absolute paths for comparison
    absPath, err := filepath.Abs(fullPath)
    if err != nil {
        return "", fmt.Errorf("invalid file path: %w", err)
    }

    absBase, err := filepath.Abs(s.basePath)
    if err != nil {
        return "", fmt.Errorf("invalid base path: %w", err)
    }

    // Security check: ensure file is within allowed directory
    if !strings.HasPrefix(absPath+string(filepath.Separator),
        absBase+string(filepath.Separator)) {
        return "", ErrPathTraversal
    }

    return absPath, nil
}

// ValidateFile checks file extension and size
func ValidateFile(filename string, size int64) error {
    ext := strings.ToLower(filepath.Ext(filename))

    if BlockedExtensions[ext] {
        return fmt.Errorf("file extension %s is not allowed", ext)
    }

    if size > MaxFileSize {
        return ErrFileTooLarge
    }

    return nil
}

// Get retrieves a file by its path (SECURE VERSION)
func (s *localStorage) Get(filePath string) (io.ReadCloser, error) {
    fullPath, err := s.validatePath(filePath)
    if err != nil {
        return nil, err
    }

    file, err := os.Open(fullPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, ErrFileNotFound
        }
        return nil, fmt.Errorf("failed to open file: %w", err)
    }

    return file, nil
}

// Delete removes a file by its path (SECURE VERSION)
func (s *localStorage) Delete(filePath string) error {
    fullPath, err := s.validatePath(filePath)
    if err != nil {
        return err
    }

    if err := os.Remove(fullPath); err != nil {
        if os.IsNotExist(err) {
            return nil // File already gone, not an error
        }
        return fmt.Errorf("failed to delete file: %w", err)
    }

    return nil
}
```

### Secure Download Handler

```go
func (h *AttachmentHandler) Download(c echo.Context) error {
    id, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        return response.BadRequest(c, "invalid attachment ID")
    }

    attachment, err := h.attachmentRepo.GetByID(c.Request().Context(), uint(id))
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return response.NotFound(c, "attachment not found")
        }
        return response.InternalError(c, "failed to get attachment")
    }

    file, err := h.fileStorage.Get(attachment.FilePath)
    if err != nil {
        if errors.Is(err, storage.ErrPathTraversal) {
            return response.BadRequest(c, "invalid file path")
        }
        return response.InternalError(c, "failed to retrieve file")
    }
    defer file.Close()

    // Sanitize filename for Content-Disposition header
    safeFilename := validator.SanitizeFilename(attachment.Filename)

    // Set secure headers
    c.Response().Header().Set("Content-Type", attachment.ContentType)
    c.Response().Header().Set("X-Content-Type-Options", "nosniff")
    c.Response().Header().Set("Content-Disposition",
        mime.FormatMediaType("attachment", map[string]string{
            "filename": safeFilename,
        }))

    _, err = io.Copy(c.Response().Writer, file)
    return err
}
```

---

## 4. Database Security

### Parameterized Queries

```go
// CORRECT - Using GORM with parameterized queries
result := r.db.WithContext(ctx).Where("name = ?", name).First(&domain)

// CORRECT - Using GORM's struct-based queries
result := r.db.WithContext(ctx).Where(&models.Domain{Name: name}).First(&domain)

// WRONG - String concatenation (SQL Injection vulnerable!)
// query := fmt.Sprintf("SELECT * FROM domains WHERE name = '%s'", name)
// r.db.Raw(query).Scan(&domain)
```

### Connection Security

```go
// internal/database/database.go

func Connect(databaseURL string) (*gorm.DB, error) {
    // Validate SSL mode in production
    if os.Getenv("ENV") == "production" {
        if strings.Contains(databaseURL, "sslmode=disable") {
            return nil, errors.New("sslmode=disable not allowed in production")
        }
    }

    db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Warn),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Configure connection pool
    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get sql.DB: %w", err)
    }

    sqlDB.SetMaxOpenConns(25)
    sqlDB.SetMaxIdleConns(5)
    sqlDB.SetConnMaxLifetime(5 * time.Minute)
    sqlDB.SetConnMaxIdleTime(10 * time.Minute)

    return db, nil
}
```

---

## 5. API Security

### Security Headers Middleware

```go
// internal/api/middleware/security.go

func SecureHeaders() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            h := c.Response().Header()

            // Prevent clickjacking
            h.Set("X-Frame-Options", "DENY")

            // Prevent MIME sniffing
            h.Set("X-Content-Type-Options", "nosniff")

            // XSS Protection (legacy browsers)
            h.Set("X-XSS-Protection", "1; mode=block")

            // Content Security Policy
            h.Set("Content-Security-Policy",
                "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "+
                "img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

            // HSTS (only enable over HTTPS)
            if c.Scheme() == "https" {
                h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
            }

            // Referrer policy
            h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

            // Permissions policy
            h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

            return next(c)
        }
    }
}
```

### CORS Configuration

```go
// internal/api/middleware/middleware.go

func CORS() echo.MiddlewareFunc {
    allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
    if allowedOrigins == "" {
        allowedOrigins = "http://localhost:3000" // Dev default only
    }

    return middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins:     strings.Split(allowedOrigins, ","),
        AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE},
        AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
        AllowCredentials: true,
        MaxAge:           300,
    })
}
```

### Rate Limiting

```go
// internal/api/middleware/ratelimit.go

func RateLimiter(requestsPerSecond float64, burst int) echo.MiddlewareFunc {
    limiter := NewIPRateLimiter(rate.Limit(requestsPerSecond), burst)

    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            ip := c.RealIP()
            l := limiter.GetLimiter(ip)

            if !l.Allow() {
                return echo.NewHTTPError(429, map[string]string{
                    "error": "rate limit exceeded",
                    "retry_after": "60",
                })
            }

            return next(c)
        }
    }
}
```

---

## 6. SMTP Security

### Secure Configuration

```go
// cmd/server/main.go

smtpServer := gosmtp.NewServer(smtpBackend)
smtpServer.Addr = fmt.Sprintf(":%d", cfg.SMTPPort)
smtpServer.Domain = cfg.SMTPDomain
smtpServer.AllowInsecureAuth = false // NEVER allow insecure auth
smtpServer.MaxMessageBytes = 25 * 1024 * 1024
smtpServer.MaxRecipients = 50
smtpServer.ReadTimeout = 30 * time.Second
smtpServer.WriteTimeout = 30 * time.Second

// Enable TLS
if cfg.SMTPTLSEnabled {
    cert, err := tls.LoadX509KeyPair(cfg.SMTPCertFile, cfg.SMTPKeyFile)
    if err != nil {
        logger.Error("failed to load TLS certificate", slog.Any("error", err))
    } else {
        smtpServer.TLSConfig = &tls.Config{
            Certificates: []tls.Certificate{cert},
            MinVersion:   tls.VersionTLS12,
            CipherSuites: []uint16{
                tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
                tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
            },
        }
    }
}
```

### Email Parsing Limits

```go
// internal/smtp/session.go

const (
    MaxEmailSize       = 25 * 1024 * 1024  // 25 MB
    MaxAttachments     = 20
    MaxParseTime       = 30 * time.Second
    MaxRecursionDepth  = 10
)

func (s *Session) Data(r io.Reader) error {
    // Limit email size
    limitedReader := io.LimitReader(r, MaxEmailSize)

    // Parse with timeout
    ctx, cancel := context.WithTimeout(context.Background(), MaxParseTime)
    defer cancel()

    // ... parsing logic with context
}
```

---

## 7. WebSocket Security

### Origin Validation

```go
upgrader := websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")

        // Allow same-origin requests
        if origin == "" {
            return true
        }

        allowedOrigins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
        for _, allowed := range allowedOrigins {
            if strings.TrimSpace(allowed) == origin {
                return true
            }
        }

        logger.Warn("rejected websocket connection",
            slog.String("origin", origin),
            slog.String("remote_ip", r.RemoteAddr))
        return false
    },
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}
```

### Message Validation

```go
// internal/websocket/client.go

const (
    maxMessageSize = 512   // bytes
    pongWait       = 60 * time.Second
    pingPeriod     = (pongWait * 9) / 10
    writeWait      = 10 * time.Second
)

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
        // Validate message before processing
        if !isValidMessage(message) {
            continue
        }
        c.hub.broadcast <- message
    }
}
```

---

## 8. Logging & Monitoring

### Security Event Logging

```go
// internal/logger/security.go

type SecurityLogger struct {
    logger *slog.Logger
}

func NewSecurityLogger(logger *slog.Logger) *SecurityLogger {
    return &SecurityLogger{logger: logger}
}

func (s *SecurityLogger) AuthFailure(ip, path, reason string) {
    s.logger.Warn("authentication failure",
        slog.String("event", "auth_failure"),
        slog.String("ip", ip),
        slog.String("path", path),
        slog.String("reason", reason))
}

func (s *SecurityLogger) RateLimitExceeded(ip string) {
    s.logger.Warn("rate limit exceeded",
        slog.String("event", "rate_limit"),
        slog.String("ip", ip))
}

func (s *SecurityLogger) SuspiciousActivity(ip, activity string, details map[string]any) {
    attrs := []any{
        slog.String("event", "suspicious_activity"),
        slog.String("ip", ip),
        slog.String("activity", activity),
    }
    for k, v := range details {
        attrs = append(attrs, slog.Any(k, v))
    }
    s.logger.Warn("suspicious activity detected", attrs...)
}
```

### What NOT to Log

```go
// NEVER log these:
// - Passwords or API keys
// - Full credit card numbers
// - Social security numbers
// - Full email content (in most cases)
// - Session tokens
// - Private keys

// SAFE to log:
// - Usernames (not passwords)
// - IP addresses
// - Request paths
// - HTTP status codes
// - Timestamps
// - Error messages (without sensitive data)
```

---

## 9. Secret Management

### Environment Variables

```bash
# .env.secure.example

# Database (use strong password)
DATABASE_URL=postgres://infinimail:CHANGE_THIS_STRONG_PASSWORD@localhost:5432/infinimail?sslmode=verify-full

# API Security
API_KEY=generate_with_openssl_rand_hex_32
ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com

# Server
API_PORT=8080
SMTP_PORT=2525
SMTP_DOMAIN=mail.yourdomain.com
ENV=production

# TLS (optional but recommended)
SMTP_TLS_ENABLED=true
SMTP_CERT_FILE=/etc/ssl/certs/smtp.crt
SMTP_KEY_FILE=/etc/ssl/private/smtp.key

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Generating Secure Secrets

```bash
# Generate API key (32 bytes = 64 hex chars)
openssl rand -hex 32

# Generate database password
openssl rand -base64 32

# Generate JWT secret (if using JWT)
openssl rand -base64 64
```

---

## 10. Docker Security

### Secure Dockerfile

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o server ./cmd/server

# Final stage - minimal attack surface
FROM gcr.io/distroless/static-debian11:nonroot

WORKDIR /app
COPY --from=builder /app/server /app/server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER 65532:65532
EXPOSE 8080 2525
ENTRYPOINT ["/app/server"]
```

### Docker Compose Security

```yaml
services:
  backend:
    build: .
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - API_KEY=${API_KEY}
    restart: unless-stopped
```

---

## 11. Dependency Management

### Regular Scanning

```bash
# Check for vulnerabilities
govulncheck ./...

# Check for outdated packages
go list -m -u all

# Update dependencies
go get -u ./...
go mod tidy
```

### CI/CD Integration

```yaml
# .github/workflows/security.yml
name: Security Scan

on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

      - name: Run gosec
        run: |
          go install github.com/securego/gosec/v2/cmd/gosec@latest
          gosec ./...
```

---

## 12. Code Review Checklist

### Before Approving a PR

- [ ] **Authentication**: Is the endpoint properly protected?
- [ ] **Authorization**: Does it check user permissions?
- [ ] **Input Validation**: Is all user input validated?
- [ ] **SQL Injection**: Are queries parameterized?
- [ ] **Path Traversal**: Are file paths validated?
- [ ] **XSS**: Is output properly escaped?
- [ ] **CSRF**: Are state-changing operations protected?
- [ ] **Logging**: No secrets in logs?
- [ ] **Error Handling**: No sensitive info in error messages?
- [ ] **Rate Limiting**: Is abuse prevention in place?
- [ ] **Dependencies**: Are new deps from trusted sources?
- [ ] **Tests**: Are security tests included?

---

## Quick Reference

### Common Vulnerabilities to Avoid

| Vulnerability | Prevention |
|--------------|------------|
| SQL Injection | Use GORM parameterized queries |
| XSS | Sanitize output, use CSP |
| CSRF | Token validation |
| Path Traversal | Validate file paths |
| Command Injection | Never use exec with user input |
| Insecure Deserialization | Validate JSON structure |
| Sensitive Data Exposure | Encrypt, don't log secrets |

### Emergency Contacts

- Security issues: security@webrana.com
- On-call engineer: [Internal contact]

---

**Document Version:** 1.0
**Last Updated:** 2025-12-29
