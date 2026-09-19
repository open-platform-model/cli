package render

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	liberrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/helper/objectset"
	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/output"
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

// The kernel's values-validation error tree (a -f file violating #config)
// prints through the grouped branch: one summary line and every position
// attributed to the source's origin.
func TestPrintValidationError_UsesGroupedFormatting(t *testing.T) {
	k := kernel.New()
	schema := cuecontext.New().CompileString(`close({
		media?: [Name=string]: {
			mountPath: string
			type:      "pvc" | *"emptyDir"
			size:      string
		}
	})`, cue.Filename("module.cue"))
	require.NoError(t, schema.Err())
	src, err := k.LoadSourceFromBytes("values.cue", []byte(`{
		test: "test"
		media: {
			test: "test"
		}
	}`))
	require.NoError(t, err)

	_, cfgErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, cfgErr)

	logs, details := captureValidationOutput(t, cfgErr)

	assert.Contains(t, logs, "render failed: 2 issues")
	assert.Contains(t, details, "field not allowed")
	assert.Contains(t, details, "values.test")
	assert.Contains(t, details, "> values.cue:2:3")
	assert.Contains(t, details, "> values.cue:4:10")
	assert.Contains(t, details, "conflicting values \"test\"")
	assert.NotContains(t, logs, "values do not satisfy #config")
}

// A duplicate-identity refusal reaches the user in the CLI's two streams:
// the library's header line under render failed, every shared identity with
// both producing components as details. The words are the library's, so the
// operator refuses the same module identically; only the split is the CLI's.
func TestPrintValidationError_DuplicateIdentitiesSplitsHeaderAndRows(t *testing.T) {
	dupErr := &objectset.DuplicateIdentitiesError{Duplicates: []objectset.Duplicate{{
		Identity: objectset.Identity{
			APIVersion: "opmodel.dev/v1alpha1",
			Kind:       "TransformerRegistration",
			Namespace:  "backup-system",
			Name:       "backup-system.k8up",
		},
		Producers: []objectset.Producer{
			{Component: "registration", Transformer: "opmodel.dev/catalogs/opm/transformers/transformer-registration-transformer@4.4.0"},
			{Component: "registration-copy", Transformer: "opmodel.dev/catalogs/opm/transformers/transformer-registration-transformer@4.4.0"},
		},
	}}}

	logs, details := captureValidationOutput(t, fmt.Errorf("wrapped: %w", dupErr))

	header, rows, found := strings.Cut(dupErr.Error(), "\n")
	require.True(t, found, "the library's message is a header line plus at least one identity row")
	assert.Contains(t, logs, "render failed: "+header)
	assert.NotContains(t, logs, rows, "the identity rows are details, not part of the header line")
	assert.Contains(t, details, "opmodel.dev/v1alpha1 TransformerRegistration backup-system/backup-system.k8up")
	assert.Contains(t, details, `component "registration"`)
	assert.Contains(t, details, `component "registration-copy"`)
}
