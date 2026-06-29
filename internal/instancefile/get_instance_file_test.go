package instancefile

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInstanceFile_ModuleInstancePartial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "instance.cue")
	require.NoError(t, os.WriteFile(path, []byte(`package test

apiVersion: "opmodel.dev/core/v1alpha1"
kind: "ModuleInstance"
metadata: {
	name: "demo"
	namespace: "apps"
}
values: {
	replicas: 2
}
`), 0o644))

	rel, err := GetInstanceFile(cuecontext.New(), path)
	require.NoError(t, err)
	require.NotNil(t, rel.Module)
	assert.Equal(t, KindModuleInstance, rel.Kind)
	assert.Equal(t, "demo", rel.Module.Metadata.Name)
	assert.Equal(t, "apps", rel.Module.Metadata.Namespace)
	assert.True(t, rel.Module.Spec.Exists())
}

func TestGetInstanceFile_UnknownKind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "instance.cue")
	require.NoError(t, os.WriteFile(path, []byte(`package test
kind: "MysteryRelease"
`), 0o644))

	_, err := GetInstanceFile(cuecontext.New(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown instance kind")
}

func TestGetInstanceFile_FailsWhenModuleMetadataNotConcrete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "instance.cue")
	require.NoError(t, os.WriteFile(path, []byte(`package test

kind: "ModuleInstance"
metadata: {
	name: string
	namespace: "apps"
}
`), 0o644))

	_, err := GetInstanceFile(cuecontext.New(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata must be concrete")
}
