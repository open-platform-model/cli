package cmdutil

import (
	"context"
	"errors"
	"testing"

	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/pkg/modulerelease"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustReleaseMetadata creates a ReleaseMetadata for tests.
func mustReleaseMetadata(name, namespace string) modulerelease.ReleaseMetadata {
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
