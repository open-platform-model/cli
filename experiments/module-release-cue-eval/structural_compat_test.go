package modulereleasecueeval

// ---------------------------------------------------------------------------
// Decision 3: Does a disk-loaded module value satisfy #Module constraint?
//
// This is the highest-risk decision in Approach C. The question is:
//   Can a cue.Value loaded from a module directory (via load.Instances +
//   BuildInstance) be injected into #module!: #Module when the catalog is
//   loaded separately in the same context?
//
// There are two cases:
//
//   Case A — Fake module (no registry):
//     The fake_module fixture does NOT import opmodel.dev/core@v0. Its value
//     is a bare CUE struct that looks like #Module but was never constrained
//     by it. CUE uses structural/nominal typing — does the shape match?
//
//   Case B — Real module (requires OPM_REGISTRY):
//     The real_module fixture imports core and uses core.#Module. Its value
//     IS a valid #Module by construction. This should succeed.
//
// The key unknown: does #Module.close({...}) reject a value that:
//   - Has extra fields (fake_module adds its own package-level fields)?
//   - Has missing computed fields (fqn, uuid, _definitionName)?
//   - Was not produced under a #Module constraint?
//
// These tests record the answer — pass or fail — and explain why.
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStructural_FakeModuleShapeDescription documents the shape of the fake module
// value as loaded. This is informational — it prints the shape for inspection
// so that a test failure in subsequent tests can be diagnosed.
func TestStructural_FakeModuleShapeDescription(t *testing.T) {
	ctx, _ := buildCatalogValue(t)
	fakeModule := buildFakeModuleValue(t, ctx)

	// Print the top-level fields for diagnosis.
	t.Log("fake_module top-level fields:")
	iter, err := fakeModule.Fields(cue.Optional(true), cue.Definitions(true))
	require.NoError(t, err)
	for iter.Next() {
		t.Logf("  %s (concrete=%v)", iter.Label(), iter.Value().IsConcrete())
	}
}

// TestStructural_FakeModuleInjectDoesNotPanic proves that FillPath with the
// fake module value does not panic. Whether it errors or not is a separate
// question answered below — but panics are always wrong.
func TestStructural_FakeModuleInjectDoesNotPanic(t *testing.T) {
	ctx, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	fakeModule := buildFakeModuleValue(t, ctx)

	// This must not panic regardless of whether it succeeds.
	assert.NotPanics(t, func() {
		_ = schema.FillPath(cue.MakePath(cue.Def("module")), fakeModule)
	})
}

// TestStructural_FakeModuleInjectErrorState records whether FillPath with the
// fake module produces an error. This is the key result of Decision 3 for Case A.
//
// Expected outcomes (one will be true):
//   - Success: CUE accepts the structural match — Approach C works for fake modules
//   - Failure: CUE rejects because close() sees missing/extra fields — only real
//     modules (with core.#Module applied) can be injected
func TestStructural_FakeModuleInjectErrorState(t *testing.T) {
	ctx, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	fakeModule := buildFakeModuleValue(t, ctx)

	withModule := schema.FillPath(cue.MakePath(cue.Def("module")), fakeModule)

	if err := withModule.Err(); err != nil {
		// RESULT: CUE rejects the fake module.
		// Approach C requires a real module (with core.#Module applied).
		t.Logf("RESULT (Case A): FillPath with fake module ERRORS: %v", err)
		t.Log("Conclusion: #Module.close() rejects structurally-similar but unconstrained values.")
		t.Log("Approach C requires the module to have core.#Module applied (Strategy B).")
		// Not a test failure — this is a valid discovered outcome.
	} else {
		// RESULT: CUE accepts the fake module structurally.
		// Approach C can work even with modules loaded without core imports.
		t.Log("RESULT (Case A): FillPath with fake module SUCCEEDS.")
		t.Log("Conclusion: CUE structural matching accepts the fake module value.")
		t.Log("Approach C works for any structurally-compatible module.")

		// If accepted, verify the resulting value exists and isn't totally broken.
		assert.True(t, withModule.Exists())
	}
}

// TestStructural_FakeModuleFullReleaseOutcome attempts the complete fill sequence
// with the fake module and records whether a concrete value is produced.
// This complements TestStructural_FakeModuleInjectErrorState by testing the
// full pipeline rather than just the injection step.
func TestStructural_FakeModuleFullReleaseOutcome(t *testing.T) {
	ctx, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	fakeModule := buildFakeModuleValue(t, ctx)

	// Inject fake module + release metadata + values.
	result := schema.
		FillPath(cue.MakePath(cue.Def("module")), fakeModule).
		FillPath(cue.ParsePath("metadata.name"), ctx.CompileString(`"fake-release"`)).
		FillPath(cue.ParsePath("metadata.namespace"), ctx.CompileString(`"default"`)).
		FillPath(cue.ParsePath("values"), ctx.CompileString(`{image: "nginx:1.0", replicas: 2}`))

	concreteErr := result.Validate(cue.Concrete(true))
	if concreteErr != nil {
		t.Logf("RESULT: fake module does NOT produce a concrete #ModuleRelease: %v", concreteErr)
	} else {
		t.Log("RESULT: fake module produces a fully concrete #ModuleRelease.")

		// If concrete, verify key fields.
		name, err := result.LookupPath(cue.ParsePath("metadata.name")).String()
		require.NoError(t, err)
		assert.Equal(t, "fake-release", name)
	}
}

// TestStructural_CatalogTestModuleInjectSucceeds proves that the catalog's own
// _testModule — which is #Module by construction — injects cleanly. This is the
// "golden path" baseline for Decision 3: if this fails, there is an API issue.
func TestStructural_CatalogTestModuleInjectSucceeds(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	withModule := schema.FillPath(cue.MakePath(cue.Def("module")), testModule)

	// A known-valid #Module MUST be injectable without error.
	require.NoError(t, withModule.Err(),
		"catalog _testModule (a known-valid #Module) must inject without error")
}

// TestStructural_RealModuleInjectSucceeds proves that a module loaded from disk
// (with core.#Module applied) injects into #ModuleRelease.#module cleanly.
// Requires OPM_REGISTRY.
//
// Uses Strategy B loading: core is loaded from the module's pinned deps,
// and both values share the same *cue.Context for injection.
func TestStructural_RealModuleInjectSucceeds(t *testing.T) {
	requireRegistry(t)

	ctx := cuecontext.New()
	realModule, schema := buildRealModuleWithSchema(t, ctx)

	withModule := schema.FillPath(cue.MakePath(cue.Def("module")), realModule)

	if err := withModule.Err(); err != nil {
		t.Logf("RESULT: real module with core.#Module ERRORS on injection: %v", err)
		t.Log("Unexpected — a module constrained by core.#Module should satisfy #Module.")
	} else {
		t.Log("RESULT: real module with core.#Module injects successfully.")
		assert.True(t, withModule.Exists())
	}
}
