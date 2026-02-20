package modulereleasecueeval

// ---------------------------------------------------------------------------
// Decision 5: CUE merges module labels + release standard labels
//
// #ModuleRelease.metadata.labels is defined as:
//   labels: {if #moduleMetadata.labels != _|_ {#moduleMetadata.labels}} & {
//     "module-release.opmodel.dev/name":    "\(name)"
//     "module-release.opmodel.dev/version": "\(version)"
//     "module-release.opmodel.dev/uuid":    "\(uuid)"
//   }
//
// This merges:
//   - Module-level labels (from #module.metadata.labels) — e.g., module.opmodel.dev/name
//   - Release standard labels — name, version, uuid computed from release metadata
//
// These tests prove:
//   - metadata.labels is a concrete map after full fill
//   - Module-level labels are forwarded into the release labels
//   - Release standard labels (name, version, uuid) are present
//   - Label values are correct strings (not CUE constraints)
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLabels_AreConcrete proves that metadata.labels is concrete after full fill.
func TestLabels_AreConcrete(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	result := fillRelease(schema, testModule, "my-release", "default", `{
		replicaCount: 1
		image:        "nginx:latest"
	}`)
	require.NoError(t, result.Err())

	labelsVal := result.LookupPath(cue.ParsePath("metadata.labels"))
	require.True(t, labelsVal.Exists())
	require.NoError(t, labelsVal.Err())

	err := labelsVal.Validate(cue.Concrete(true))
	assert.NoError(t, err, "metadata.labels should be fully concrete after fill")
}

// TestLabels_ReleaseStandardLabelsPresent proves that the three release-standard
// labels are present and concrete in the merged labels map.
func TestLabels_ReleaseStandardLabelsPresent(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	result := fillRelease(schema, testModule, "my-release", "default", `{
		replicaCount: 1
		image:        "nginx:latest"
	}`)
	require.NoError(t, result.Err())

	labels := extractLabels(t, result)
	t.Logf("merged labels: %v", labels)

	// Release standard labels.
	assert.Equal(t, "my-release", labels["module-release.opmodel.dev/name"],
		"release name label should be set")
	assert.NotEmpty(t, labels["module-release.opmodel.dev/version"],
		"release version label should be set (from module metadata)")
	assert.Regexp(t, `^[0-9a-f-]{36}$`, labels["module-release.opmodel.dev/uuid"],
		"release uuid label should be a UUID")
}

// TestLabels_ModuleLevelLabelsForwarded proves that module-level labels
// (from #module.metadata.labels) are forwarded into the merged release labels.
//
// _testModule.metadata.labels contains:
//
//	"module.opmodel.dev/name":    "\(name)"
//	"module.opmodel.dev/version": "\(version)"
//	"module.opmodel.dev/uuid":    "\(uuid)"
func TestLabels_ModuleLevelLabelsForwarded(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	result := fillRelease(schema, testModule, "my-release", "default", `{
		replicaCount: 1
		image:        "nginx:latest"
	}`)
	require.NoError(t, result.Err())

	labels := extractLabels(t, result)

	// Module-level labels from _testModule (forwarded by #ModuleRelease.metadata.labels merge).
	assert.Equal(t, "test-module", labels["module.opmodel.dev/name"],
		"module name label should be forwarded from module metadata")
	assert.Equal(t, "0.1.0", labels["module.opmodel.dev/version"],
		"module version label should be forwarded from module metadata")
	assert.NotEmpty(t, labels["module.opmodel.dev/uuid"],
		"module uuid label should be forwarded from module metadata")
}

// TestLabels_VersionComesFromModule proves that metadata.labels["module-release.opmodel.dev/version"]
// matches the module's version, not any hardcoded value.
func TestLabels_VersionComesFromModule(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	result := fillRelease(schema, testModule, "my-release", "staging", `{
		replicaCount: 1
		image:        "nginx:latest"
	}`)
	require.NoError(t, result.Err())

	// Get module version directly.
	moduleVersion, err := testModule.LookupPath(cue.ParsePath("metadata.version")).String()
	require.NoError(t, err)

	labels := extractLabels(t, result)
	assert.Equal(t, moduleVersion, labels["module-release.opmodel.dev/version"],
		"release version label should match the module version")
}

// extractLabels is a helper that reads metadata.labels from a filled #ModuleRelease
// and returns it as a map[string]string.
func extractLabels(t *testing.T, releaseVal cue.Value) map[string]string {
	t.Helper()
	labelsVal := releaseVal.LookupPath(cue.ParsePath("metadata.labels"))
	require.True(t, labelsVal.Exists())
	require.NoError(t, labelsVal.Err())

	labels := make(map[string]string)
	iter, err := labelsVal.Fields()
	require.NoError(t, err)
	for iter.Next() {
		v, err := iter.Value().String()
		require.NoError(t, err, "label value %q should be a string", iter.Label())
		labels[iter.Label()] = v
	}
	return labels
}
