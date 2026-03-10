package render

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/opmodel/cli/internal/config"
	internalreleasefile "github.com/opmodel/cli/internal/releasefile"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/pkg/modulerelease"
	"github.com/opmodel/cli/pkg/releaseprocess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustReleaseMetadata(name, namespace string) modulerelease.ReleaseMetadata { //nolint:unparam
	return modulerelease.ReleaseMetadata{Name: name, Namespace: namespace}
}

func makeReleaseFileFixture(t *testing.T, filename, content string) string {
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

func TestRenderModule_NilConfig(t *testing.T) {
	_, err := Release(context.Background(), ReleaseOpts{Config: nil, K8sConfig: nil})
	require.Error(t, err)
	var exitErr *oerrors.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, oerrors.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "configuration not loaded")
}

func TestShowRenderOutput_NoErrors_DefaultMode(t *testing.T) {
	result := &Result{Release: mustReleaseMetadata("test-module", "default"), Warnings: []string{}}
	ShowOutput(result, ShowOutputOpts{Verbose: false})
}

func TestShowRenderOutput_Warnings(t *testing.T) {
	result := &Result{Release: mustReleaseMetadata("test-module", "default"), Warnings: []string{"deprecated transformer used", "unused values"}}
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

func TestRenderFromReleaseFile_NilConfig(t *testing.T) {
	_, err := ReleaseFile(context.Background(), ReleaseFileOpts{ReleaseFilePath: "release.cue", Config: nil, K8sConfig: nil})
	require.Error(t, err)
	var exitErr *oerrors.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, oerrors.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "configuration not loaded")
}

func TestRenderFromReleaseFile_NilK8sConfig(t *testing.T) {
	_, err := ReleaseFile(context.Background(), ReleaseFileOpts{ReleaseFilePath: "release.cue", Config: &config.GlobalConfig{}, K8sConfig: nil})
	require.Error(t, err)
	var exitErr *oerrors.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, oerrors.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "kubernetes config not resolved")
}

func TestResolveReleaseValues_UsesInlineValues(t *testing.T) {
	ctx := cuecontext.New()
	raw := ctx.CompileString(`{values: {replicas: 2}}`)
	values, err := resolveReleaseValues(ctx, raw, "./release.cue", nil)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.True(t, values[0].Exists())
	assert.NoError(t, values[0].Validate())
}

func TestResolveReleaseValues_UsesValuesFile(t *testing.T) {
	ctx := cuecontext.New()
	dir := t.TempDir()
	valuesFile := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))
	values, err := resolveReleaseValues(ctx, ctx.CompileString(`{}`), filepath.Join(dir, "release.cue"), []string{valuesFile})
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.True(t, values[0].Exists())
}

func TestLoadModuleReleaseForRender_UsesReleaseNameOverride(t *testing.T) {
	ctx := cuecontext.New()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "release.cue"), []byte("package test\nkind: \"ModuleRelease\"\nmetadata: {name: \"from-file\", namespace: \"apps\"}\nvalues: {replicas: 2}\n"), 0o644))
	rel, values, err := loadModuleReleaseForRender(ctx, dir, nil, false, "override-name")
	require.NoError(t, err)
	assert.Equal(t, "override-name", rel.Metadata.Name)
	require.Len(t, values, 1)
	assert.True(t, values[0].Exists())
}

func TestRenderFromReleaseFile_RejectsBundleRelease(t *testing.T) {
	ctx := cuecontext.New()
	filePath := makeReleaseFileFixture(t, "bundle_release.cue", "package releases\nkind: \"BundleRelease\"\nmetadata: name: \"my-bundle\"\n")
	_, err := ReleaseFile(context.Background(), ReleaseFileOpts{ReleaseFilePath: filePath, Config: &config.GlobalConfig{CueContext: ctx, Providers: map[string]cue.Value{}}, K8sConfig: &config.ResolvedKubernetesConfig{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bundle releases are not yet supported")
}

func TestRenderFromReleaseFile_ValidValuesDoNotPanicAcrossRuntimes(t *testing.T) {
	ctx := cuecontext.New()
	dir := t.TempDir()
	releaseFile := filepath.Join(dir, "release.cue")
	valuesFile := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(releaseFile, []byte("package test\nkind: \"ModuleRelease\"\nmetadata: {name: \"demo\", namespace: \"apps\"}\n#module: {metadata: {name: \"demo\", version: \"0.1.0\", modulePath: \"example.com/demo\"}, #config: close({replicas: int})}\ncomponents: {}\n"), 0o644))
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {replicas: 2}\n"), 0o644))
	values, err := resolveReleaseValues(ctx, ctx.CompileString(`{}`), releaseFile, []string{valuesFile})
	require.NoError(t, err)
	require.Len(t, values, 1)
	fileRelease, err := internalreleasefile.GetReleaseFile(ctx, releaseFile)
	require.NoError(t, err)
	require.NotNil(t, fileRelease.Module)
	merged, cfgErr := releaseprocess.ValidateConfig(fileRelease.Module.Config, values, "module", "demo")
	require.Nil(t, cfgErr)
	assert.NotPanics(t, func() {
		filled := fileRelease.Module.RawCUE.FillPath(cue.ParsePath("values"), merged)
		require.NoError(t, filled.Err())
	})
}
