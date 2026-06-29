package ownership

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCLIMutable_AllowsLegacyAndCLI(t *testing.T) {
	assert.NoError(t, EnsureCLIMutable("", "my-instance", "default"))
	assert.NoError(t, EnsureCLIMutable(CreatedByCLI, "my-instance", "default"))
}

func TestEnsureCLIMutable_BlocksControllerManagedInstance(t *testing.T) {
	err := EnsureCLIMutable(CreatedByController, "demo", "apps")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "controller-managed")
	assert.Contains(t, err.Error(), "demo")
}
