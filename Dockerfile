# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Create attachments directory
RUN mkdir -p /app/attachments

# Expose ports
EXPOSE 8080 2525

# Set default environment variables
ENV API_PORT=8080
ENV SMTP_PORT=2525
ENV ATTACHMENT_STORAGE_PATH=/app/attachments
ENV AUTO_PROVISIONING_ENABLED=true
ENV LOG_LEVEL=info

# Run the server
CMD ["./server"]
