package database

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connection pool configuration
const (
	DefaultMaxIdleConns    = 10
	DefaultMaxOpenConns    = 100
	DefaultConnMaxLifetime = time.Hour
	DefaultConnMaxIdleTime = 10 * time.Minute
)

// Connect establishes a connection to the PostgreSQL database
func Connect(databaseURL string) (*gorm.DB, error) {
	// Validate SSL mode in production
	env := os.Getenv("APP_ENV")
	if env == "production" {
		if err := validateSSLMode(databaseURL); err != nil {
			return nil, err
		}
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	if err := configureConnectionPool(db); err != nil {
		return nil, err
	}

	slog.Info("Connected to database successfully")
	return db, nil
}

// validateSSLMode ensures SSL is enabled in production
func validateSSLMode(databaseURL string) error {
	// Check if sslmode is explicitly disabled
	if strings.Contains(databaseURL, "sslmode=disable") {
		return fmt.Errorf("SSL mode cannot be disabled in production")
	}

	// If no sslmode specified, it's okay (defaults to prefer/require depending on server)
	return nil
}

// configureConnectionPool sets up connection pool limits
func configureConnectionPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool limits
	sqlDB.SetMaxIdleConns(DefaultMaxIdleConns)
	sqlDB.SetMaxOpenConns(DefaultMaxOpenConns)
	sqlDB.SetConnMaxLifetime(DefaultConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(DefaultConnMaxIdleTime)

	return nil
}

// ConnectWithConfig establishes a connection with custom pool configuration
func ConnectWithConfig(databaseURL string, maxIdleConns, maxOpenConns int, connMaxLifetime, connMaxIdleTime time.Duration) (*gorm.DB, error) {
	// Validate SSL mode in production
	env := os.Getenv("APP_ENV")
	if env == "production" {
		if err := validateSSLMode(databaseURL); err != nil {
			return nil, err
		}
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	slog.Info("Connected to database successfully with custom config")
	return db, nil
}

// Migrate runs auto-migration for all models
func Migrate(db *gorm.DB) error {
	slog.Info("Running database migrations...")

	err := db.AutoMigrate(
		&models.Domain{},
		&models.Mailbox{},
		&models.Message{},
		&models.Attachment{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}
