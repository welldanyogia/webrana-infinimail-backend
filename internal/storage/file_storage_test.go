package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePath_PathTraversalDots(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewLocalStorage(tempDir)
	require.NoError(t, err)

	ls := storage.(*localStorage)

	tests := []struct {
		name string
		path string
	}{
		{"simple traversal", "../etc/passwd"},
		{"double traversal", "../../etc/passwd"},
		{"nested traversal", "subdir/../../../etc/passwd"},
		{"windows style", "..\\..\\windows\\system32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ls.validatePath(tt.path)
			assert.ErrorIs(t, err, ErrPathTraversal)
		})
	}
}

func TestValidatePath_AbsolutePath(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewLocalStorage(tempDir)
	require.NoError(t, err)

	ls := storage.(*localStorage)

	// Windows absolute path
	_, err = ls.validatePath("C:\\Windows\\System32")
	assert.ErrorIs(t, err, ErrPathTraversal)

	// Also test that paths outside base are rejected via containment check
	// even if they don't look "absolute" on this OS
}

func TestValidatePath_ValidPath(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewLocalStorage(tempDir)
	require.NoError(t, err)

	ls := storage.(*localStorage)

	tests := []struct {
		name string
		path string
	}{
		{"simple file", "file.txt"},
		{"subdirectory", "subdir/file.txt"},
		{"nested subdirectory", "a/b/c/file.txt"},
		{"uuid style", "ab/ab123456-7890.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ls.validatePath(tt.path)
			assert.NoError(t, err)
			assert.True(t, strings.HasPrefix(result, tempDir))
		})
	}
}

func TestValidatePath_Containment(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewLocalStorage(tempDir)
	require.NoError(t, err)

	ls := storage.(*localStorage)

	// Valid path should be within base directory
	result, err := ls.validatePath("test/file.txt")
	assert.NoError(t, err)

	absBase, _ := filepath.Abs(tempDir)
	assert.True(t, strings.HasPrefix(result, absBase))
}

func TestGet_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewLocalStorage(tempDir)
	require.NoError(t, err)

	// Try to access file outside storage directory
	_, err = storage.Get("../../../etc/passwd")
	assert.ErrorIs(t, err, ErrPathTraversal)
}

func TestDelete_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewLocalStorage(tempDir)
	require.NoError(t, err)

	// Try to delete file outside storage directory
	err = storage.Delete("../../../etc/passwd")
	assert.ErrorIs(t, err, ErrPathTraversal)
}

func TestGet_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewLocalStorage(tempDir)
	require.NoError(t, err)

	_, err = storage.Get("nonexistent.txt")
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestValidateFile_BlockedExtensions(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"exe blocked", "malware.exe", true},
		{"bat blocked", "script.bat", true},
		{"cmd blocked", "script.cmd", true},
		{"sh blocked", "script.sh", true},
		{"ps1 blocked", "script.ps1", true},
		{"jar blocked", "app.jar", true},
		{"pdf allowed", "document.pdf", false},
		{"txt allowed", "readme.txt", false},
		{"jpg allowed", "image.jpg", false},
		{"uppercase exe blocked", "MALWARE.EXE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFile(tt.filename, 1024)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrBlockedExt)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFile_SizeLimit(t *testing.T) {
	// File within limit
	err := ValidateFile("file.pdf", MaxFileSize-1)
	assert.NoError(t, err)

	// File at limit
	err = ValidateFile("file.pdf", MaxFileSize)
	assert.NoError(t, err)

	// File exceeds limit
	err = ValidateFile("file.pdf", MaxFileSize+1)
	assert.ErrorIs(t, err, ErrFileTooLarge)
}

func TestSaveAndGet_Integration(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewLocalStorage(tempDir)
	require.NoError(t, err)

	// Save a file
	content := strings.NewReader("test content")
	path, err := storage.Save("test.txt", content)
	require.NoError(t, err)
	assert.NotEmpty(t, path)

	// Get the file
	reader, err := storage.Get(path)
	require.NoError(t, err)
	defer reader.Close()

	// Verify content
	buf := make([]byte, 100)
	n, _ := reader.Read(buf)
	assert.Equal(t, "test content", string(buf[:n]))
}

func TestDelete_Integration(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewLocalStorage(tempDir)
	require.NoError(t, err)

	// Save a file
	content := strings.NewReader("test content")
	path, err := storage.Save("test.txt", content)
	require.NoError(t, err)

	// Delete the file
	err = storage.Delete(path)
	assert.NoError(t, err)

	// Verify file is gone
	_, err = storage.Get(path)
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestDelete_NonexistentFile(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewLocalStorage(tempDir)
	require.NoError(t, err)

	// Deleting nonexistent file should not error
	err = storage.Delete("nonexistent.txt")
	assert.NoError(t, err)
}

func TestNewLocalStorage_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	newDir := filepath.Join(tempDir, "new", "nested", "dir")

	_, err := NewLocalStorage(newDir)
	assert.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(newDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}
