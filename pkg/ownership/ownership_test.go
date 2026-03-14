package ownership

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCLIMutable_AllowsLegacyAndCLI(t *testing.T) {
	assert.NoError(t, EnsureCLIMutable("", "my-release", "default"))
	assert.NoError(t, EnsureCLIMutable(CreatedByCLI, "my-release", "default"))
}

func TestEnsureCLIMutable_BlocksControllerManagedRelease(t *testing.T) {
	err := EnsureCLIMutable(CreatedByController, "demo", "apps")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "controller-managed")
	assert.Contains(t, err.Error(), "demo")
}
