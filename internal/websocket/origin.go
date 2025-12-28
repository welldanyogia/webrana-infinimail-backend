package websocket

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

// NewSecureUpgrader creates a WebSocket upgrader with origin validation
func NewSecureUpgrader(logger *slog.Logger) websocket.Upgrader {
	allowedOrigins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}

	// Filter empty strings
	filtered := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin != "" {
			filtered = append(filtered, origin)
		}
	}
	allowedOrigins = filtered

	// Default to localhost if no origins configured
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:3000"}
	}

	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")

			// Allow same-origin requests (empty Origin)
			if origin == "" {
				return true
			}

			// Check against allowed origins
			for _, allowed := range allowedOrigins {
				if allowed == origin {
					return true
				}
			}

			if logger != nil {
				logger.Warn("rejected websocket connection",
					slog.String("origin", origin),
					slog.String("remote_ip", r.RemoteAddr))
			}
			return false
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

// DefaultUpgrader returns an upgrader that allows all origins (for development)
func DefaultUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}
