package ownership

import (
	"testing"

	pkginventory "github.com/opmodel/cli/pkg/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCLIMutable_AllowsLegacyAndCLI(t *testing.T) {
	assert.NoError(t, EnsureCLIMutable(nil))
	assert.NoError(t, EnsureCLIMutable(&pkginventory.InventorySecret{}))
	assert.NoError(t, EnsureCLIMutable(&pkginventory.InventorySecret{ReleaseMetadata: pkginventory.ReleaseMetadata{CreatedBy: pkginventory.CreatedByCLI}}))
}

func TestEnsureCLIMutable_BlocksControllerManagedRelease(t *testing.T) {
	err := EnsureCLIMutable(&pkginventory.InventorySecret{ReleaseMetadata: pkginventory.ReleaseMetadata{ReleaseName: "demo", ReleaseNamespace: "apps", CreatedBy: pkginventory.CreatedByController}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "controller-managed")
	assert.Contains(t, err.Error(), "demo")
}
