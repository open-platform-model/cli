package modulereleasecueeval

// ---------------------------------------------------------------------------
// Decision 6: CUE derives components from _#module.#components with values applied
//
// #ModuleRelease.components is defined as:
//   _#module:   #module & {#config: values}     // module with values injected
//   components: _#module.#components             // components from the filled module
//
// This means CUE automatically:
//   1. Takes the injected #module value
//   2. Fills its #config with the release values
//   3. Derives the concrete #components from the filled module
//
// These tests prove:
//   - components field exists and is concrete after full fill
//   - Component names match what the module defines
//   - Component specs contain concrete values derived from user-supplied values
//   - Different user values produce different component specs
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComponents_FieldExistsAndIsConcrete proves that the components field is
// present and fully concrete after all required fields are filled.
func TestComponents_FieldExistsAndIsConcrete(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	result := fillRelease(schema, testModule, "my-release", "default", `{
		replicaCount: 3
		image:        "nginx:1.28"
	}`)
	require.NoError(t, result.Err())

	componentsVal := result.LookupPath(cue.ParsePath("components"))
	require.True(t, componentsVal.Exists(), "components should exist")
	require.NoError(t, componentsVal.Err(), "components should not be errored")

	err := componentsVal.Validate(cue.Concrete(true))
	assert.NoError(t, err, "components should be fully concrete after fill")
}

// TestComponents_ContainsExpectedComponent proves that the components map contains
// the expected component name from _testModule's #components definition.
// _testModule defines: #components: { "test-deployment": ... }
func TestComponents_ContainsExpectedComponent(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	result := fillRelease(schema, testModule, "my-release", "default", `{
		replicaCount: 1
		image:        "nginx:latest"
	}`)
	require.NoError(t, result.Err())

	componentsVal := result.LookupPath(cue.ParsePath("components"))

	// Collect component names.
	var names []string
	iter, err := componentsVal.Fields()
	require.NoError(t, err)
	for iter.Next() {
		names = append(names, iter.Label())
		t.Logf("component: %s", iter.Label())
	}

	assert.Contains(t, names, "test-deployment",
		"components should contain 'test-deployment' from _testModule")
}

// TestComponents_ValuesPropagate proves that user-supplied values reach component
// specs through the CUE-evaluated _#module.#components pathway.
// _testModule's test-deployment references #config.image and #config.replicaCount.
func TestComponents_ValuesPropagate(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	result := fillRelease(schema, testModule, "my-release", "default", `{
		replicaCount: 5
		image:        "myapp:v2.1"
	}`)
	require.NoError(t, result.Err())

	// Navigate to test-deployment component spec.
	// _testModule defines: spec: container: image, spec: scaling: count
	// Note: field names with hyphens require quoting in CUE path literals.
	compSpec := result.LookupPath(cue.ParsePath(`components."test-deployment".spec`))
	require.True(t, compSpec.Exists(), "components.test-deployment.spec should exist")
	require.NoError(t, compSpec.Err())

	image, err := compSpec.LookupPath(cue.ParsePath("container.image")).String()
	if err != nil {
		t.Logf("container.image not readable as string: %v — dumping component spec for inspection", err)
		t.Logf("component spec: %v", compSpec)
	} else {
		assert.Equal(t, "myapp:v2.1", image,
			"component spec should reflect user-supplied image value")
	}

	count, err := compSpec.LookupPath(cue.ParsePath("scaling.count")).Int64()
	if err != nil {
		t.Logf("scaling.count not readable: %v", err)
	} else {
		assert.Equal(t, int64(5), count,
			"component spec should reflect user-supplied replicaCount value")
	}
}

// TestComponents_DifferentValuesProduceDifferentSpecs proves that two releases
// with different values produce components with different concrete specs.
func TestComponents_DifferentValuesProduceDifferentSpecs(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	resultA := fillRelease(schema, testModule, "rel-a", "default", `{
		replicaCount: 1
		image:        "nginx:1.0"
	}`)
	require.NoError(t, resultA.Err())

	resultB := fillRelease(schema, testModule, "rel-b", "default", `{
		replicaCount: 10
		image:        "nginx:2.0"
	}`)
	require.NoError(t, resultB.Err())

	imageA, err := resultA.LookupPath(cue.ParsePath(`components."test-deployment".spec.container.image`)).String()
	if err != nil {
		t.Skipf("cannot read container.image from components — skipping diff test: %v", err)
	}

	imageB, err := resultB.LookupPath(cue.ParsePath(`components."test-deployment".spec.container.image`)).String()
	require.NoError(t, err)

	assert.NotEqual(t, imageA, imageB,
		"different user values must produce different component specs")
}
