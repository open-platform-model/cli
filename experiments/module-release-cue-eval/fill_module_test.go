package modulereleasecueeval

// ---------------------------------------------------------------------------
// Decision 2: FillPath(cue.Def("module"), moduleVal) compiles without error
//
// The injection point for Approach C is:
//   releaseSchema.FillPath(cue.MakePath(cue.Def("module")), moduleVal)
//
// This decision answers:
//   - Does FillPath on a definition field (#module!) accept a cue.Value?
//   - Does the resulting value have a lower error rate than the pre-fill schema?
//   - Does filling #module alone (without metadata/values) not produce a panic?
//   - What happens when metadata.name and metadata.namespace are also filled?
//   - Is the fill result still extensible (can further FillPath calls succeed)?
//
// The "module" input for these tests is _testModule from the catalog — a known-
// valid #Module value. Decision 3 tests the fake_module (structurally compatible
// but not constrained by #Module).
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFill_ModuleDefFieldAcceptsValue proves that FillPath with cue.Def("module")
// does not panic and produces a non-errored cue.Value when given a valid #Module.
func TestFill_ModuleDefFieldAcceptsValue(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	// The core injection call — must not panic.
	withModule := schema.FillPath(cue.MakePath(cue.Def("module")), testModule)

	assert.NoError(t, withModule.Err(), "FillPath(#module) with a valid #Module should not error")
	assert.True(t, withModule.Exists(), "result of FillPath should exist")
}

// TestFill_ModuleAloneDoesNotMakeReleaseConcrete proves that filling only #module
// (without metadata.name, metadata.namespace, or values) leaves the release
// non-concrete. This is expected: the remaining required fields must still be filled.
func TestFill_ModuleAloneDoesNotMakeReleaseConcrete(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	withModule := schema.FillPath(cue.MakePath(cue.Def("module")), testModule)
	require.NoError(t, withModule.Err())

	err := withModule.Validate(cue.Concrete(true))
	assert.Error(t, err, "release with only #module filled should still be non-concrete")
}

// TestFill_MetadataNameAndNamespaceReduceErrors proves that filling metadata.name
// and metadata.namespace after #module moves the value closer to concrete.
func TestFill_MetadataNameAndNamespaceReduceErrors(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)
	ctx := schema.Context()

	withModule := schema.FillPath(cue.MakePath(cue.Def("module")), testModule)
	withMeta := withModule.
		FillPath(cue.ParsePath("metadata.name"), ctx.CompileString(`"test-rel"`)).
		FillPath(cue.ParsePath("metadata.namespace"), ctx.CompileString(`"default"`))

	assert.NoError(t, withMeta.Err(), "filling metadata should not error")

	// After name + namespace, metadata.version should be derivable from _testModule.metadata.version.
	version, err := withMeta.LookupPath(cue.ParsePath("metadata.version")).String()
	require.NoError(t, err, "metadata.version should be concrete after name+namespace+module")
	assert.Equal(t, "0.1.0", version, "version should come from _testModule metadata")
}

// TestFill_FullSequenceDoesNotError proves that the complete FillPath sequence —
// module + name + namespace + values — produces a non-errored value.
// This is the pre-condition for all subsequent Decision tests.
func TestFill_FullSequenceDoesNotError(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	// _testModule expects: replicaCount int & >=1, image string
	result := fillRelease(schema, testModule, "my-release", "production", `{
		replicaCount: 3
		image:        "nginx:1.28"
	}`)

	assert.NoError(t, result.Err(), "full FillPath sequence should not error")
}

// TestFill_FullSequenceIsConcrete proves that after all required fields are filled,
// the resulting #ModuleRelease value is fully concrete.
func TestFill_FullSequenceIsConcrete(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	result := fillRelease(schema, testModule, "my-release", "production", `{
		replicaCount: 3
		image:        "nginx:1.28"
	}`)
	require.NoError(t, result.Err())

	err := result.Validate(cue.Concrete(true))
	assert.NoError(t, err, "fully-filled #ModuleRelease should be concrete")
}

// TestFill_FillIsImmutable proves that FillPath does not mutate the schema value.
// The schema must remain reusable across multiple releases.
func TestFill_FillIsImmutable(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	// First release.
	_ = fillRelease(schema, testModule, "release-a", "ns-a", `{replicaCount: 1, image: "nginx:1.0"}`)

	// Schema must be unaffected — a second fill produces an independent value.
	release2 := fillRelease(schema, testModule, "release-b", "ns-b", `{replicaCount: 5, image: "nginx:2.0"}`)
	require.NoError(t, release2.Err())

	name, err := release2.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "release-b", name, "second release should have its own metadata.name")
}
