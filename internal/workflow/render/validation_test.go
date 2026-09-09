package render

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
	"github.com/open-platform-model/cli/pkg/validate"
)

// captureValidationOutput runs printValidationError and returns the log
// stream and the details stream (stderr) it wrote.
func captureValidationOutput(t *testing.T, err error) (logs, details string) {
	t.Helper()
	var logBuf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&logBuf)

	oldStderr := os.Stderr
	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	printValidationError(err)
	require.NoError(t, w.Close())
	raw, readErr := io.ReadAll(r)
	require.NoError(t, readErr)
	require.NoError(t, r.Close())
	return logBuf.String(), string(raw)
}

func TestPrintValidationError_RenderErrorPrintsDiagnostics(t *testing.T) {
	renderErr := &kernel.RenderError{
		Err: &liberrors.UnresolvedDemandsError{Demands: []liberrors.UnresolvedDemand{
			{Component: "web", Kind: "trait", FQN: "opmodel.dev/catalogs/opm@v4#Expose", Alternatives: []string{"opmodel.dev/catalogs/k8s@v1#Expose"}},
		}},
		Diagnostics: kernel.RenderDiagnostics{
			Unresolved: []liberrors.UnresolvedDemand{
				{Component: "web", Kind: "trait", FQN: "opmodel.dev/catalogs/opm@v4#Expose", Alternatives: []string{"opmodel.dev/catalogs/k8s@v1#Expose"}},
			},
			Unmatched: []liberrors.UnmatchedComponent{{
				Component: "worker",
				Candidates: []liberrors.CandidateVerdict{
					{Transformer: "opmodel.dev/catalogs/opm@v4#DeploymentTransformer", MissingLabels: []string{"workload.opmodel.dev/type=stateless"}},
				},
			}},
			OverSubscribed: []liberrors.OverSubscribedContract{
				{Key: "opmodel.dev/contracts/ingress@v1", Catalogs: []string{"opmodel.dev/catalogs/a@v1", "opmodel.dev/catalogs/b@v1"}},
			},
			FailedPairs: []kernel.RenderPair{{Component: "db", Transformer: "opmodel.dev/catalogs/opm@v4#StatefulSetTransformer"}},
		},
	}

	logs, details := captureValidationOutput(t, fmt.Errorf("wrapped: %w", renderErr))

	assert.Contains(t, logs, "render failed")
	assert.Contains(t, logs, renderErr.Err.Error(), "the kernel's message is printed verbatim")
	assert.Contains(t, details, `component "web": unresolved trait demand "opmodel.dev/catalogs/opm@v4#Expose"`)
	assert.Contains(t, details, "implemented at: opmodel.dev/catalogs/k8s@v1#Expose")
	assert.Contains(t, details, `component "worker": no transformer matched`)
	assert.NotContains(t, details, "candidate", "the default output stays one line per unmatched component")
	assert.Contains(t, details, `contract "opmodel.dev/contracts/ingress@v1"`)
	assert.Contains(t, details, "opmodel.dev/catalogs/a@v1, opmodel.dev/catalogs/b@v1")
	assert.Contains(t, details, `component "db": transformer opmodel.dev/catalogs/opm@v4#StatefulSetTransformer failed`)
}

// The candidate matrix the kernel now carries on the unmatched rows is
// printed in verbose mode only: the labels the predicate found missing, or
// the FQNs the always-unify rung conflicted at (looked up on the Unify rows).
func TestFormatRenderDiagnostics_UnmatchedCandidatesVerbose(t *testing.T) {
	d := kernel.RenderDiagnostics{
		Unmatched: []liberrors.UnmatchedComponent{{
			Component: "worker",
			Candidates: []liberrors.CandidateVerdict{
				{Transformer: "opmodel.dev/catalogs/opm@v4#DeploymentTransformer", MissingLabels: []string{"workload.opmodel.dev/type=stateless"}},
				{Transformer: "opmodel.dev/catalogs/opm@v4#StatefulSetTransformer"},
				{Transformer: "opmodel.dev/catalogs/opm@v4#JobTransformer", Matched: true},
			},
		}},
		Unify: []liberrors.UnifyRefusal{
			{Component: "worker", Transformer: "opmodel.dev/catalogs/opm@v4#StatefulSetTransformer", Conflicts: []string{"opmodel.dev/catalogs/opm/resources/container@v1"}},
		},
	}

	compact := formatRenderDiagnostics(d, false)
	assert.Equal(t, `component "worker": no transformer matched`, compact)

	verbose := formatRenderDiagnostics(d, true)
	assert.Contains(t, verbose, `component "worker": no transformer matched`)
	assert.Contains(t, verbose, `  candidate "opmodel.dev/catalogs/opm@v4#DeploymentTransformer" did not match: missing labels workload.opmodel.dev/type=stateless`)
	assert.Contains(t, verbose, `  candidate "opmodel.dev/catalogs/opm@v4#StatefulSetTransformer" did not match: bodies conflict at opmodel.dev/catalogs/opm/resources/container@v1`)
	assert.NotContains(t, verbose, "JobTransformer", "a matched candidate is not a refusal")
}

func TestPrintValidationError_SkewErrorVerbatim(t *testing.T) {
	skew := &liberrors.SkewError{Path: "opmodel.dev/catalogs/opm@v4", ModuleVersion: "v4.1.0", PlatformVersion: "v4.0.1"}

	logs, _ := captureValidationOutput(t, errors.Join(skew))

	assert.Contains(t, logs, "render failed")
	assert.Contains(t, logs, skew.Error(), "the kernel's skew message names the path and both versions")
}

func TestFormatRenderDiagnostics_EmptyIsEmpty(t *testing.T) {
	assert.Empty(t, formatRenderDiagnostics(kernel.RenderDiagnostics{Pairs: []kernel.RenderPair{{Component: "web", Transformer: "x"}}}, true),
		"matched pairs are not refusals and are not repeated")
}

func TestPrintValidationError_UsesGroupedFormatting(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`close({
		media?: [Name=string]: {
			mountPath: string
			type:      "pvc" | *"emptyDir"
			size:      string
		}
	})`, cue.Filename("module.cue"))
	values := ctx.CompileString(`{
		test: "test"
		media: {
			test: "test"
		}
	}`, cue.Filename("values.cue"))

	_, cfgErr := validate.Config(schema, []cue.Value{values}, "module", "demo")
	require.NotNil(t, cfgErr)

	var logBuf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&logBuf)

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	printValidationError(cfgErr)
	require.NoError(t, w.Close())

	details, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	assert.Contains(t, logBuf.String(), "render failed: 2 issues")
	assert.Contains(t, string(details), "field not allowed")
	assert.Contains(t, string(details), "values.test")
	assert.Contains(t, string(details), "> values.cue:2:3")
	assert.Contains(t, string(details), "> values.cue:4:10")
	assert.Contains(t, string(details), "conflicting values \"test\"")
	assert.NotContains(t, logBuf.String(), "values do not satisfy #config")
}
