package cmdutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	liberrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/output"
	pkgerrors "github.com/open-platform-model/cli/pkg/errors"
)

// captureOutput runs fn and returns the log stream and the details stream
// (stderr) it wrote.
func captureOutput(t *testing.T, fn func()) (logs, details string) {
	t.Helper()
	var logBuf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&logBuf)

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	fn()
	require.NoError(t, w.Close())
	raw, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return logBuf.String(), string(raw)
}

// TestPrintValidationError_KernelErrorGroupsPositions asserts the kernel's
// values-validation error tree, framed by the caller, prints as one summary
// line counting the distinct issues and a grouped block naming each source
// position — never as one flattened log line per CUE error.
func TestPrintValidationError_KernelErrorGroupsPositions(t *testing.T) {
	k := kernel.New()
	schema := cuecontext.New().CompileString(`close({
		media?: [Name=string]: {
			type: "pvc" | *"emptyDir"
		}
	})`, cue.Filename("module.cue"))
	require.NoError(t, schema.Err())
	src, err := k.LoadSourceFromBytes("values.cue", []byte("{\n\ttest: \"test\"\n\tmedia: test: \"test\"\n}\n"))
	require.NoError(t, err)
	_, cfgErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, cfgErr)

	logs, details := captureOutput(t, func() {
		PrintValidationError("values do not satisfy #config", fmt.Errorf("module %q: values do not satisfy #config: %w", "demo", cfgErr))
	})

	assert.Contains(t, logs, "values do not satisfy #config: 2 issues")
	assert.NotContains(t, logs, "values do not satisfy #config: - ", "the flattened one-line-per-error shape must not appear")
	assert.Contains(t, details, "field not allowed")
	assert.Contains(t, details, "values.test")
	assert.Contains(t, details, "> values.cue:2:2", "the disallowed field is attributed to the source's origin")
	assert.Contains(t, details, "conflicting values")
	assert.Contains(t, details, "values.media.test")
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

func TestFormatUnresolvedDemands(t *testing.T) {
	demands := []liberrors.UnresolvedDemand{
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
	}

	got := FormatUnresolvedDemands(demands)
	assert.Contains(t, got, `component "web": unresolved resource demand "opmodel.dev/catalogs/opm/resources/container@v1beta1"`)
	assert.Contains(t, got, "implemented at: opmodel.dev/catalogs/opm/resources/container@v2")
	assert.Contains(t, got, `component "api": unresolved trait demand`)
	assert.Contains(t, got, "nothing on this platform implements this contract")
}

// TestFormatUnresolvedDemands_DefinedByNamesTheCatalog covers the arm the row
// could not previously express: the demanded contract IS defined by an
// enabled catalog and implemented by nothing, which is a different situation
// from a contract no enabled catalog defines at all.
func TestFormatUnresolvedDemands_DefinedByNamesTheCatalog(t *testing.T) {
	got := FormatUnresolvedDemands([]liberrors.UnresolvedDemand{
		{
			Component: "api",
			FQN:       "opmodel.dev/catalogs/opm/traits/backup@v1alpha1",
			Kind:      "trait",
			DefinedBy: "opmodel.dev/catalogs/opm@v4",
		},
		{
			Component:    "web",
			FQN:          "opmodel.dev/catalogs/opm/resources/container@v1beta1",
			Kind:         "resource",
			Alternatives: []string{"opmodel.dev/catalogs/opm/resources/container@v2"},
			DefinedBy:    "opmodel.dev/catalogs/opm@v4",
		},
		{
			Component: "worker",
			FQN:       "example.com/catalogs/other/traits/mesh@v1",
			Kind:      "trait",
		},
	})

	assert.Contains(t, got, `  defined by "opmodel.dev/catalogs/opm@v4", implemented by nothing on this platform`)
	// A row that also carries alternatives keeps the actionable line and
	// gains the catalog beside it.
	assert.Contains(t, got, "  defined by \"opmodel.dev/catalogs/opm@v4\"\n  implemented at: opmodel.dev/catalogs/opm/resources/container@v2")
	// A row no enabled catalog defines keeps today's wording, naming no
	// catalog.
	assert.Contains(t, got, "  nothing on this platform implements this contract")
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
