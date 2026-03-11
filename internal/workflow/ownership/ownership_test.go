package ownership

import (
	"testing"

	"github.com/opmodel/cli/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCLIMutable_AllowsLegacyAndCLI(t *testing.T) {
	assert.NoError(t, EnsureCLIMutable(nil))
	assert.NoError(t, EnsureCLIMutable(&inventory.ReleaseInventoryRecord{}))
	assert.NoError(t, EnsureCLIMutable(&inventory.ReleaseInventoryRecord{CreatedBy: inventory.CreatedByCLI}))
}

func TestEnsureCLIMutable_BlocksControllerManagedRelease(t *testing.T) {
	err := EnsureCLIMutable(&inventory.ReleaseInventoryRecord{CreatedBy: inventory.CreatedByController, ReleaseMetadata: inventory.ReleaseMetadata{ReleaseName: "demo", ReleaseNamespace: "apps"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "controller-managed")
	assert.Contains(t, err.Error(), "demo")
}
