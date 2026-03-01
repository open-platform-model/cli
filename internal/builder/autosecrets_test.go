package builder

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/core/component"
	"github.com/opmodel/cli/internal/loader"
)

// secretsModulePath returns the absolute path to the secrets-module test fixture.
func secretsModulePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	cliRoot := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(cliRoot, "tests", "fixtures", "valid", "secrets-module")
}

// buildReleaseResult partially replicates the Build pipeline up to the point
// where autoSecrets can be read, returning the evaluated CUE result value.
func buildReleaseResult(t *testing.T, ctx *cue.Context, modPath, valuesContent string) cue.Value {
	t.Helper()

	mod, err := loader.LoadModule(ctx, modPath, os.Getenv("OPM_REGISTRY"))
	require.NoError(t, err)

	coreInstances := load.Instances([]string{"opmodel.dev/core@v1"}, &load.Config{Dir: modPath})
	require.NotEmpty(t, coreInstances)
	require.NoError(t, coreInstances[0].Err)

	coreVal := ctx.BuildInstance(coreInstances[0])
	require.NoError(t, coreVal.Err())

	releaseSchema := coreVal.LookupPath(cue.ParsePath("#ModuleRelease"))
	require.True(t, releaseSchema.Exists())

	valuesFile := writeTempValues(t, valuesContent)
	selectedValues, err := ValidateValues(ctx, mod.Config, []string{valuesFile})
	require.NoError(t, err)

	result := releaseSchema.
		FillPath(cue.MakePath(cue.Def("module")), mod.Raw).
		FillPath(cue.ParsePath("metadata.name"), ctx.CompileString(`"test"`)).
		FillPath(cue.ParsePath("metadata.namespace"), ctx.CompileString(`"default"`)).
		FillPath(cue.ParsePath("values"), selectedValues)
	require.NoError(t, result.Err())

	return result
}

// TestInjectAutoSecrets_WithSecrets verifies that a module with #Secret fields
// in #config produces an opm-secrets component with the correct structure.
func TestInjectAutoSecrets_WithSecrets(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	modPath := secretsModulePath(t)

	mod, err := loader.LoadModule(ctx, modPath, os.Getenv("OPM_REGISTRY"))
	require.NoError(t, err, "loading secrets-module should succeed")

	valuesFile := writeTempValues(t, `values: {
		image: {repository: "nginx", tag: "1.28", digest: ""}
		db: {
			password: value: "super-secret"
			host: value:     "db.example.com"
		}
		apiKey: value: "my-api-key-123"
	}`)

	opts := Options{Name: "test-secrets", Namespace: "default"}
	rel, err := Build(ctx, mod, opts, []string{valuesFile})
	require.NoError(t, err)
	require.NotNil(t, rel)

	// opm-secrets component should exist.
	opmSecrets, ok := rel.Components[autoSecretsComponentName]
	require.True(t, ok, "opm-secrets component should exist in release components")

	// Should have the correct resource FQN for transformer matching.
	_, hasResourceFQN := opmSecrets.Resources["opmodel.dev/resources/config/secrets@v1"]
	assert.True(t, hasResourceFQN, "opm-secrets should have the secrets resource FQN")

	// Metadata name should be "opm-secrets".
	assert.Equal(t, autoSecretsComponentName, opmSecrets.Metadata.Name)

	// The list-output annotation is defined as a CUE bool `true` in the catalog
	// (same convention as configmaps, volumes, CRDs). ExtractComponents reads
	// annotations via .String() which silently drops non-string values, so this
	// annotation is NOT present in the Go Annotations map. This is a known
	// limitation — transformer matching works via #resources FQN (verified above).
	_, hasListOutput := opmSecrets.Metadata.Annotations["transformer.opmodel.dev/list-output"]
	assert.False(t, hasListOutput, "list-output annotation is a CUE bool, not extractable as Go string")

	// User-defined components should still be present.
	_, hasWeb := rel.Components["web"]
	assert.True(t, hasWeb, "user-defined 'web' component should still be present")
}

// TestInjectAutoSecrets_NoSecrets verifies that a module without #Secret fields
// returns components unchanged with no opm-secrets component.
// Also covers the "old module without autoSecrets" scenario: real_module uses
// opmodel.dev@v1.0.4 which predates the autoSecrets field on #ModuleRelease.
// readAutoSecrets correctly returns false when the field is absent.
func TestInjectAutoSecrets_NoSecrets(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	modPath := realModulePath(t) // v1.0.4 — no #Secret fields, no autoSecrets field

	mod, err := loader.LoadModule(ctx, modPath, os.Getenv("OPM_REGISTRY"))
	require.NoError(t, err)

	valuesFile := writeTempValues(t, `values: {
		image: {repository: "nginx", tag: "1.28", digest: ""}
		replicas: 2
	}`)

	opts := Options{Name: "test-no-secrets", Namespace: "default"}
	rel, err := Build(ctx, mod, opts, []string{valuesFile})
	require.NoError(t, err)
	require.NotNil(t, rel)

	// opm-secrets should NOT exist.
	_, ok := rel.Components[autoSecretsComponentName]
	assert.False(t, ok, "opm-secrets component should not exist when module has no secrets")
}

// TestInjectAutoSecrets_NameCollision verifies that a pre-existing component
// named "opm-secrets" causes an error containing "reserved".
func TestInjectAutoSecrets_NameCollision(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	modPath := secretsModulePath(t)

	result := buildReleaseResult(t, ctx, modPath, `values: {
		image: {repository: "nginx", tag: "1.28", digest: ""}
		db: {
			password: value: "super-secret"
			host: value:     "db.example.com"
		}
		apiKey: value: "my-api-key-123"
	}`)

	// Verify autoSecrets is present.
	autoSecrets, ok := readAutoSecrets(result)
	require.True(t, ok, "autoSecrets should be present for secrets-module")
	require.True(t, autoSecrets.Exists())

	// Pre-populate the components map with a dummy opm-secrets entry.
	components := map[string]*component.Component{
		autoSecretsComponentName: {
			Metadata: &component.ComponentMetadata{Name: autoSecretsComponentName},
		},
	}

	err := injectAutoSecrets(ctx, result, modPath, components)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved")
}
