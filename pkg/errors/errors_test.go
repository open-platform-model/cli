package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

func TestSentinelErrors(t *testing.T) {
	assert.NotEqual(t, oerrors.ErrValidation, oerrors.ErrConnectivity)
	assert.NotEqual(t, oerrors.ErrValidation, oerrors.ErrPermission)
	assert.NotEqual(t, oerrors.ErrValidation, oerrors.ErrNotFound)
}

func TestDetailErrorError(t *testing.T) {
	detail := &oerrors.DetailError{
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

func TestDetailErrorUnwrap(t *testing.T) {
	detail := &oerrors.DetailError{
		Type:    "test",
		Message: "test message",
		Cause:   oerrors.ErrValidation,
	}

	assert.True(t, errors.Is(detail, oerrors.ErrValidation))
	assert.Equal(t, oerrors.ErrValidation, detail.Unwrap())
}

func TestNewValidationError(t *testing.T) {
	err := oerrors.NewValidationError(
		"invalid value",
		"/path/to/file.cue:42",
		"metadata.version",
		"Use semver format",
	)

	require.NotNil(t, err)
	assert.True(t, errors.Is(err, oerrors.ErrValidation))

	var detail *oerrors.DetailError
	require.True(t, errors.As(err, &detail))
	assert.Equal(t, "validation failed", detail.Type)
	assert.Equal(t, "invalid value", detail.Message)
	assert.Equal(t, "/path/to/file.cue:42", detail.Location)
	assert.Equal(t, "metadata.version", detail.Field)
	assert.Equal(t, "Use semver format", detail.Hint)
}

func TestWrap(t *testing.T) {
	wrapped := oerrors.Wrap(oerrors.ErrValidation, "schema check failed")

	assert.True(t, errors.Is(wrapped, oerrors.ErrValidation))
	assert.Contains(t, wrapped.Error(), "schema check failed")
}
