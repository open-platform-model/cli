package builder

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/loader"
)

// requireRegistry skips the test if OPM_REGISTRY is not set and configures
// CUE_REGISTRY for the duration of the test.
func requireRegistry(t *testing.T) {
	t.Helper()
	registry := os.Getenv("OPM_REGISTRY")
	if registry == "" {
		t.Skip("OPM_REGISTRY not set — skipping registry-dependent test")
	}
	t.Setenv("CUE_REGISTRY", registry)
}

// realModulePath returns the absolute path to the shared real_module test fixture
// from the module-release-cue-eval experiment.
func realModulePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	// internal/builder/ → internal/ → cli/ → experiments/module-release-cue-eval/testdata/real_module
	cliRoot := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(cliRoot, "experiments", "module-release-cue-eval", "testdata", "real_module")
}

var uuidRegexp = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// TestBuild_EndToEnd tests a successful build with a real module fixture and valid values.
// Requires OPM_REGISTRY to be set; skipped otherwise.
func TestBuild_EndToEnd(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	modPath := realModulePath(t)

	mod, err := loader.LoadModule(ctx, modPath, os.Getenv("OPM_REGISTRY"))
	require.NoError(t, err, "loading real_module should succeed")

	// Provide concrete values that satisfy #config.
	valuesFile := writeTempValues(t, `values: {
		image:    "nginx:1.28"
		replicas: 2
	}`)

	opts := Options{Name: "test-release", Namespace: "staging"}
	rel, err := Build(ctx, mod, opts, []string{valuesFile})
	require.NoError(t, err)
	require.NotNil(t, rel)
	require.NotNil(t, rel.Metadata)

	// UUID should be a valid UUID.
	assert.Regexp(t, uuidRegexp, rel.Metadata.UUID, "UUID should be a valid UUID")

	// Labels should be populated (CUE evaluation sets them).
	assert.NotEmpty(t, rel.Metadata.Labels, "labels should be populated by CUE evaluation")

	// Components should be present.
	assert.NotEmpty(t, rel.Components, "components should not be empty")

	// Name and namespace should match opts.
	assert.Equal(t, opts.Name, rel.Metadata.Name)
	assert.Equal(t, opts.Namespace, rel.Metadata.Namespace)
}

// TestBuild_ValuesViolateConfig ensures a *core.ValidationError is returned when
// provided values do not satisfy the module's #config schema.
func TestBuild_ValuesViolateConfig(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	modPath := realModulePath(t)

	mod, err := loader.LoadModule(ctx, modPath, os.Getenv("OPM_REGISTRY"))
	require.NoError(t, err)

	// replicas must be >=1; provide 0 to violate the constraint.
	valuesFile := writeTempValues(t, `values: {
		image:    "nginx:1.0"
		replicas: 0
	}`)

	_, err = Build(ctx, mod, Options{Name: "rel", Namespace: "default"}, []string{valuesFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed", "should be a ValidationError")
}

// TestBuild_NonConcreteResult ensures that if values are incomplete and leave
// #ModuleRelease non-concrete, a *core.ValidationError is returned.
func TestBuild_NonConcreteResult(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	modPath := realModulePath(t)

	mod, err := loader.LoadModule(ctx, modPath, os.Getenv("OPM_REGISTRY"))
	require.NoError(t, err)

	// Provide values with image but omit replicas — #config.replicas has a default (>=1 | *1)
	// so this will actually be concrete. Use a values file that is entirely empty to
	// test the "no values" path instead.
	//
	// The real_module's #config has defaults for both fields, so omitting them still
	// produces a concrete release. Instead, confirm that providing an explicit abstract
	// value (CUE string type) triggers the concreteness error.
	valuesFile := writeTempValues(t, `values: {
		image:    string
		replicas: 1
	}`)

	_, err = Build(ctx, mod, Options{Name: "rel", Namespace: "default"}, []string{valuesFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

// TestBuild_UUIDIsDeterministic ensures that calling Build twice with identical
// inputs produces the same UUID.
func TestBuild_UUIDIsDeterministic(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	modPath := realModulePath(t)

	mod, err := loader.LoadModule(ctx, modPath, os.Getenv("OPM_REGISTRY"))
	require.NoError(t, err)

	valuesFile := writeTempValues(t, `values: {
		image:    "nginx:1.0"
		replicas: 1
	}`)

	opts := Options{Name: "same-release", Namespace: "same-ns"}

	rel1, err := Build(ctx, mod, opts, []string{valuesFile})
	require.NoError(t, err)

	rel2, err := Build(ctx, mod, opts, []string{valuesFile})
	require.NoError(t, err)

	assert.Equal(t, rel1.Metadata.UUID, rel2.Metadata.UUID, "UUID must be deterministic across builds")
}

// writeTempValues writes a CUE string to a temp file and returns its path.
func writeTempValues(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "values-*.cue")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}
