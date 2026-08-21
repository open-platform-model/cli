package render

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/pkg/module"
)

func mustInstanceMetadata(name, namespace string) module.InstanceMetadata {
	return module.InstanceMetadata{Name: name, Namespace: namespace}
}

func TestShowRenderOutput_NoErrors_DefaultMode(t *testing.T) {
	result := &Result{Instance: mustInstanceMetadata("demo", "default")}
	assert.NotPanics(t, func() { ShowOutput(result, ShowOutputOpts{}) })
}

func TestShowRenderOutput_Warnings(t *testing.T) {
	result := &Result{Instance: mustInstanceMetadata("demo", "default"), Warnings: []string{"w1"}}
	assert.NotPanics(t, func() { ShowOutput(result, ShowOutputOpts{Verbose: true}) })
}

func TestRenderResult_HasWarnings(t *testing.T) {
	assert.False(t, (&Result{}).HasWarnings())
	assert.True(t, (&Result{Warnings: []string{"x"}}).HasWarnings())
}

func TestRenderResult_ResourceCount(t *testing.T) {
	assert.Equal(t, 0, (&Result{}).ResourceCount())
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
	// The path guard fires before platform resolution, so no registry or
	// platform file is needed.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte("package test\n"), 0o644))

	_, err := FromInstanceFile(context.Background(), InstanceFileOpts{
		InstanceFilePath: dir,
		Config:           &config.GlobalConfig{},
		K8sConfig:        &config.ResolvedKubernetesConfig{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module package, not an instance")
}

func TestNewResult_CarriesResolvedPlatform(t *testing.T) {
	// The apply workflow seeds the cluster Platform from Result.PlatformSpec
	// (0006 D12): the assembly must carry the resolved input verbatim, or
	// the seeded document degrades to the zero value.
	enable := true
	input := synth.PlatformInput{
		Name: "local-default",
		Type: "kubernetes",
		Subscriptions: map[string]synth.SubscriptionSpec{
			"opmodel.dev/catalogs/opm@v2": {Enable: &enable, Version: "2.0.0-alpha.3"},
		},
	}
	env := &renderEnv{
		resolution: platform.Resolution{Source: platform.SourceLocalDefault, Location: "~/.opm/platform.cue", Warning: "cluster Platform not found"},
		input:      input,
	}

	result := newResult(env, &kernel.CompileResult{Warnings: []string{"w1"}}, "digest", map[string]any{"k": "v"}, true)

	assert.Equal(t, env.resolution, result.Platform)
	assert.Equal(t, input, result.PlatformSpec)
	assert.NotEmpty(t, result.PlatformSpec.Type, "seeded spec must never carry an empty type")
	assert.Equal(t, []string{"w1"}, result.Warnings)
	assert.Equal(t, "digest", result.RenderDigest)
	assert.Equal(t, map[string]any{"k": "v"}, result.Values)
	assert.True(t, result.SourceLocal)
}

func TestUnifyValuesFiles_Empty(t *testing.T) {
	v, err := unifyValuesFiles(cuecontext.New(), nil)
	require.NoError(t, err)
	assert.False(t, v.Exists(), "zero value signals no files given")
}

func TestUnifyValuesFiles_SingleFile(t *testing.T) {
	ctx := cuecontext.New()
	dir := t.TempDir()
	valuesFile := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))

	v, err := unifyValuesFiles(ctx, []string{valuesFile})
	require.NoError(t, err)
	require.True(t, v.Exists())
	assert.NoError(t, v.Validate())
}

func TestUnifyValuesFiles_MultipleFilesUnify(t *testing.T) {
	ctx := cuecontext.New()
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.cue")
	f2 := filepath.Join(dir, "b.cue")
	require.NoError(t, os.WriteFile(f1, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))
	require.NoError(t, os.WriteFile(f2, []byte("package test\nvalues: {image: \"nginx\"}\n"), 0o644))

	v, err := unifyValuesFiles(ctx, []string{f1, f2})
	require.NoError(t, err)
	require.True(t, v.Exists())
	assert.NoError(t, v.Validate())
}

func TestUnifyValuesFiles_ConflictFails(t *testing.T) {
	ctx := cuecontext.New()
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.cue")
	f2 := filepath.Join(dir, "b.cue")
	require.NoError(t, os.WriteFile(f1, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))
	require.NoError(t, os.WriteFile(f2, []byte("package test\nvalues: {replicas: 4}\n"), 0o644))

	_, err := unifyValuesFiles(ctx, []string{f1, f2})
	require.Error(t, err)
}

func writeD19File(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// The D19 warning fires exactly when the effective module context carries a
// local replacement; both render entries share warnLocalReplacement, so the
// warning string cannot diverge between them.
func TestWarnLocalReplacement(t *testing.T) {
	assert.True(t, warnLocalReplacement(true))
	assert.False(t, warnLocalReplacement(false))
}

func TestModuleContextHasLocalReplacement_PresentWithReplaceWith(t *testing.T) {
	root := t.TempDir()
	writeD19File(t, filepath.Join(root, "cue.mod", "module.cue"), `module: "example.com/main@v0"`)
	writeD19File(t, filepath.Join(root, "cue.mod", "local-module.cue"),
		`deps: "opmodel.dev/modules/podinfo@v0": replaceWith: "../podinfo"`)

	assert.True(t, moduleContextHasLocalReplacement(root))
}

func TestModuleContextHasLocalReplacement_AbsentStaysSilent(t *testing.T) {
	root := t.TempDir()
	writeD19File(t, filepath.Join(root, "cue.mod", "module.cue"), `module: "example.com/main@v0"`)

	assert.False(t, moduleContextHasLocalReplacement(root))
}
