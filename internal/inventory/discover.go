package inventory

import (
	"context"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// discoverConcurrency bounds the in-flight GETs issued for a single instance's
// inventory. Large instances (CRD-heavy platform modules) carry 50+ entries, so
// fetching them serially made one instance dominate the whole command's latency.
const discoverConcurrency = 8

// UnreadableEntry is an inventory entry whose live object could not be read:
// its GET failed with an error other than NotFound.
type UnreadableEntry struct {
	Entry InventoryEntry
	Err   error
}

// Resource converts the entry into the form kubernetes.Delete and
// kubernetes.GetInstanceStatus take (internal/kubernetes cannot import this
// package).
func (u UnreadableEntry) Resource() kubernetes.UnreadableResource {
	return kubernetes.UnreadableResource{
		Group:     u.Entry.Group,
		Kind:      u.Entry.Kind,
		Namespace: u.Entry.Namespace,
		Name:      u.Entry.Name,
		Err:       u.Err,
	}
}

// UnreadableResources converts every entry with Resource.
func UnreadableResources(unreadable []UnreadableEntry) []kubernetes.UnreadableResource {
	if len(unreadable) == 0 {
		return nil
	}
	out := make([]kubernetes.UnreadableResource, len(unreadable))
	for i, u := range unreadable {
		out[i] = u.Resource()
	}
	return out
}

// DiscoverResourcesFromInventory fetches the live state of each resource
// tracked in the inventory. It performs one targeted GET per entry
// (N API calls for N resources) rather than scanning all API types.
//
// Every entry lands in exactly one of three groups, each in inventory order:
//   - live: resources that currently exist on the cluster
//   - missing: inventory entries whose GET returned NotFound
//   - unreadable: inventory entries whose GET failed with any other error
//     (Forbidden, a timeout, a 5xx), with that error
//
// The error result is always nil: a failed read is reported per entry.
func DiscoverResourcesFromInventory(ctx context.Context, client *kubernetes.Client, inv *Record) (live []*unstructured.Unstructured, missing []InventoryEntry, unreadable []UnreadableEntry, err error) {
	if inv == nil || len(inv.Inventory.Entries) == 0 {
		return nil, nil, nil, nil
	}

	entries := inv.Inventory.Entries

	// Fetch entries concurrently. Results are collected by index so that every
	// group keeps inventory order regardless of completion order.
	type entryResult struct {
		obj     *unstructured.Unstructured
		missing bool
		readErr error
	}
	results := make([]entryResult, len(entries))

	var wg sync.WaitGroup
	sem := make(chan struct{}, discoverConcurrency)

	for i, entry := range entries {
		wg.Add(1)
		go func(idx int, entry InventoryEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			gvr := schema.GroupVersionResource{
				Group:    entry.Group,
				Version:  entry.Version,
				Resource: kubernetes.KindToResource(entry.Kind),
			}

			obj, getErr := client.ResourceClient(gvr, entry.Namespace).Get(ctx, entry.Name, metav1.GetOptions{})
			if getErr != nil {
				if apierrors.IsNotFound(getErr) {
					results[idx] = entryResult{missing: true}
					output.Debug("inventory resource missing from cluster",
						"kind", entry.Kind, "namespace", entry.Namespace, "name", entry.Name)
					return
				}
				// Any other error: the entry is unreadable, not missing.
				results[idx] = entryResult{readErr: getErr}
				output.Debug("could not fetch inventory resource",
					"kind", entry.Kind, "name", entry.Name, "err", getErr)
				return
			}

			results[idx] = entryResult{obj: obj}
		}(i, entry)
	}
	wg.Wait()

	for i, r := range results {
		switch {
		case r.missing:
			missing = append(missing, entries[i])
		case r.readErr != nil:
			unreadable = append(unreadable, UnreadableEntry{Entry: entries[i], Err: r.readErr})
		case r.obj != nil:
			live = append(live, r.obj)
		}
	}

	return live, missing, unreadable, nil
}
