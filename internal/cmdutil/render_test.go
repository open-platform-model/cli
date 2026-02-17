package cmdutil

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderModule_NilConfig(t *testing.T) {
	_, err := RenderModule(context.Background(), RenderModuleOpts{
		OPMConfig: nil,
		Render:    &RenderFlags{},
	})

	require.Error(t, err)
	var exitErr *ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, exitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "configuration not loaded")
}

func TestResolveModulePath_Defaults(t *testing.T) {
	assert.Equal(t, ".", ResolveModulePath(nil))
	assert.Equal(t, ".", ResolveModulePath([]string{}))
}

func TestResolveModulePath_Arg(t *testing.T) {
	assert.Equal(t, "./my-module", ResolveModulePath([]string{"./my-module"}))
}
