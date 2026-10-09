package cmdutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

func TestNoRegistryError(t *testing.T) {
	cause := errors.New("module opmodel.dev/core@v2.0.0-beta.4: module not found")

	t.Run("nothing configured exits 2 and points to config init", func(t *testing.T) {
		t.Setenv("CUE_REGISTRY", "")
		err := NoRegistryError(&config.GlobalConfig{}, "loading core schema", cause)
		var exitErr *opmexit.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
		assert.False(t, exitErr.Printed, "main prints it")
		var noReg *config.NoRegistryError
		require.ErrorAs(t, err, &noReg)
		assert.Contains(t, err.Error(), "no registry is configured")
		assert.Contains(t, err.Error(), "opm config init")
		assert.Contains(t, err.Error(), cause.Error())
	})

	t.Run("a resolved registry is configured", func(t *testing.T) {
		t.Setenv("CUE_REGISTRY", "")
		assert.NoError(t, NoRegistryError(&config.GlobalConfig{Registry: "localhost:5000"}, "loading core schema", cause))
	})

	t.Run("only CUE_REGISTRY is configured", func(t *testing.T) {
		t.Setenv("CUE_REGISTRY", "opmodel.dev=ghcr.io/open-platform-model")
		assert.NoError(t, NoRegistryError(&config.GlobalConfig{}, "loading core schema", cause))
	})

	t.Run("no failure is no error", func(t *testing.T) {
		t.Setenv("CUE_REGISTRY", "")
		assert.NoError(t, NoRegistryError(&config.GlobalConfig{}, "loading core schema", nil))
	})
}
