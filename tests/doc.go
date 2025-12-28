// Package tests contains test utilities and test files for the Infinimail backend.
// This file ensures testing dependencies are kept in go.mod.
package tests

import (
	// Testing dependencies - imported to ensure they stay in go.mod
	_ "github.com/DATA-DOG/go-sqlmock"
	_ "github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/mock"
	_ "github.com/stretchr/testify/require"
	_ "github.com/testcontainers/testcontainers-go"
)
