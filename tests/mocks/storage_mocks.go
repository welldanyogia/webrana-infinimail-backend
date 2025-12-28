package mocks

import (
	"io"

	"github.com/stretchr/testify/mock"
)

// MockFileStorage implements storage.FileStorage
type MockFileStorage struct {
	mock.Mock
}

// Save stores a file and returns the relative path
func (m *MockFileStorage) Save(filename string, content io.Reader) (string, error) {
	args := m.Called(filename, content)
	return args.String(0), args.Error(1)
}

// Get retrieves a file by its path
func (m *MockFileStorage) Get(filePath string) (io.ReadCloser, error) {
	args := m.Called(filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// Delete removes a file by its path
func (m *MockFileStorage) Delete(filePath string) error {
	args := m.Called(filePath)
	return args.Error(0)
}
