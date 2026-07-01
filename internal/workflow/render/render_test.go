package render

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/open-platform-model/cli/internal/config"
	internalinstancefile "github.com/open-platform-model/cli/internal/instancefile"
	"github.com/open-platform-model/cli/pkg/module"
	"github.com/open-platform-model/cli/pkg/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustInstanceMetadata(name, namespace string) module.InstanceMetadata {
	return module.InstanceMetadata{Name: name, Namespace: namespace}
}

func makeInstanceFileFixture(t *testing.T, filename, content string) string {
	t.Helper()
	dir := t.TempDir()
	modDir := filepath.Join(dir, "cue.mod")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(`module: "test.example.com/releases@v0"
language: version: "v0.15.0"
`), 0o644))

	filePath := filepath.Join(dir, filename)
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
	return filePath
}

func TestShowRenderOutput_NoErrors_DefaultMode(t *testing.T) {
	result := &Result{Instance: mustInstanceMetadata("test-module", "default"), Warnings: []string{}}
	ShowOutput(result, ShowOutputOpts{Verbose: false})
}

func TestShowRenderOutput_Warnings(t *testing.T) {
	result := &Result{Instance: mustInstanceMetadata("test-module", "default"), Warnings: []string{"deprecated transformer used", "unused values"}}
	ShowOutput(result, ShowOutputOpts{})
}

func TestRenderResult_HasWarnings(t *testing.T) {
	r := &Result{}
	assert.False(t, r.HasWarnings())
	r.Warnings = []string{"a warning"}
	assert.True(t, r.HasWarnings())
}

func TestRenderResult_ResourceCount(t *testing.T) {
	r := &Result{}
	assert.Equal(t, 0, r.ResourceCount())
}

func TestRenderFromInstanceFile_NilConfig(t *testing.T) {
	_, err := FromInstanceFile(context.Background(), InstanceFileOpts{InstanceFilePath: "instance.cue", Config: nil, K8sConfig: nil})
	require.Error(t, err)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "configuration not loaded")
}

func TestRenderFromInstanceFile_NilK8sConfig(t *testing.T) {
	_, err := FromInstanceFile(context.Background(), InstanceFileOpts{InstanceFilePath: "instance.cue", Config: &config.GlobalConfig{}, K8sConfig: nil})
	require.Error(t, err)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "kubernetes config not resolved")
}

func TestRenderFromInstanceFile_RejectsModulePackagePath(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte("package test\n"), 0o644))

	_, err := FromInstanceFile(context.Background(), InstanceFileOpts{
		InstanceFilePath: dir,
		Config:           &config.GlobalConfig{CueContext: cuecontext.New()},
		K8sConfig:        &config.ResolvedKubernetesConfig{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module package, not an instance")
}

func TestResolveInstanceValues_UsesInlineValues(t *testing.T) {
	ctx := cuecontext.New()
	raw := ctx.CompileString(`{values: {replicas: 2}}`)
	values, err := resolveInstanceValues(ctx, raw, "./instance.cue", nil)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.True(t, values[0].Exists())
	assert.NoError(t, values[0].Validate())
}

func TestResolveInstanceValues_UsesValuesFile(t *testing.T) {
	ctx := cuecontext.New()
	dir := t.TempDir()
	valuesFile := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))
	values, err := resolveInstanceValues(ctx, ctx.CompileString(`{}`), filepath.Join(dir, "instance.cue"), []string{valuesFile})
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.True(t, values[0].Exists())
}

// A bundle file is no longer specially handled — the bundle path was removed in
// 0002 X2 (D15). It now fails upstream at kind detection as an unknown kind.
func TestRenderFromInstanceFile_RejectsBundleRelease(t *testing.T) {
	ctx := cuecontext.New()
	filePath := makeInstanceFileFixture(t, "bundle_release.cue", "package releases\nkind: \"BundleRelease\"\nmetadata: name: \"my-bundle\"\n")
	_, err := FromInstanceFile(context.Background(), InstanceFileOpts{InstanceFilePath: filePath, Config: &config.GlobalConfig{CueContext: ctx, Providers: map[string]cue.Value{}}, K8sConfig: &config.ResolvedKubernetesConfig{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown instance kind")
}

func TestRenderFromInstanceFile_ValidValuesDoNotPanicAcrossRuntimes(t *testing.T) {
	ctx := cuecontext.New()
	dir := t.TempDir()
	instanceFile := filepath.Join(dir, "instance.cue")
	valuesFile := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(instanceFile, []byte("package test\nkind: \"ModuleInstance\"\nmetadata: {name: \"demo\", namespace: \"apps\"}\n#module: {metadata: {name: \"demo\", version: \"0.1.0\", modulePath: \"example.com/demo\"}, #config: close({replicas: int})}\ncomponents: {}\n"), 0o644))
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {replicas: 2}\n"), 0o644))
	values, err := resolveInstanceValues(ctx, ctx.CompileString(`{}`), instanceFile, []string{valuesFile})
	require.NoError(t, err)
	require.Len(t, values, 1)
	fileInstance, err := internalinstancefile.GetInstanceFile(ctx, instanceFile)
	require.NoError(t, err)
	require.NotNil(t, fileInstance.Module)
	merged, cfgErr := validate.Config(fileInstance.Module.Module.Config, values, "module", "demo")
	require.Nil(t, cfgErr)
	assert.NotPanics(t, func() {
		filled := fileInstance.Module.Spec.FillPath(cue.ParsePath("values"), merged)
		require.NoError(t, filled.Err())
	})
}
