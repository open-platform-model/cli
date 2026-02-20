package modulefullload

// ---------------------------------------------------------------------------
// Decision 9: Module metadata is read from CUE evaluation, not recomputed
//
// The design moves metadata extraction (uuid, fqn, version, labels,
// defaultNamespace) from separate Go computation to direct reads from the
// evaluated cue.Value. After BuildInstance(), all of these must be accessible
// as concrete values via LookupPath — otherwise the design cannot work.
//
// These tests prove:
//   - metadata.name, version, fqn, uuid, defaultNamespace are concrete strings
//   - metadata.labels is iterable as a map[string]string
//   - #config and values are extractable as sub-values (for schema extraction)
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetadata_NameReadable proves metadata.name is a concrete string in the
// evaluated value.
func TestMetadata_NameReadable(t *testing.T) {
	_, val := buildBaseValue(t)
	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "test-module", name)
}

// TestMetadata_VersionReadable proves metadata.version is readable.
func TestMetadata_VersionReadable(t *testing.T) {
	_, val := buildBaseValue(t)
	version, err := val.LookupPath(cue.ParsePath("metadata.version")).String()
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", version)
}

// TestMetadata_FQNReadable proves metadata.fqn is readable.
// In the design, this is the primary input to ComputeReleaseUUID().
func TestMetadata_FQNReadable(t *testing.T) {
	_, val := buildBaseValue(t)
	fqn, err := val.LookupPath(cue.ParsePath("metadata.fqn")).String()
	require.NoError(t, err)
	assert.Equal(t, "example.com/test-module@v0#TestModule", fqn)
}

// TestMetadata_UUIDReadable proves metadata.uuid is readable as a concrete
// string from the evaluated value. In production this is CUE-computed via
// uid.SHA1; here it is a static string. What matters is the read path works.
func TestMetadata_UUIDReadable(t *testing.T) {
	_, val := buildBaseValue(t)
	uuid, err := val.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)
	assert.Equal(t, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", uuid)
}

// TestMetadata_DefaultNamespaceReadable proves metadata.defaultNamespace is
// readable. This drives the --namespace default in the build phase.
func TestMetadata_DefaultNamespaceReadable(t *testing.T) {
	_, val := buildBaseValue(t)
	ns, err := val.LookupPath(cue.ParsePath("metadata.defaultNamespace")).String()
	require.NoError(t, err)
	assert.Equal(t, "default", ns)
}

// TestMetadata_LabelsIterable proves metadata.labels is iterable as a
// map[string]string from the evaluated value. The design reads these labels
// to populate Module.Metadata.Labels for use in release labeling.
func TestMetadata_LabelsIterable(t *testing.T) {
	_, val := buildBaseValue(t)
	labelsVal := val.LookupPath(cue.ParsePath("metadata.labels"))
	require.True(t, labelsVal.Exists(), "metadata.labels should exist")
	require.NoError(t, labelsVal.Err())

	labels := map[string]string{}
	iter, err := labelsVal.Fields()
	require.NoError(t, err)
	for iter.Next() {
		v, err := iter.Value().String()
		require.NoError(t, err, "each label value should be a concrete string")
		labels[iter.Label()] = v
	}

	assert.Equal(t, "test-module", labels["module.opmodel.dev/name"])
	assert.Equal(t, "1.0.0", labels["module.opmodel.dev/version"])
}

// TestMetadata_ConfigExtractable proves that #config is accessible from the
// evaluated base value. module.Load() will store this as Module.Config for
// values validation in the build phase.
//
// With defaults on all #config fields (e.g., image: string | *"nginx:latest"),
// CUE resolves the disjunction to its default at evaluation time — so #config
// fields ARE concrete at schema level. The test verifies the default values.
func TestMetadata_ConfigExtractable(t *testing.T) {
	_, val := buildBaseValue(t)
	configVal := val.LookupPath(cue.ParsePath("#config"))
	assert.True(t, configVal.Exists(), "#config should exist in evaluated value")
	assert.NoError(t, configVal.Err(), "#config should not be errored")

	// All #config fields have defaults — they resolve to concrete values at schema level.
	imageField := configVal.LookupPath(cue.ParsePath("image"))
	assert.True(t, imageField.Exists(), "#config.image should exist")
	image, err := imageField.String()
	require.NoError(t, err, "#config.image should be readable as a concrete string (schema default)")
	assert.Equal(t, "nginx:latest", image, "#config.image default should be nginx:latest")

	replicas, err := configVal.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err, "#config.replicas should be readable as a concrete int (schema default)")
	assert.Equal(t, int64(1), replicas, "#config.replicas default should be 1")

	port, err := configVal.LookupPath(cue.ParsePath("port")).Int64()
	require.NoError(t, err, "#config.port should be readable as a concrete int (schema default)")
	assert.Equal(t, int64(8080), port, "#config.port default should be 8080")

	debug, err := configVal.LookupPath(cue.ParsePath("debug")).Bool()
	require.NoError(t, err, "#config.debug should be readable as a concrete bool (schema default)")
	assert.Equal(t, false, debug, "#config.debug default should be false")
}

// TestMetadata_DefaultValuesExtractable proves that the defaultValues field is
// accessible from the evaluated base value. module.Load() will store this as
// Module.Values for display and default-injection.
func TestMetadata_DefaultValuesExtractable(t *testing.T) {
	_, val := buildBaseValue(t)
	valuesVal := val.LookupPath(cue.ParsePath("defaultValues"))
	assert.True(t, valuesVal.Exists(), "defaultValues should exist in evaluated value")
	assert.NoError(t, valuesVal.Err())

	// defaultValues.replicas is a concrete int.
	replicas, err := valuesVal.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), replicas)
}
