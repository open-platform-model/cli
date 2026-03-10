package releasefile

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReleaseFile_ModuleReleasePartial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "release.cue")
	require.NoError(t, os.WriteFile(path, []byte(`package test

apiVersion: "opmodel.dev/core/v1alpha1"
kind: "ModuleRelease"
metadata: {
	name: "demo"
	namespace: "apps"
}
values: {
	replicas: 2
}
`), 0o644))

	rel, err := GetReleaseFile(cuecontext.New(), path)
	require.NoError(t, err)
	require.NotNil(t, rel.Module)
	assert.Equal(t, KindModuleRelease, rel.Kind)
	assert.Equal(t, "demo", rel.Module.Metadata.Name)
	assert.Equal(t, "apps", rel.Module.Metadata.Namespace)
	assert.True(t, rel.Module.RawCUE.Exists())
	assert.False(t, rel.Module.Values.Exists())
	assert.False(t, rel.Module.DataComponents.Exists())
}

func TestGetReleaseFile_BundleReleasePartial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "release.cue")
	require.NoError(t, os.WriteFile(path, []byte(`package test

apiVersion: "opmodel.dev/core/v1alpha1"
kind: "BundleRelease"
metadata: {
	name: "stack"
}
values: {
	replicas: 2
}
`), 0o644))

	rel, err := GetReleaseFile(cuecontext.New(), path)
	require.NoError(t, err)
	require.NotNil(t, rel.Bundle)
	assert.Equal(t, KindBundleRelease, rel.Kind)
	assert.Equal(t, "stack", rel.Bundle.Metadata.Name)
	assert.True(t, rel.Bundle.RawCUE.Exists())
	assert.Empty(t, rel.Bundle.Releases)
	assert.False(t, rel.Bundle.Values.Exists())
}

func TestGetReleaseFile_UnknownKind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "release.cue")
	require.NoError(t, os.WriteFile(path, []byte(`package test
kind: "MysteryRelease"
`), 0o644))

	_, err := GetReleaseFile(cuecontext.New(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown release kind")
}

func TestGetReleaseFile_FailsWhenModuleMetadataNotConcrete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "release.cue")
	require.NoError(t, os.WriteFile(path, []byte(`package test

kind: "ModuleRelease"
metadata: {
	name: string
	namespace: "apps"
}
`), 0o644))

	_, err := GetReleaseFile(cuecontext.New(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata must be concrete")
}

func TestGetReleaseFile_FailsWhenBundleMetadataNotConcrete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "release.cue")
	require.NoError(t, os.WriteFile(path, []byte(`package test

kind: "BundleRelease"
metadata: {
	name: string
}
`), 0o644))

	_, err := GetReleaseFile(cuecontext.New(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata must be concrete")
}
