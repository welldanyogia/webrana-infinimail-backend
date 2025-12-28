# Webrana Infinimail Backend

A robust, self-hosted temporary email service backend built with Go. Infinimail provides a complete solution for receiving, storing, and managing temporary emails with real-time notifications.

## Table of Contents

- [Features](#features)
- [Architecture Overview](#architecture-overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the Application](#running-the-application)
- [API Documentation](#api-documentation)
- [Docker Deployment](#docker-deployment)
- [Security Considerations](#security-considerations)
- [Development](#development)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features

### Core Features
- **Multi-Protocol Support**: SMTP server for receiving emails + REST API for email management
- **Real-time Notifications**: WebSocket support for instant email notifications
- **Multi-Domain Support**: Manage multiple domains with catch-all functionality
- **Auto-Provisioning**: Automatically create mailboxes when emails arrive
- **File Attachments**: Full support for email attachments with secure storage
- **Persistent Storage**: PostgreSQL database for permanent email storage

### Security Features
- API key authentication
- Rate limiting (configurable per IP)
- CORS protection with whitelist
- Security headers (CSP, HSTS, X-Frame-Options, etc.)
- Input validation and sanitization
- Secure WebSocket connections
- TLS/SSL support for SMTP

### Developer Features
- Comprehensive test suite (unit, integration, E2E)
- Health check endpoints
- Structured JSON logging
- Docker support with multi-stage builds
- Hot reload support for development
- Graceful shutdown handling

## Architecture Overview

```
┌─────────────┐
│   Internet  │
└──────┬──────┘
       │
       ├─────────► SMTP (Port 25/2525)
       │              │
       │              ▼
       │         ┌─────────────────┐
       │         │  SMTP Handler   │
       │         │  - Receive Mail │
       │         │  - Parse MIME   │
       │         │  - Store Email  │
       │         └────────┬────────┘
       │                  │
       └─────────► HTTP/WS (Port 8080)
                        │
                        ▼
                  ┌──────────────────┐
                  │   Echo Router    │
                  │   - REST API     │
                  │   - WebSocket    │
                  │   - Middleware   │
                  └────────┬─────────┘
                           │
       ┌───────────────────┼───────────────────┐
       │                   │                   │
       ▼                   ▼                   ▼
  ┌─────────┐      ┌──────────────┐     ┌──────────┐
  │ Domains │      │   Mailboxes  │     │ Messages │
  └─────────┘      └──────────────┘     └──────────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                           ▼
                   ┌───────────────┐
                   │  PostgreSQL   │
                   │   Database    │
                   └───────────────┘
```

### Technology Stack

- **Language**: Go 1.24
- **Web Framework**: [Echo v4](https://echo.labstack.com/)
- **SMTP Library**: [go-smtp](https://github.com/emersion/go-smtp)
- **Email Parser**: [enmime](https://github.com/jhillyerd/enmime)
- **Database**: PostgreSQL 16
- **ORM**: [GORM v2](https://gorm.io/)
- **WebSocket**: [Gorilla WebSocket](https://github.com/gorilla/websocket)
- **Testing**: testify, testcontainers-go

## Prerequisites

Before installing Infinimail Backend, ensure you have the following installed:

### Required
- **Go**: 1.24 or higher ([Download](https://golang.org/dl/))
- **PostgreSQL**: 16 or higher ([Download](https://www.postgresql.org/download/))
- **Git**: For cloning the repository

### Optional (for Docker deployment)
- **Docker**: 20.10 or higher ([Download](https://www.docker.com/get-started))
- **Docker Compose**: v2.0 or higher

### System Requirements
- **RAM**: Minimum 512 MB (2 GB recommended for production)
- **Disk Space**: 100 MB for application + storage for attachments
- **Network**: Ports 8080 (API) and 25/2525 (SMTP) must be available

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/welldanyogia/webrana-infinimail-backend.git
cd webrana-infinimail-backend
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Set Up PostgreSQL Database

#### Option A: Using Docker (Recommended for Development)

```bash
docker run -d \
  --name infinimail-postgres \
  -e POSTGRES_USER=infinimail \
  -e POSTGRES_PASSWORD=infinimail \
  -e POSTGRES_DB=infinimail \
  -p 5432:5432 \
  postgres:16-alpine
```

#### Option B: Manual PostgreSQL Setup

1. Create a database and user:

```sql
CREATE DATABASE infinimail;
CREATE USER infinimail WITH ENCRYPTED PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE infinimail TO infinimail;
```

2. Update the connection string in your `.env` file (see Configuration section)

### 4. Configure Environment Variables

Create a `.env` file in the project root:

```bash
cp .env.example .env
```

Edit `.env` with your settings (see [Configuration](#configuration) section for details)

### 5. Build the Application

```bash
# Using Make
make build

# Or using Go directly
go build -o bin/server ./cmd/server
```

## Configuration

Configuration is managed through environment variables. Create a `.env` file based on the examples provided.

### Development Configuration (.env.example)

```bash
# Database Configuration
DATABASE_URL=postgres://infinimail:infinimail@localhost:5432/infinimail?sslmode=disable

# Server Ports
API_PORT=8080          # HTTP API server port
SMTP_PORT=2525         # SMTP server port (use 2525 for dev, 25 for production)

# Features
AUTO_PROVISIONING_ENABLED=true  # Auto-create mailboxes when emails arrive

# Storage
ATTACHMENT_STORAGE_PATH=./attachments  # Directory for email attachments

# Logging
LOG_LEVEL=info  # Options: debug, info, warn, error

# Security Configuration (Development)
API_KEY=                          # Leave empty for development (disables auth)
ALLOWED_ORIGINS=http://localhost:3000  # Comma-separated CORS origins
APP_ENV=development

# Rate Limiting
RATE_LIMIT_REQUESTS=10   # Requests per second per IP
RATE_LIMIT_BURST=20      # Burst capacity

# SMTP Security
SMTP_ALLOW_INSECURE=true          # Allow insecure connections (dev only)
SMTP_MAX_MESSAGE_SIZE=26214400    # 25 MB in bytes
SMTP_MAX_RECIPIENTS=100
SMTP_READ_TIMEOUT=60s
SMTP_WRITE_TIMEOUT=60s
```

### Production Configuration (.env.secure.example)

For production deployments, refer to `.env.secure.example` which includes:

- **Required**: API key authentication
- **Required**: Strict CORS configuration (no wildcards)
- **Required**: SSL/TLS for database connections
- **Required**: Enhanced rate limiting
- **Required**: Secure SMTP settings

**Production Checklist**:
- [ ] Set `APP_ENV=production`
- [ ] Generate strong `API_KEY` (min 32 characters): `openssl rand -hex 32`
- [ ] Configure `ALLOWED_ORIGINS` (no wildcards)
- [ ] Use `sslmode=verify-full` in `DATABASE_URL`
- [ ] Set `SMTP_ALLOW_INSECURE=false`
- [ ] Use port 25 for SMTP
- [ ] Configure TLS certificates for SMTP
- [ ] Set appropriate file permissions: `chmod 600 .env`

### Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | - | PostgreSQL connection string |
| `API_PORT` | No | 8080 | HTTP server port |
| `SMTP_PORT` | No | 2525 | SMTP server port |
| `AUTO_PROVISIONING_ENABLED` | No | true | Auto-create mailboxes |
| `ATTACHMENT_STORAGE_PATH` | No | ./attachments | Attachment storage directory |
| `LOG_LEVEL` | No | info | Logging level (debug/info/warn/error) |
| `API_KEY` | Production | - | API authentication key |
| `ALLOWED_ORIGINS` | Production | - | CORS allowed origins (comma-separated) |
| `APP_ENV` | No | development | Environment (development/staging/production) |
| `RATE_LIMIT_REQUESTS` | No | 10 | Requests per second per IP |
| `RATE_LIMIT_BURST` | No | 20 | Rate limiter burst capacity |
| `SMTP_ALLOW_INSECURE` | No | false | Allow insecure SMTP connections |
| `SMTP_MAX_MESSAGE_SIZE` | No | 26214400 | Max email size in bytes |
| `SMTP_MAX_RECIPIENTS` | No | 100 | Max recipients per email |
| `SMTP_READ_TIMEOUT` | No | 60s | SMTP read timeout |
| `SMTP_WRITE_TIMEOUT` | No | 60s | SMTP write timeout |

## Running the Application

### Development Mode

#### Using Make (Recommended)

```bash
# Run with hot reload (if you have air installed)
make run

# Or run directly
make build && ./bin/server
```

#### Using Go

```bash
go run ./cmd/server
```

The application will start with:
- **HTTP API**: `http://localhost:8080`
- **SMTP Server**: `localhost:2525`
- **WebSocket**: `ws://localhost:8080/ws`

### Production Mode

1. Build the optimized binary:

```bash
go build -ldflags="-s -w" -o server ./cmd/server
```

2. Run with production environment:

```bash
export $(cat .env | xargs) && ./server
```

3. Or use systemd (see [Deployment with systemd](#deployment-with-systemd))

### Verify Installation

Check if the server is running:

```bash
# Health check
curl http://localhost:8080/health

# Should return: {"status":"ok","timestamp":"2025-12-29T..."}
```

## API Documentation

### Base URL

```
http://localhost:8080/api
```

### Authentication

For production environments, include the API key in the `X-API-Key` header:

```bash
curl -H "X-API-Key: your_api_key_here" http://localhost:8080/api/domains
```

### Health Endpoints

#### GET /health
Check if the server is running.

**Response:**
```json
{
  "status": "ok",
  "timestamp": "2025-12-29T10:00:00Z"
}
```

#### GET /ready
Check if the server is ready (database connected).

**Response:**
```json
{
  "status": "ready",
  "database": "connected"
}
```

### Domain Management

#### POST /api/domains
Create a new domain.

**Request:**
```json
{
  "name": "example.com",
  "is_active": true
}
```

**Response:**
```json
{
  "id": 1,
  "name": "example.com",
  "is_active": true,
  "created_at": "2025-12-29T10:00:00Z"
}
```

#### GET /api/domains
List all domains.

**Query Parameters:**
- `limit` (optional): Number of results (default: 20)
- `offset` (optional): Pagination offset (default: 0)

**Response:**
```json
[
  {
    "id": 1,
    "name": "example.com",
    "is_active": true,
    "created_at": "2025-12-29T10:00:00Z"
  }
]
```

#### GET /api/domains/:id
Get a specific domain.

#### PUT /api/domains/:id
Update a domain.

**Request:**
```json
{
  "name": "neweexample.com",
  "is_active": false
}
```

#### DELETE /api/domains/:id
Delete a domain.

### Mailbox Management

#### POST /api/mailboxes
Create a new mailbox.

**Request:**
```json
{
  "local_part": "john.doe",
  "domain_id": 1
}
```

**Response:**
```json
{
  "id": 1,
  "local_part": "john.doe",
  "domain_id": 1,
  "full_address": "john.doe@example.com",
  "created_at": "2025-12-29T10:00:00Z",
  "last_accessed_at": null
}
```

#### POST /api/mailboxes/random
Create a random mailbox.

**Request:**
```json
{
  "domain_id": 1
}
```

**Response:**
```json
{
  "id": 2,
  "local_part": "x7z9qm2k",
  "domain_id": 1,
  "full_address": "x7z9qm2k@example.com",
  "created_at": "2025-12-29T10:00:00Z"
}
```

#### GET /api/mailboxes
List all mailboxes.

**Query Parameters:**
- `domain_id` (optional): Filter by domain
- `limit` (optional): Number of results (default: 20)
- `offset` (optional): Pagination offset (default: 0)

#### GET /api/mailboxes/:id
Get a specific mailbox.

#### DELETE /api/mailboxes/:id
Delete a mailbox and all its messages.

### Message Management

#### GET /api/mailboxes/:mailbox_id/messages
List all messages for a mailbox.

**Query Parameters:**
- `limit` (optional): Number of results (default: 20)
- `offset` (optional): Pagination offset (default: 0)

**Response:**
```json
[
  {
    "id": 1,
    "mailbox_id": 1,
    "sender_email": "sender@example.com",
    "sender_name": "John Sender",
    "subject": "Welcome to Infinimail",
    "snippet": "This is a preview of the email content...",
    "is_read": false,
    "received_at": "2025-12-29T10:00:00Z"
  }
]
```

#### GET /api/messages/:id
Get full message details including body.

**Response:**
```json
{
  "id": 1,
  "mailbox_id": 1,
  "sender_email": "sender@example.com",
  "sender_name": "John Sender",
  "subject": "Welcome to Infinimail",
  "snippet": "This is a preview...",
  "body_text": "Plain text email content",
  "body_html": "<html>HTML email content</html>",
  "is_read": false,
  "received_at": "2025-12-29T10:00:00Z",
  "attachments": []
}
```

#### PATCH /api/messages/:id/read
Mark a message as read.

#### DELETE /api/messages/:id
Delete a message.

### Attachment Management

#### GET /api/messages/:message_id/attachments
List all attachments for a message.

**Response:**
```json
[
  {
    "id": 1,
    "message_id": 1,
    "filename": "document.pdf",
    "content_type": "application/pdf",
    "size_bytes": 102400
  }
]
```

#### GET /api/attachments/:id
Get attachment metadata.

#### GET /api/attachments/:id/download
Download an attachment.

**Response:** Binary file with appropriate Content-Type and Content-Disposition headers.

### WebSocket Connection

#### WS /ws
Connect to receive real-time notifications.

**Example (JavaScript):**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const notification = JSON.parse(event.data);
  console.log('New email:', notification);
};

// Notification format:
// {
//   "type": "new_message",
//   "mailbox_id": 1,
//   "message_id": 42,
//   "subject": "New Email",
//   "sender": "sender@example.com"
// }
```

### Error Responses

All endpoints return consistent error responses:

```json
{
  "error": "Error message description",
  "code": 400
}
```

**HTTP Status Codes:**
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `429` - Too Many Requests
- `500` - Internal Server Error

## Docker Deployment

### Quick Start with Docker Compose

1. **Create docker-compose.yml**:

```yaml
version: '3.8'

services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: infinimail
      POSTGRES_PASSWORD: your_secure_password
      POSTGRES_DB: infinimail
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U infinimail"]
      interval: 5s
      timeout: 5s
      retries: 5

  backend:
    build: .
    ports:
      - "8080:8080"
      - "25:2525"
    environment:
      DATABASE_URL: postgres://infinimail:your_secure_password@db:5432/infinimail?sslmode=disable
      API_PORT: 8080
      SMTP_PORT: 2525
      AUTO_PROVISIONING_ENABLED: "true"
      LOG_LEVEL: info
    depends_on:
      db:
        condition: service_healthy
    volumes:
      - attachment_data:/app/attachments

volumes:
  postgres_data:
  attachment_data:
```

2. **Start the services**:

```bash
docker-compose up -d
```

3. **Check logs**:

```bash
docker-compose logs -f backend
```

4. **Stop services**:

```bash
docker-compose down
```

### Building Docker Image Manually

```bash
# Build the image
docker build -t infinimail-backend:latest .

# Run the container
docker run -d \
  -p 8080:8080 \
  -p 2525:2525 \
  -e DATABASE_URL=postgres://user:pass@host:5432/infinimail \
  -v $(pwd)/attachments:/app/attachments \
  --name infinimail-backend \
  infinimail-backend:latest
```

### Docker Image Details

The Dockerfile uses a multi-stage build:
- **Builder stage**: Compiles the Go application
- **Runtime stage**: Minimal Alpine Linux image (~20MB)
- **Exposed ports**: 8080 (API), 2525 (SMTP)
- **Volume**: `/app/attachments` for email attachments

## Security Considerations

### Authentication

#### API Key Authentication

In production, all API endpoints (except `/health` and `/ready`) require API key authentication:

```bash
# Generate a secure API key
openssl rand -hex 32

# Set in .env
API_KEY=your_generated_key_here
```

Include the key in requests:
```bash
curl -H "X-API-Key: your_api_key_here" http://localhost:8080/api/domains
```

### CORS Configuration

Configure allowed origins to prevent unauthorized cross-origin requests:

```bash
# Development
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001

# Production (NEVER use wildcard *)
ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com
```

### Rate Limiting

Protect against abuse with per-IP rate limiting:

```bash
RATE_LIMIT_REQUESTS=100  # Requests per second
RATE_LIMIT_BURST=20      # Burst capacity
```

**Default limits**:
- Development: 10 requests/second, burst 20
- Production: 100 requests/second, burst 20

### Database Security

#### Development
```bash
DATABASE_URL=postgres://user:pass@localhost:5432/infinimail?sslmode=disable
```

#### Production (REQUIRED)
```bash
DATABASE_URL=postgres://user:pass@localhost:5432/infinimail?sslmode=verify-full
```

**Best Practices**:
- Use strong passwords (minimum 16 characters)
- Enable SSL/TLS for database connections
- Restrict database access to application server only
- Regular security updates for PostgreSQL

### SMTP Security

#### Development
```bash
SMTP_PORT=2525
SMTP_ALLOW_INSECURE=true
```

#### Production
```bash
SMTP_PORT=25
SMTP_ALLOW_INSECURE=false
# Configure TLS certificates
SMTP_TLS_CERT=/path/to/cert.pem
SMTP_TLS_KEY=/path/to/key.pem
```

### File Storage Security

- Attachments are stored outside the web root
- Filenames are sanitized to prevent directory traversal
- Maximum file size enforced (25 MB default)
- Content-Type validation

### Security Headers

The application automatically sets security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security` (if HTTPS)
- `Content-Security-Policy`

### Input Validation

- All user inputs are validated
- SQL injection protection via GORM parameterized queries
- XSS protection via output encoding
- Email address validation

### Logging and Monitoring

- Structured JSON logging
- Security event logging
- No sensitive data in logs (passwords, API keys)
- Failed authentication attempts logged

## Development

### Project Structure

```
webrana-infinimail-backend/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/            # HTTP request handlers
│   │   ├── middleware/          # HTTP middleware
│   │   ├── response/            # Response helpers
│   │   └── router.go            # Route definitions
│   ├── config/                  # Configuration management
│   ├── database/                # Database connection & migrations
│   ├── errors/                  # Custom error types
│   ├── logger/                  # Logging utilities
│   ├── models/                  # Data models
│   ├── repository/              # Data access layer
│   ├── smtp/                    # SMTP server implementation
│   ├── storage/                 # File storage
│   ├── validator/               # Input validation
│   └── websocket/               # WebSocket implementation
├── tests/
│   ├── integration/             # Integration tests
│   ├── e2e/                     # End-to-end tests
│   ├── fixtures/                # Test fixtures
│   └── mocks/                   # Mock implementations
├── .env.example                 # Development config template
├── .env.secure.example          # Production config template
├── Dockerfile                   # Docker image definition
├── docker-compose.test.yml      # Testing environment
├── Makefile                     # Build automation
└── README.md                    # This file
```

### Code Style

This project follows standard Go conventions:

```bash
# Format code
go fmt ./...
# or
make fmt

# Run linter (requires golangci-lint)
golangci-lint run
# or
make lint

# Vet code
go vet ./...
```

### Adding New Features

1. **Create models** in `internal/models/`
2. **Add repository interface** in `internal/repository/`
3. **Implement handlers** in `internal/api/handlers/`
4. **Register routes** in `internal/api/router.go`
5. **Write tests** in corresponding `_test.go` files
6. **Update documentation**

### Database Migrations

Migrations are automatically run on startup. To add new migrations:

1. Add migration code in `internal/database/database.go`
2. Use GORM AutoMigrate for simple schema changes:

```go
db.AutoMigrate(&models.YourNewModel{})
```

For complex migrations, consider using a migration tool like [golang-migrate](https://github.com/golang-migrate/migrate).

## Testing

The project includes comprehensive tests: unit tests, integration tests, and end-to-end tests.

### Quick Start

```bash
# Run all tests
make test

# Run only unit tests (fast, no dependencies)
make test-unit

# Run integration tests (requires Docker)
make test-integration

# Run end-to-end tests
make test-e2e

# Generate coverage report
make test-coverage
# Opens coverage.html in your browser

# Run with race detection
make test-race
```

### Unit Tests

Unit tests are located alongside the code they test:

```bash
# Run unit tests for specific package
go test ./internal/api/handlers -v

# Run with coverage
go test ./internal/api/handlers -cover -v
```

### Integration Tests

Integration tests use testcontainers to spin up real PostgreSQL instances:

```bash
# Requires Docker
go test ./tests/integration/... -v -tags=integration
```

### End-to-End Tests

E2E tests validate the entire email flow:

```bash
go test ./tests/e2e/... -v -tags=e2e
```

### Test Coverage

Current coverage: **~85%** (target: 80%+)

```bash
# Generate and view coverage report
make test-coverage
```

### Writing Tests

Example test structure:

```go
func TestHandler_Create(t *testing.T) {
    // Setup
    repo := mocks.NewMockRepository()
    handler := NewHandler(repo)

    // Test cases
    tests := []struct {
        name           string
        input          interface{}
        expectedStatus int
        expectedError  string
    }{
        {
            name:           "valid input",
            input:          validInput,
            expectedStatus: 201,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Execute
            result := handler.Create(tt.input)

            // Assert
            assert.Equal(t, tt.expectedStatus, result.Status)
        })
    }
}
```

## Troubleshooting

### Common Issues

#### 1. Database Connection Failed

**Error**: `failed to connect to database: connection refused`

**Solutions**:
- Check if PostgreSQL is running: `docker ps` or `systemctl status postgresql`
- Verify DATABASE_URL is correct
- Check firewall settings
- Ensure PostgreSQL accepts connections from your host

```bash
# Test PostgreSQL connection
psql "postgres://infinimail:infinimail@localhost:5432/infinimail"
```

#### 2. Port Already in Use

**Error**: `bind: address already in use`

**Solutions**:
```bash
# Find process using port 8080
lsof -i :8080
# or
netstat -tlnp | grep 8080

# Kill the process
kill -9 <PID>

# Or change the port in .env
API_PORT=8081
```

#### 3. SMTP Server Not Receiving Emails

**Symptoms**: Emails sent to the server don't appear

**Solutions**:
- Check if SMTP server is running: `telnet localhost 2525`
- Verify domain is registered in the database:
  ```bash
  curl http://localhost:8080/api/domains
  ```
- Check server logs for errors
- Ensure AUTO_PROVISIONING_ENABLED=true if mailbox doesn't exist
- Verify firewall allows SMTP port (25/2525)

```bash
# Test SMTP manually
telnet localhost 2525
EHLO localhost
MAIL FROM:<test@example.com>
RCPT TO:<user@yourdomain.com>
DATA
Subject: Test
This is a test email.
.
QUIT
```

#### 4. Attachment Upload Fails

**Error**: `failed to store attachment`

**Solutions**:
- Check ATTACHMENT_STORAGE_PATH directory exists and is writable:
  ```bash
  mkdir -p ./attachments
  chmod 755 ./attachments
  ```
- Verify disk space: `df -h`
- Check file size against SMTP_MAX_MESSAGE_SIZE
- Review file permissions

#### 5. WebSocket Connection Refused

**Error**: `WebSocket connection failed`

**Solutions**:
- Verify ALLOWED_ORIGINS includes your frontend URL
- Check if WebSocket endpoint is accessible: `curl -i http://localhost:8080/ws`
- Ensure no proxy is blocking WebSocket upgrade
- Check browser console for CORS errors

#### 6. API Returns 401 Unauthorized

**Error**: `{"error": "unauthorized", "code": 401}`

**Solutions**:
- Include X-API-Key header in requests
- Verify API_KEY is set in .env
- Check if API_KEY matches between .env and request
- In development, leave API_KEY empty to disable authentication

```bash
# Test with API key
curl -H "X-API-Key: your_key" http://localhost:8080/api/domains
```

#### 7. Rate Limit Exceeded

**Error**: `{"error": "rate limit exceeded", "code": 429}`

**Solutions**:
- Wait for rate limit window to reset (typically 1 second)
- Increase RATE_LIMIT_REQUESTS in .env
- Implement exponential backoff in client
- Check for infinite loops in client code

### Debug Mode

Enable debug logging for troubleshooting:

```bash
LOG_LEVEL=debug go run ./cmd/server
```

This will log:
- All HTTP requests and responses
- Database queries
- SMTP transactions
- WebSocket events

### Getting Help

If you encounter issues not covered here:

1. **Check logs**: `docker-compose logs -f backend`
2. **Search issues**: [GitHub Issues](https://github.com/welldanyogia/webrana-infinimail-backend/issues)
3. **Create an issue**: Include:
   - Go version: `go version`
   - PostgreSQL version
   - Error messages and logs
   - Steps to reproduce
   - Configuration (remove sensitive data)

## Contributing

We welcome contributions! Here's how to get started:

### Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/webrana-infinimail-backend.git
   cd webrana-infinimail-backend
   ```
3. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```
4. Make your changes
5. Run tests:
   ```bash
   make test
   ```
6. Run linter:
   ```bash
   make lint
   ```
7. Commit your changes:
   ```bash
   git commit -m "feat: add amazing feature"
   ```
8. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```
9. Create a Pull Request

### Commit Message Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding or updating tests
- `refactor:` Code refactoring
- `perf:` Performance improvements
- `chore:` Maintenance tasks

Examples:
```
feat: add attachment compression
fix: resolve race condition in WebSocket hub
docs: update API documentation
test: add integration tests for SMTP
```

### Code Review Process

1. All submissions require review
2. Tests must pass
3. Code coverage should not decrease
4. Follow Go best practices
5. Update documentation if needed

### Areas for Contribution

- [ ] Attachment compression (gzip)
- [ ] Email search functionality
- [ ] Spam filtering
- [ ] S3 storage backend
- [ ] Email forwarding
- [ ] Webhook notifications
- [ ] Metrics and monitoring (Prometheus)
- [ ] Admin dashboard
- [ ] Email templates
- [ ] Internationalization (i18n)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Appendix

### Deployment with systemd

Create `/etc/systemd/system/infinimail.service`:

```ini
[Unit]
Description=Infinimail Backend Service
After=network.target postgresql.service

[Service]
Type=simple
User=infinimail
WorkingDirectory=/opt/infinimail
EnvironmentFile=/opt/infinimail/.env
ExecStart=/opt/infinimail/server
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable infinimail
sudo systemctl start infinimail
sudo systemctl status infinimail
```

### Nginx Reverse Proxy

Example Nginx configuration:

```nginx
server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_cache_bypass $http_upgrade;
    }

    # WebSocket support
    location /ws {
        proxy_pass http://localhost:8080/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
    }
}
```

### Performance Tuning

#### PostgreSQL

Edit `postgresql.conf`:
```ini
max_connections = 100
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
work_mem = 10MB
```

#### Application

```bash
# Increase rate limits
RATE_LIMIT_REQUESTS=1000
RATE_LIMIT_BURST=100

# Optimize database connections
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
```

### Monitoring

#### Health Checks

```bash
# Basic health check
curl http://localhost:8080/health

# Database connectivity
curl http://localhost:8080/ready
```

#### Logging

Logs are output in JSON format to stdout:
```json
{
  "time":"2025-12-29T10:00:00Z",
  "level":"INFO",
  "msg":"starting HTTP server",
  "addr":":8080"
}
```

Parse with jq:
```bash
docker-compose logs -f backend | jq -r '.msg'
```

### Backup and Restore

#### Database Backup

```bash
# Backup
docker exec infinimail-postgres pg_dump -U infinimail infinimail > backup.sql

# Restore
docker exec -i infinimail-postgres psql -U infinimail infinimail < backup.sql
```

#### Attachment Backup

```bash
# Backup attachments
tar -czf attachments-backup.tar.gz ./attachments/

# Restore
tar -xzf attachments-backup.tar.gz
```

---

**Built with Go by the Webrana Team**

For questions and support, please open an issue on [GitHub](https://github.com/welldanyogia/webrana-infinimail-backend).
