package cmdutil

import (
	"context"
	"errors"
	"testing"

	"github.com/opmodel/cli/internal/core"
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestShowRenderOutput_WithErrors(t *testing.T) {
	// Create a RenderResult with errors
	result := &pipeline.RenderResult{
		Release: core.ReleaseMetadata{
			Name:      "test-module",
			Namespace: "default",
		},
		Module: core.ModuleMetadata{
			Name: "test-module",
		},
		Errors: []error{
			&pipeline.UnmatchedComponentError{
				ComponentName: "web",
			},
		},
	}

	err := ShowRenderOutput(result, ShowOutputOpts{})

	// Should return ExitError with ValidationError code
	require.Error(t, err)
	var exitErr *oerrors.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, oerrors.ExitValidationError, exitErr.Code)
	assert.True(t, exitErr.Printed, "error should be marked as printed")
}

func TestShowRenderOutput_NoErrors_DefaultMode(t *testing.T) {
	// Create a RenderResult with no errors
	result := &pipeline.RenderResult{
		Release: core.ReleaseMetadata{
			Name:      "test-module",
			Namespace: "default",
		},
		Module: core.ModuleMetadata{
			Name: "test-module",
		},
		MatchPlan: core.MatchPlan{
			Matches: map[string][]core.TransformerMatchOld{
				"web": {
					{
						TransformerFQN: "example.com/transformers@v1#DeploymentTransformer",
						Reason:         "Matched: requiredResources[Container]",
					},
				},
			},
		},
		Resources: []*core.Resource{},
		Errors:    []error{},
	}

	err := ShowRenderOutput(result, ShowOutputOpts{Verbose: false})

	// Should not return an error
	assert.NoError(t, err)
}

func TestShowRenderOutput_Warnings(t *testing.T) {
	// Create a RenderResult with warnings
	result := &pipeline.RenderResult{
		Release: core.ReleaseMetadata{
			Name:      "test-module",
			Namespace: "default",
		},
		Module: core.ModuleMetadata{
			Name: "test-module",
		},
		MatchPlan: core.MatchPlan{
			Matches: map[string][]core.TransformerMatchOld{},
		},
		Resources: []*core.Resource{},
		Errors:    []error{},
		Warnings:  []string{"deprecated transformer used", "unused values"},
	}

	err := ShowRenderOutput(result, ShowOutputOpts{})

	// Should not return an error (warnings are non-fatal)
	assert.NoError(t, err)
	// Note: we don't capture the log output here, but the warning logging path is exercised
}
