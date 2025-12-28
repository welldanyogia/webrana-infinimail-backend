package database

import (
	"fmt"
	"log/slog"

	"github.com/welldanyogia/webrana-infinimail-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect establishes a connection to the PostgreSQL database
func Connect(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	slog.Info("Connected to database successfully")
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
