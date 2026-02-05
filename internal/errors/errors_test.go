package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentinelErrors(t *testing.T) {
	// Verify sentinel errors are distinct
	assert.NotEqual(t, ErrValidation, ErrConnectivity)
	assert.NotEqual(t, ErrValidation, ErrPermission)
	assert.NotEqual(t, ErrValidation, ErrNotFound)
}

func TestErrorDetailError(t *testing.T) {
	detail := &ErrorDetail{
		Type:     "validation failed",
		Message:  "invalid value",
		Location: "/path/to/file.cue:42",
		Field:    "metadata.version",
		Context:  map[string]string{"Provider": "kubernetes"},
		Hint:     "Use semver format",
	}

	output := detail.Error()

	assert.Contains(t, output, "Error: validation failed")
	assert.Contains(t, output, "Location: /path/to/file.cue:42")
	assert.Contains(t, output, "Field: metadata.version")
	assert.Contains(t, output, "Provider: kubernetes")
	assert.Contains(t, output, "invalid value")
	assert.Contains(t, output, "Hint: Use semver format")
}

func TestErrorDetailUnwrap(t *testing.T) {
	detail := &ErrorDetail{
		Type:    "test",
		Message: "test message",
		Cause:   ErrValidation,
	}

	assert.True(t, errors.Is(detail, ErrValidation))
	assert.Equal(t, ErrValidation, detail.Unwrap())
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError(
		"invalid value",
		"/path/to/file.cue:42",
		"metadata.version",
		"Use semver format",
	)

	require.NotNil(t, err)
	assert.True(t, errors.Is(err, ErrValidation))

	detail, ok := err.(*ErrorDetail)
	require.True(t, ok)
	assert.Equal(t, "validation failed", detail.Type)
	assert.Equal(t, "invalid value", detail.Message)
	assert.Equal(t, "/path/to/file.cue:42", detail.Location)
	assert.Equal(t, "metadata.version", detail.Field)
	assert.Equal(t, "Use semver format", detail.Hint)
}

func TestNewConnectivityError(t *testing.T) {
	err := NewConnectivityError(
		"connection refused",
		map[string]string{"Registry": "localhost:5000"},
		"Check registry is running",
	)

	require.NotNil(t, err)
	assert.True(t, errors.Is(err, ErrConnectivity))

	detail, ok := err.(*ErrorDetail)
	require.True(t, ok)
	assert.Equal(t, "connectivity failed", detail.Type)
	assert.Equal(t, "localhost:5000", detail.Context["Registry"])
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError(
		"file does not exist",
		"~/.opm/config.cue",
		"Run 'opm config init'",
	)

	require.NotNil(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))

	detail, ok := err.(*ErrorDetail)
	require.True(t, ok)
	assert.Equal(t, "not found", detail.Type)
}

func TestNewPermissionError(t *testing.T) {
	err := NewPermissionError(
		"cannot create deployments",
		map[string]string{"Namespace": "production"},
		"Contact admin",
	)

	require.NotNil(t, err)
	assert.True(t, errors.Is(err, ErrPermission))
}

func TestWrap(t *testing.T) {
	wrapped := Wrap(ErrValidation, "schema check failed")

	assert.True(t, errors.Is(wrapped, ErrValidation))
	assert.Contains(t, wrapped.Error(), "schema check failed")
}
