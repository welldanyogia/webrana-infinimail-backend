package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Security errors
var (
	ErrPathTraversal = errors.New("path traversal detected")
	ErrFileNotFound  = errors.New("file not found")
	ErrFileTooLarge  = errors.New("file exceeds size limit")
	ErrBlockedExt    = errors.New("file extension is blocked")
)

// MaxFileSize is the maximum allowed file size (25 MB)
const MaxFileSize = 25 * 1024 * 1024

// BlockedExtensions contains file extensions that are not allowed
var BlockedExtensions = map[string]bool{
	".exe": true, ".bat": true, ".cmd": true, ".com": true,
	".pif": true, ".scr": true, ".vbs": true, ".js": true,
	".jar": true, ".ps1": true, ".sh": true, ".bash": true,
	".msi": true, ".dll": true, ".sys": true,
}

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
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) &&
		absPath != absBase {
		return "", ErrPathTraversal
	}

	return absPath, nil
}

// ValidateFile checks file extension and size
func ValidateFile(filename string, size int64) error {
	ext := strings.ToLower(filepath.Ext(filename))

	if BlockedExtensions[ext] {
		return ErrBlockedExt
	}

	if size > MaxFileSize {
		return ErrFileTooLarge
	}

	return nil
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
	// Validate path to prevent traversal
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

// Delete removes a file by its path
func (s *localStorage) Delete(filePath string) error {
	// Validate path to prevent traversal
	fullPath, err := s.validatePath(filePath)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			// File already doesn't exist, not an error
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}
