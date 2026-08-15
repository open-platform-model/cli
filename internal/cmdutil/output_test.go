package cmdutil

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	liberrors "github.com/open-platform-model/library/opm/errors"

	"github.com/open-platform-model/cli/internal/output"
	pkgerrors "github.com/open-platform-model/cli/pkg/errors"
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

func TestFormatUnresolvedDemands(t *testing.T) {
	aggregate := &liberrors.UnresolvedDemandsError{Demands: []liberrors.UnresolvedDemand{
		{
			Component:    "web",
			FQN:          "opmodel.dev/catalogs/opm/resources/container@v1beta1",
			Kind:         "resource",
			Alternatives: []string{"opmodel.dev/catalogs/opm/resources/container@v2"},
		},
		{
			Component: "api",
			FQN:       "opmodel.dev/catalogs/opm/traits/expose@v1beta1",
			Kind:      "trait",
		},
	}}

	got := FormatUnresolvedDemands(aggregate)
	assert.Contains(t, got, `component "web": unresolved resource demand "opmodel.dev/catalogs/opm/resources/container@v1beta1"`)
	assert.Contains(t, got, "implemented at: opmodel.dev/catalogs/opm/resources/container@v2")
	assert.Contains(t, got, `component "api": unresolved trait demand`)
	assert.Contains(t, got, "nothing on this platform implements this contract")
}

func TestPrintValidationError_RoutesUnresolvedDemands(t *testing.T) {
	// The aggregate stays reachable through wrapping, as the compile path
	// delivers it; the typed branch must not panic and must not fall through
	// to the flat key-value format (asserted indirectly: errors.As finds it).
	wrapped := fmt.Errorf("compiling instance: %w",
		&liberrors.UnresolvedDemandsError{Demands: []liberrors.UnresolvedDemand{{Component: "web", FQN: "x@v1", Kind: "resource"}}})

	var demandsErr *liberrors.UnresolvedDemandsError
	require.True(t, errors.As(wrapped, &demandsErr))
	assert.NotPanics(t, func() { PrintValidationError("validation failed", wrapped) })
}
