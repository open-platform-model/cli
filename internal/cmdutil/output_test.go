package cmdutil

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/output"
	pkgerrors "github.com/opmodel/cli/pkg/errors"
)

func TestPrintValidationError_ConfigError(t *testing.T) {
	// Setup: capture log output.
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	// Create a ConfigError (with a nil RawError — simulates a gate error without CUE tree).
	err := &pkgerrors.ConfigError{
		Context: "module gate",
		Name:    "test-module",
	}

	PrintValidationError("render failed", err)

	got := buf.String()
	assert.Contains(t, got, "render failed", "should contain message")
}

func TestPrintValidationError_ValidationError(t *testing.T) {
	// Setup: capture log output.
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	err := &pkgerrors.ValidationError{
		Message: "value not concrete",
		Details: "path.to.field:\n    conflicting values",
	}

	PrintValidationError("render failed", err)

	got := buf.String()
	assert.Contains(t, got, "render failed", "should contain message")
	assert.Contains(t, got, "value not concrete", "should contain error message")
}

func TestPrintValidationError_GenericError(t *testing.T) {
	// Setup: capture log output.
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	err := fmt.Errorf("something went wrong")

	PrintValidationError("render failed", err)

	got := buf.String()
	assert.Contains(t, got, "render failed", "should contain message")
	assert.Contains(t, got, "something went wrong", "should contain error message")
}

func TestPrintValidationError_GroupedConfigError(t *testing.T) {
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	configErr := &pkgerrors.ConfigError{
		Context:  "module gate",
		Name:     "demo",
		RawError: fmt.Errorf("field not allowed\nconflicting values"),
	}

	PrintValidationError("render failed", configErr)

	got := buf.String()
	assert.Contains(t, got, "render failed")
}
