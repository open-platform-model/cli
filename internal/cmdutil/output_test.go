package cmdutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/core"
	build "github.com/opmodel/cli/internal/legacy"
	"github.com/opmodel/cli/internal/legacy/transform"
	"github.com/opmodel/cli/internal/output"
)

func TestPrintValidationError_ReleaseValidationWithDetails(t *testing.T) {
	// Setup: capture both log output and stderr
	var logBuf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&logBuf)

	// Capture stderr (Details writes directly to stderr)
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	// Create a ValidationError with CUE details
	relErr := &core.ValidationError{
		Message: "value not concrete",
		Details: "path.to.field:\n    conflicting values \"foo\" and \"bar\"",
	}

	PrintValidationError("render failed", relErr)

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	var stderrBuf bytes.Buffer
	io.Copy(&stderrBuf, r)

	logOutput := logBuf.String()
	stderrOutput := stderrBuf.String()

	// Should contain summary line in log output
	assert.Contains(t, logOutput, "render failed", "should contain message")
	assert.Contains(t, logOutput, "value not concrete", "should contain error message")

	// Should contain CUE details in stderr
	assert.Contains(t, stderrOutput, "path.to.field", "should contain CUE path")
	assert.Contains(t, stderrOutput, "conflicting values", "should contain CUE error details")
}

func TestPrintValidationError_ReleaseValidationWithoutDetails(t *testing.T) {
	// Setup: capture stderr output
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	// Create a ValidationError without CUE details
	err := &core.ValidationError{
		Message: "value not concrete",
		Details: "", // empty details
	}

	PrintValidationError("render failed", err)

	got := buf.String()

	// Should fall through to key-value format
	assert.Contains(t, got, "render failed", "should contain message")
	assert.Contains(t, got, "error", "should contain error key in key-value format")
}

func TestPrintValidationError_GenericError(t *testing.T) {
	// Setup: capture stderr output
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	// Create a generic error
	err := fmt.Errorf("something went wrong")

	PrintValidationError("render failed", err)

	got := buf.String()

	// Should use key-value format
	assert.Contains(t, got, "render failed", "should contain message")
	assert.Contains(t, got, "error", "should contain error key")
	assert.Contains(t, got, "something went wrong", "should contain error message")
}

func TestPrintRenderErrors_UnmatchedWithAvailable(t *testing.T) {
	// Setup: capture stderr output
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	// Create an UnmatchedComponentError with Available transformers
	errs := []error{
		&build.UnmatchedComponentError{
			ComponentName: "database",
			Available: []core.TransformerRequirements{
				&transform.LoadedTransformer{
					FQN:               "example.com/transformers@v1#PostgresTransformer",
					RequiredLabels:    map[string]string{"db-type": "postgres"},
					RequiredResources: []string{"opmodel.dev/resources/Database@v0"},
					RequiredTraits:    []string{"opmodel.dev/traits/Persistence@v0"},
				},
			},
		},
	}

	PrintRenderErrors(errs)

	got := buf.String()

	// Should contain component name
	assert.Contains(t, got, "database", "should contain component name")
	assert.Contains(t, got, "no matching transformer", "should contain error message")

	// Should list available transformers
	assert.Contains(t, got, "Available transformers", "should show available transformers header")
	assert.Contains(t, got, "PostgresTransformer", "should show transformer FQN")
	assert.Contains(t, got, "requiredLabels", "should show required labels")
	assert.Contains(t, got, "db-type", "should show label key")
	assert.Contains(t, got, "requiredResources", "should show required resources")
	assert.Contains(t, got, "Database", "should show resource name")
	assert.Contains(t, got, "requiredTraits", "should show required traits")
	assert.Contains(t, got, "Persistence", "should show trait name")
}

func TestPrintRenderErrors_UnmatchedWithoutAvailable(t *testing.T) {
	// Setup: capture stderr output
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	// Create an UnmatchedComponentError without Available transformers
	errs := []error{
		&build.UnmatchedComponentError{
			ComponentName: "cache",
			Available:     nil,
		},
	}

	PrintRenderErrors(errs)

	got := buf.String()

	// Should contain component name and error
	assert.Contains(t, got, "cache", "should contain component name")
	assert.Contains(t, got, "no matching transformer", "should contain error message")

	// Should NOT show available transformers section
	assert.NotContains(t, got, "Available transformers", "should not show available section when none exist")
}

func TestPrintRenderErrors_TransformError(t *testing.T) {
	// Setup: capture stderr output
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	// Create a TransformError
	errs := []error{
		&core.TransformError{
			ComponentName:  "api",
			TransformerFQN: "example.com/transformers@v1#APITransformer",
			Cause:          fmt.Errorf("missing required field: port"),
		},
	}

	PrintRenderErrors(errs)

	got := buf.String()

	// Should contain component name, transformer FQN, and cause
	assert.Contains(t, got, "api", "should contain component name")
	assert.Contains(t, got, "transform failed", "should contain transform failed message")
	assert.Contains(t, got, "APITransformer", "should contain transformer FQN")
	assert.Contains(t, got, "missing required field: port", "should contain cause error")
}

func TestPrintRenderErrors_GenericError(t *testing.T) {
	// Setup: capture stderr output
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	// Create a generic error
	errs := []error{
		fmt.Errorf("unexpected render failure"),
	}

	PrintRenderErrors(errs)

	got := buf.String()

	// Should contain raw error string
	assert.Contains(t, got, "unexpected render failure", "should contain generic error message")
}

func TestPrintRenderErrors_MultipleErrors(t *testing.T) {
	// Setup: capture stderr output
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	// Create multiple errors
	errs := []error{
		&build.UnmatchedComponentError{
			ComponentName: "worker",
			Available:     nil,
		},
		&core.TransformError{
			ComponentName:  "api",
			TransformerFQN: "example.com/transformers@v1#APITransformer",
			Cause:          fmt.Errorf("validation failed"),
		},
	}

	PrintRenderErrors(errs)

	got := buf.String()

	// Should contain summary line
	assert.Contains(t, got, "render completed with errors", "should contain summary")

	// Should contain both error details
	assert.Contains(t, got, "worker", "should contain first component")
	assert.Contains(t, got, "api", "should contain second component")
	assert.Contains(t, got, "no matching transformer", "should contain unmatched error")
	assert.Contains(t, got, "transform failed", "should contain transform error")
}
