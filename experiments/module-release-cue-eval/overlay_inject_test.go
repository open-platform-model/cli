package modulereleasecueeval

// ---------------------------------------------------------------------------
// Decisions 8-9: Approach C real-module mechanic and end-to-end
//
// Strategy B (pure Approach C, requires OPM_REGISTRY):
//   The module ALREADY imports opmodel.dev/core@v0 in its cue.mod/module.cue.
//   We load opmodel.dev/core@v0 from within the module's directory — CUE
//   resolves it from the module's pinned deps without a separate catalog load.
//   The module itself is loaded in the SAME *cue.Context, enabling FillPath.
//
// Decision 8: #ModuleRelease and the module value are accessible from the same
//             context when loaded this way (no overlay needed).
// Decision 9: A fully-concrete #ModuleRelease can be produced end-to-end from a
//             real module using only dep-resolution loading (no catalog dual-load).
//
// Both tests require OPM_REGISTRY to be set. They are skipped otherwise.
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOverlay_ReleaseTypeAccessible proves that after loading the real_module
// and opmodel.dev/core@v0 (from the module's pinned deps), #ModuleRelease is
// accessible and is a non-errored, non-concrete (schema) cue.Value.
//
// This is the core Approach C mechanic: the module's existing dep on
// opmodel.dev/core@v0 makes #ModuleRelease available without a separate catalog
// load — just load the package from within the module's directory.
func TestOverlay_ReleaseTypeAccessible(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	_, releaseSchema := buildRealModuleWithSchema(t, ctx)

	// #ModuleRelease is loaded from the core package resolved via the module's deps.
	assert.True(t, releaseSchema.Exists(), "#ModuleRelease should be accessible")
	assert.NoError(t, releaseSchema.Err(), "#ModuleRelease should not be errored")

	// It is a schema — should not be concrete.
	concreteErr := releaseSchema.Validate(cue.Concrete(true))
	assert.Error(t, concreteErr, "#ModuleRelease should be a schema (not concrete)")
}

// TestOverlay_ModulePathExists proves that the module value (the real module
// itself) is accessible alongside #ModuleRelease in the same context.
// This confirms that both the module content AND the schema can coexist in
// the same *cue.Context — the starting point for FillPath injection.
func TestOverlay_ModulePathExists(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	moduleVal, releaseSchema := buildRealModuleWithSchema(t, ctx)

	// The real module defines: metadata, #config, #components, values.
	metadataName, err := moduleVal.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err, "real module metadata.name should be concrete")
	assert.Equal(t, "exp-release-module", metadataName)

	// The schema from core is also available.
	assert.True(t, releaseSchema.Exists(), "#ModuleRelease should coexist with module in context")
}

// TestOverlay_EndToEnd proves the complete Approach C flow:
//  1. Load core from module's pinned deps → get #ModuleRelease schema
//  2. Load module in the same context
//  3. FillPath: inject module as #module, fill metadata.name, namespace, values
//  4. Let CUE evaluate uuid, labels, components
//  5. Read back concrete values from Go
//
// This is the target state for the Approach C design.
func TestOverlay_EndToEnd(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	moduleVal, releaseSchema := buildRealModuleWithSchema(t, ctx)

	// Step 2-3: Inject the module and fill release metadata + values.
	result := fillRelease(releaseSchema, moduleVal, "exp-release", "staging", `{
		image:    "nginx:1.28"
		replicas: 3
	}`)

	// Step 4: Check for errors.
	if err := result.Err(); err != nil {
		t.Logf("Approach C end-to-end ERRORS: %v", err)
		require.NoError(t, result.Err(), "end-to-end Approach C should produce a non-errored result")
	}

	// Step 5: Read back concrete values.
	t.Log("Approach C end-to-end SUCCEEDED")

	uuid, err := result.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err, "metadata.uuid should be concrete")
	t.Logf("  metadata.uuid: %s", uuid)
	assert.Regexp(t, `^[0-9a-f-]{36}$`, uuid)

	version, err := result.LookupPath(cue.ParsePath("metadata.version")).String()
	require.NoError(t, err, "metadata.version should be concrete")
	assert.Equal(t, "0.1.0", version)

	labels := extractLabels(t, result)
	t.Logf("  metadata.labels: %v", labels)
	assert.Equal(t, "exp-release", labels["module-release.opmodel.dev/name"])

	componentsVal := result.LookupPath(cue.ParsePath("components"))
	assert.True(t, componentsVal.Exists(), "components should be present")
	assert.NoError(t, componentsVal.Err(), "components should not error")
}

// TestOverlay_SelfModuleInjection tests a key Approach C variant:
// does injecting the module value into #module! succeed without issue?
// The module value is clean (no extra fields beyond #Module shape) because
// it was loaded independently without any overlay.
// This test records whether close(#Module) accepts the disk-loaded module value.
func TestOverlay_SelfModuleInjection(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	moduleVal, releaseSchema := buildRealModuleWithSchema(t, ctx)

	// Inject the module value loaded from disk (no synthetic extra fields).
	withModule := releaseSchema.FillPath(cue.MakePath(cue.Def("module")), moduleVal)

	if err := withModule.Err(); err != nil {
		t.Logf("RESULT: module injection ERRORS: %v", err)
		t.Log("Unexpected — a module constrained by core.#Module should satisfy #Module.")
		require.NoError(t, err, "disk-loaded core.#Module value must inject without error")
	} else {
		t.Log("RESULT: module injection SUCCEEDS — clean disk-loaded #Module satisfies #Module constraint.")
		assert.True(t, withModule.Exists())
	}
}

// extractCleanModuleValue loads the real module WITHOUT any overlay.
// Used as a helper for tests that need the module value without extra fields.
func extractCleanModuleValue(t *testing.T, ctx *cue.Context, path string) cue.Value {
	t.Helper()
	instances := load.Instances([]string{"."}, &load.Config{Dir: path})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)
	val := ctx.BuildInstance(instances[0])
	require.NoError(t, val.Err(), "clean module BuildInstance should not error")
	return val
}
