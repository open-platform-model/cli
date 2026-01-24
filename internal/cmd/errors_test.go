package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExitCodeFromError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error returns success",
			err:      nil,
			expected: ExitSuccess,
		},
		{
			name:     "validation error",
			err:      ErrValidation,
			expected: ExitValidationError,
		},
		{
			name:     "connectivity error",
			err:      ErrConnectivity,
			expected: ExitConnectivityError,
		},
		{
			name:     "permission error",
			err:      ErrPermission,
			expected: ExitPermissionDenied,
		},
		{
			name:     "not found error",
			err:      ErrNotFound,
			expected: ExitNotFound,
		},
		{
			name:     "version error",
			err:      ErrVersion,
			expected: ExitVersionMismatch,
		},
		{
			name:     "wrapped validation error",
			err:      fmt.Errorf("failed to validate: %w", ErrValidation),
			expected: ExitValidationError,
		},
		{
			name:     "wrapped connectivity error",
			err:      fmt.Errorf("connection failed: %w", ErrConnectivity),
			expected: ExitConnectivityError,
		},
		{
			name:     "unknown error returns general error",
			err:      errors.New("something went wrong"),
			expected: ExitGeneralError,
		},
		{
			name:     "exit error with custom code",
			err:      NewExitError(errors.New("custom error"), 42),
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := ExitCodeFromError(tt.err)
			assert.Equal(t, tt.expected, code)
		})
	}
}

func TestExitError(t *testing.T) {
	originalErr := errors.New("original error")
	exitErr := NewExitError(originalErr, ExitValidationError)

	t.Run("Error returns wrapped error message", func(t *testing.T) {
		assert.Equal(t, "original error", exitErr.Error())
	})

	t.Run("Unwrap returns original error", func(t *testing.T) {
		assert.Equal(t, originalErr, errors.Unwrap(exitErr))
	})

	t.Run("errors.Is works with unwrapped error", func(t *testing.T) {
		assert.True(t, errors.Is(exitErr, originalErr))
	})
}

func TestWrapFunctions(t *testing.T) {
	originalErr := errors.New("original")

	t.Run("WrapValidation", func(t *testing.T) {
		err := WrapValidation(originalErr, "context")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrValidation))
		assert.True(t, errors.Is(err, originalErr))
		assert.Contains(t, err.Error(), "context")
	})

	t.Run("WrapConnectivity", func(t *testing.T) {
		err := WrapConnectivity(originalErr, "context")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrConnectivity))
		assert.True(t, errors.Is(err, originalErr))
	})

	t.Run("WrapPermission", func(t *testing.T) {
		err := WrapPermission(originalErr, "context")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrPermission))
		assert.True(t, errors.Is(err, originalErr))
	})

	t.Run("WrapNotFound", func(t *testing.T) {
		err := WrapNotFound(originalErr, "context")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound))
		assert.True(t, errors.Is(err, originalErr))
	})

	t.Run("WrapVersion", func(t *testing.T) {
		err := WrapVersion(originalErr, "context")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrVersion))
		assert.True(t, errors.Is(err, originalErr))
	})
}

func TestExitCodeName(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{ExitSuccess, "Success"},
		{ExitGeneralError, "General Error"},
		{ExitValidationError, "Validation Error"},
		{ExitConnectivityError, "Connectivity Error"},
		{ExitPermissionDenied, "Permission Denied"},
		{ExitNotFound, "Not Found"},
		{ExitVersionMismatch, "Version Mismatch"},
		{999, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, ExitCodeName(tt.code))
		})
	}
}
