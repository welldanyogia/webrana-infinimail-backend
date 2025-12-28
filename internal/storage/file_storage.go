package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// FileStorage defines the interface for file storage operations
type FileStorage interface {
	Save(filename string, content io.Reader) (string, error)
	Get(filePath string) (io.ReadCloser, error)
	Delete(filePath string) error
}

// localStorage implements FileStorage using local filesystem
type localStorage struct {
	basePath string
}

// NewLocalStorage creates a new localStorage instance
func NewLocalStorage(basePath string) (FileStorage, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &localStorage{basePath: basePath}, nil
}

// Save stores a file and returns the relative path
func (s *localStorage) Save(filename string, content io.Reader) (string, error) {
	// Generate unique filename to prevent conflicts
	ext := filepath.Ext(filename)
	uniqueName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	
	// Create subdirectory based on first 2 chars of UUID for better distribution
	subDir := uniqueName[:2]
	dirPath := filepath.Join(s.basePath, subDir)
	
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create subdirectory: %w", err)
	}
	
	// Full path for the file
	filePath := filepath.Join(subDir, uniqueName)
	fullPath := filepath.Join(s.basePath, filePath)
	
	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	// Copy content to file
	if _, err := io.Copy(file, content); err != nil {
		// Clean up on error
		os.Remove(fullPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	
	return filePath, nil
}

// Get retrieves a file by its path
func (s *localStorage) Get(filePath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, filePath)
	
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	return file, nil
}

// Delete removes a file by its path
func (s *localStorage) Delete(filePath string) error {
	fullPath := filepath.Join(s.basePath, filePath)
	
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			// File already doesn't exist, not an error
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	
	return nil
}
