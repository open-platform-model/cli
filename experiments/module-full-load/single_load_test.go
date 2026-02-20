package modulefullload

// ---------------------------------------------------------------------------
// Decision 1: Single load — same inst for AST inspection and BuildInstance
//
// The design claims load.Instances() is called once, inst.Files is used for
// AST inspection (extracting name, pkgName, defaultNamespace), and then
// cueCtx.BuildInstance(inst) is called on the same inst to get the base
// cue.Value — with no second load.Instances() call.
//
// These tests prove:
//   - inst.Files and inst.PkgName are populated before BuildInstance (AST phase)
//   - BuildInstance(inst) on the same inst succeeds and returns a valid value
//   - The value matches what AST inspection found
//   - BuildInstance is idempotent on the same inst
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSingleLoad_ASTAndBuildFromSameInst proves that AST inspection (reading
// inst.PkgName and walking inst.Files) and CUE evaluation (BuildInstance) can
// both be performed from a single load.Instances() call, with no second load.
func TestSingleLoad_ASTAndBuildFromSameInst(t *testing.T) {
	modulePath := testModulePath(t)

	// --- Phase 1: load once ---
	instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)
	inst := instances[0]

	// --- AST inspection (no BuildInstance yet) ---
	// pkgName is available directly on the instance.
	assert.Equal(t, "testmodule", inst.PkgName, "pkgName should be readable before BuildInstance")

	// AST files are populated — walk to find metadata.name as a string literal.
	name := extractNameFromAST(inst.Files)
	assert.Equal(t, "test-module", name, "metadata.name should be extractable from AST")

	// --- CUE evaluation: same inst, no second load.Instances() call ---
	ctx := cuecontext.New()
	val := ctx.BuildInstance(inst)
	require.NoError(t, val.Err(), "BuildInstance on same inst should not error")

	// The evaluated value agrees with what AST inspection found.
	evalName, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, name, evalName, "AST-extracted name should match evaluated name")
}

// TestSingleLoad_BaseValueNotError proves that BuildInstance on the test module
// returns a non-error cue.Value. This is the precondition for everything
// else in the design: if BuildInstance fails, Load() fails immediately.
func TestSingleLoad_BaseValueNotError(t *testing.T) {
	modulePath := testModulePath(t)
	ctx := cuecontext.New()

	instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	val := ctx.BuildInstance(instances[0])
	assert.NoError(t, val.Err(), "base value should not be errored")
	assert.True(t, val.Exists(), "base value should exist")
}

// TestSingleLoad_InstIsReusable proves that BuildInstance can be called
// multiple times on the same *build.Instance. The design assumes the instance
// from load.Instances() is a stable, reusable input — if it were mutated to
// an unusable state after the first BuildInstance call, the design would fail.
//
// Note: tests in experiments/ast-pipeline/cross_context_test.go proved that
// concurrent BuildInstance on a shared inst has data races. This test only
// proves sequential reuse, which is the single-threaded Load() case.
func TestSingleLoad_InstIsReusable(t *testing.T) {
	modulePath := testModulePath(t)
	ctx := cuecontext.New()

	instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)
	inst := instances[0]

	// First build
	val1 := ctx.BuildInstance(inst)
	require.NoError(t, val1.Err(), "first BuildInstance should succeed")

	// Second build on same inst — must also succeed for the design to be safe.
	// (In practice Load() only calls it once; this tests the invariant.)
	val2 := ctx.BuildInstance(inst)
	require.NoError(t, val2.Err(), "second BuildInstance on same inst should succeed")

	// Both values must produce the same metadata.name.
	name1, err := val1.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	name2, err := val2.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, name1, name2, "both BuildInstance calls should produce equivalent values")
}

// TestSingleLoad_ASTFilesPopulatedBeforeBuild verifies that inst.Files contains
// the parsed *ast.File entries immediately after load.Instances(), before any
// BuildInstance call. This confirms AST inspection is a pure read operation
// that does not depend on evaluation.
func TestSingleLoad_ASTFilesPopulatedBeforeBuild(t *testing.T) {
	modulePath := testModulePath(t)

	instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)
	inst := instances[0]

	require.NotEmpty(t, inst.Files, "inst.Files should be populated before BuildInstance")

	// Each file must be a valid *ast.File with a filename.
	for _, f := range inst.Files {
		assert.NotNil(t, f, "each file in inst.Files should be non-nil")
		assert.NotEmpty(t, f.Filename, "each file should have a filename")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// extractNameFromAST walks ast.Files to find the metadata.name string literal.
// This replicates the logic module.Load() currently uses (internal/build/module/inspector.go).
func extractNameFromAST(files []*ast.File) string {
	for _, f := range files {
		for _, decl := range f.Decls {
			field, ok := decl.(*ast.Field)
			if !ok {
				continue
			}
			ident, ok := field.Label.(*ast.Ident)
			if !ok || ident.Name != "metadata" {
				continue
			}
			structLit, ok := field.Value.(*ast.StructLit)
			if !ok {
				continue
			}
			for _, elt := range structLit.Elts {
				inner, ok := elt.(*ast.Field)
				if !ok {
					continue
				}
				innerIdent, ok := inner.Label.(*ast.Ident)
				if !ok || innerIdent.Name != "name" {
					continue
				}
				if lit, ok := inner.Value.(*ast.BasicLit); ok {
					// Strip surrounding quotes from the string literal.
					s := lit.Value
					if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
						return s[1 : len(s)-1]
					}
				}
			}
		}
	}
	return ""
}
