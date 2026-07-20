package instance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

func emptyClusterClient() *kubernetes.Client {
	listKinds := map[schema.GroupVersionResource]string{
		inventory.ModuleInstanceGVR: "ModuleInstanceList",
	}
	return &kubernetes.Client{
		Dynamic: fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds),
	}
}

func operatorOwnedRecord() *inventory.Record {
	return &inventory.Record{
		Name:      "podinfo",
		Namespace: "demo",
		Owner:     inventory.OwnerOperator,
		Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{{Kind: "Deployment", Name: "podinfo", Namespace: "demo"}}},
	}
}

// Deleting a finalizer-armed ModuleInstance with no controller running does not
// delete anything — it wedges the CR in Terminating with its workloads
// orphaned. So the readiness guard refuses rather than proceeding, and it says
// why (design LD7).
func TestDeleteOperatorOwned_RefusesWhenOperatorIsNotReady(t *testing.T) {
	err := deleteOperatorOwned(context.Background(), emptyClusterClient(), operatorOwnedRecord(),
		time.Second, false, output.InstanceLogger("test"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
	assert.Contains(t, err.Error(), inventory.CleanupFinalizer)
	assert.Contains(t, err.Error(), "Terminating")
	assert.Contains(t, err.Error(), "opm operator install")
}

// The guard runs before the dry-run short-circuit: a dry run that reported
// "would delete" against a down operator would be describing an outcome that
// cannot happen.
func TestDeleteOperatorOwned_DryRunStillRequiresAReadyOperator(t *testing.T) {
	err := deleteOperatorOwned(context.Background(), emptyClusterClient(), operatorOwnedRecord(),
		time.Second, true, output.InstanceLogger("test"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
}

// --force skips the confirmation prompt; it must not reach the readiness
// guard, since forcing past that guard produces the wedge rather than avoiding
// it. deleteOperatorOwned takes no force parameter at all — this pins the flag
// to its stated meaning so a later change cannot quietly widen it.
func TestDeleteForceFlagIsConfirmationOnly(t *testing.T) {
	cmd := NewInstanceDeleteCmd(&config.GlobalConfig{})
	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Contains(t, forceFlag.Usage, "confirmation")
}
