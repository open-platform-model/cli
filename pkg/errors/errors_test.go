package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

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

func TestWrap(t *testing.T) {
	wrapped := oerrors.Wrap(oerrors.ErrValidation, "schema check failed")

	assert.True(t, errors.Is(wrapped, oerrors.ErrValidation))
	assert.Contains(t, wrapped.Error(), "schema check failed")
}
