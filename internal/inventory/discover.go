package inventory

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// DiscoverResourcesFromInventory fetches the live state of each resource
// tracked in the inventory. It performs one targeted GET per entry
// (N API calls for N resources) rather than scanning all API types.
//
// Returns:
//   - live: resources that currently exist on the cluster
//   - missing: inventory entries whose resources no longer exist on the cluster
func DiscoverResourcesFromInventory(ctx context.Context, client *kubernetes.Client, inv *InventorySecret) (live []*unstructured.Unstructured, missing []InventoryEntry, err error) {
	if inv == nil || len(inv.Index) == 0 {
		return nil, nil, nil
	}

	// Get entries from the latest change
	latestChangeID := inv.Index[0]
	latestChange, ok := inv.Changes[latestChangeID]
	if !ok || latestChange == nil {
		return nil, nil, nil
	}

	entries := latestChange.Inventory.Entries

	for _, entry := range entries {
		gvr := schema.GroupVersionResource{
			Group:    entry.Group,
			Version:  entry.Version,
			Resource: kindToResource(entry.Kind),
		}

		obj, getErr := client.ResourceClient(gvr, entry.Namespace).Get(ctx, entry.Name, metav1.GetOptions{})
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				missing = append(missing, entry)
				output.Debug("inventory resource missing from cluster",
					"kind", entry.Kind, "namespace", entry.Namespace, "name", entry.Name)
				continue
			}
			// Other errors — log and skip (don't treat as missing)
			output.Debug("could not fetch inventory resource",
				"kind", entry.Kind, "name", entry.Name, "err", getErr)
			continue
		}

		live = append(live, obj)
	}

	return live, missing, nil
}
