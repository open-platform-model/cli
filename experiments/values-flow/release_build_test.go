package valuesflow

// ---------------------------------------------------------------------------
// Release build: buildRelease() → concrete #ModuleRelease
//
// After values are selected and validated, buildRelease() performs the full
// FillPath sequence to construct a concrete #ModuleRelease. This proves that:
//
//   - The FillPath chain (#module → metadata.name → metadata.namespace → values)
//     produces no CUE error with valid inputs
//   - The resulting #ModuleRelease satisfies cue.Concrete(true) — all fields
//     are fully concrete, none abstract
//   - The release.values field reflects exactly the selected values
//
// Note on module fixtures: testdata/values_module and testdata/inline_module use
// a free-form #components spec that does not satisfy the catalog's strict closed
// #Component schema (which requires #resources and #traits). For buildRelease
// tests the catalog's own _testModule is used instead — it is a known-valid
// #Module that the #ModuleRelease schema accepts without error.
//
// _testModule #config: { replicaCount: int & >=1, image: string }
// _testModule values:  { replicaCount: 2, image: "nginx:12" }
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReleaseBuild_ConcreteAfterBuild proves that after a successful buildRelease
// call the resulting #ModuleRelease value satisfies cue.Concrete(true) — every
// field is fully resolved with no abstract constraints remaining.
func TestReleaseBuild_ConcreteAfterBuild(t *testing.T) {
	ctx, catalogVal := loadCatalog(t)
	schema := releaseSchema(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	// _testModule #config: { replicaCount: int & >=1, image: string }
	userValues := ctx.CompileString(`{ replicaCount: 3, image: "nginx:1.28" }`)
	require.NoError(t, userValues.Err())

	result := buildRelease(schema, testModule, "my-release", "default", userValues)
	require.NoError(t, result.Err(), "buildRelease must not error with valid inputs")

	err := result.Validate(cue.Concrete(true))
	assert.NoError(t, err, "#ModuleRelease must be fully concrete after buildRelease")
}

// TestReleaseBuild_ValuesFieldReflectsSelectedValues proves that the values field
// on the built release carries the selected values — concrete and correct.
// This is the key output invariant: release.values is the final, authoritative values.
func TestReleaseBuild_ValuesFieldReflectsSelectedValues(t *testing.T) {
	ctx, catalogVal := loadCatalog(t)
	schema := releaseSchema(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	userValues := ctx.CompileString(`{ replicaCount: 3, image: "nginx:1.28" }`)
	require.NoError(t, userValues.Err())

	result := buildRelease(schema, testModule, "my-release", "default", userValues)
	require.NoError(t, result.Err())

	relValues := result.LookupPath(cue.ParsePath("values"))
	require.True(t, relValues.Exists(), "release must have a values field")

	image, err := relValues.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:1.28", image,
		"release.values.image must reflect the selected user values")

	replicaCount, err := relValues.LookupPath(cue.ParsePath("replicaCount")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicaCount,
		"release.values.replicaCount must reflect the selected user values")
}

// TestReleaseBuild_FillIsImmutable proves that buildRelease does not mutate the
// schema value. The same schema can be used across multiple independent builds.
func TestReleaseBuild_FillIsImmutable(t *testing.T) {
	ctx, catalogVal := loadCatalog(t)
	schema := releaseSchema(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	vals1 := ctx.CompileString(`{ replicaCount: 1, image: "nginx:1.0" }`)
	vals2 := ctx.CompileString(`{ replicaCount: 5, image: "nginx:2.0" }`)

	// First build
	rel1 := buildRelease(schema, testModule, "release-a", "ns-a", vals1)
	require.NoError(t, rel1.Err())

	// Second build from the same schema — must be independent
	rel2 := buildRelease(schema, testModule, "release-b", "ns-b", vals2)
	require.NoError(t, rel2.Err())

	name1, err := rel1.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	name2, err := rel2.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)

	assert.Equal(t, "release-a", name1)
	assert.Equal(t, "release-b", name2, "second build must not be affected by the first")
}
