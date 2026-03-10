package cmdutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/pkg/modulerelease"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustReleaseMetadata creates a ReleaseMetadata for tests.
func mustReleaseMetadata(name, namespace string) modulerelease.ReleaseMetadata { //nolint:unparam // namespace param kept for test clarity
	return modulerelease.ReleaseMetadata{Name: name, Namespace: namespace}
}

func TestRenderModule_NilConfig(t *testing.T) {
	_, err := RenderRelease(context.Background(), RenderReleaseOpts{
		Config:    nil,
		K8sConfig: nil,
	})

	require.Error(t, err)
	var exitErr *oerrors.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, oerrors.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "configuration not loaded")
}

func TestResolveModulePath_Defaults(t *testing.T) {
	assert.Equal(t, ".", ResolveModulePath(nil))
	assert.Equal(t, ".", ResolveModulePath([]string{}))
}

func TestResolveModulePath_Arg(t *testing.T) {
	assert.Equal(t, "./my-module", ResolveModulePath([]string{"./my-module"}))
}

func TestShowRenderOutput_NoErrors_DefaultMode(t *testing.T) {
	// Create a RenderResult with no errors — ShowRenderOutput should succeed.
	result := &RenderResult{
		Release:  mustReleaseMetadata("test-module", "default"),
		Warnings: []string{},
	}

	err := ShowRenderOutput(result, ShowOutputOpts{Verbose: false})

	// Should not return an error.
	assert.NoError(t, err)
}

func TestShowRenderOutput_Warnings(t *testing.T) {
	// Create a RenderResult with warnings — should succeed (warnings are non-fatal).
	result := &RenderResult{
		Release:  mustReleaseMetadata("test-module", "default"),
		Warnings: []string{"deprecated transformer used", "unused values"},
	}

	err := ShowRenderOutput(result, ShowOutputOpts{})

	// Should not return an error.
	assert.NoError(t, err)
}

func TestRenderResult_HasWarnings(t *testing.T) {
	r := &RenderResult{}
	assert.False(t, r.HasWarnings())

	r.Warnings = []string{"a warning"}
	assert.True(t, r.HasWarnings())
}

func TestRenderResult_ResourceCount(t *testing.T) {
	r := &RenderResult{}
	assert.Equal(t, 0, r.ResourceCount())
}

func TestRenderFromReleaseFile_NilConfig(t *testing.T) {
	_, err := RenderFromReleaseFile(context.Background(), RenderFromReleaseFileOpts{
		ReleaseFilePath: "release.cue",
		Config:          nil,
		K8sConfig:       nil,
	})
	require.Error(t, err)
	var exitErr *oerrors.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, oerrors.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "configuration not loaded")
}

func TestRenderFromReleaseFile_NilK8sConfig(t *testing.T) {
	// Config set but K8sConfig nil — should fail with kubernetes config error.
	_, err := RenderFromReleaseFile(context.Background(), RenderFromReleaseFileOpts{
		ReleaseFilePath: "release.cue",
		Config:          &config.GlobalConfig{},
		K8sConfig:       nil,
	})
	require.Error(t, err)
	var exitErr *oerrors.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, oerrors.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "kubernetes config not resolved")
}

func TestResolveReleaseValues_UsesInlineValues(t *testing.T) {
	ctx := cuecontext.New()
	raw := ctx.CompileString(`{
		values: {
			replicas: 2
		}
	}`)

	values, err := resolveReleaseValues(ctx, raw, "./release.cue", nil)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.True(t, values[0].Exists())
	assert.NoError(t, values[0].Validate())
}

func TestLoadModuleReleaseForRender_UsesReleaseNameOverride(t *testing.T) {
	ctx := cuecontext.New()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "release.cue"), []byte(`package test

kind: "ModuleRelease"
metadata: {
	name: "from-file"
	namespace: "apps"
}
values: {
	replicas: 2
}
`), 0o644))

	rel, values, err := loadModuleReleaseForRender(ctx, dir, nil, false, "override-name")
	require.NoError(t, err)
	assert.Equal(t, "override-name", rel.Metadata.Name)
	require.Len(t, values, 1)
	assert.True(t, values[0].Exists())
}
