package cmdutil

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
)

// ResolveInventory resolves the inventory Secret and discovers live resources
// for a release identified by rsf. It encapsulates the lookup-and-discover flow
// shared by mod delete and mod status.
//
// Resolution strategy:
//   - ReleaseID set: direct GET via inventory.GetInventory (release name used as display name if set)
//   - ReleaseName set: label scan via inventory.FindInventoryByReleaseName
//
// If the inventory is not found and ignoreNotFound is true, returns (nil, nil, nil, nil).
// If the inventory is not found and ignoreNotFound is false, returns an *ExitError with ExitNotFound.
// Any other lookup or discovery error returns an *ExitError with ExitGeneralError.
func ResolveInventory(
	ctx context.Context,
	client *kubernetes.Client,
	rsf *ReleaseSelectorFlags,
	namespace string,
	ignoreNotFound bool,
	releaseLog *log.Logger,
) (*inventory.InventorySecret, []*unstructured.Unstructured, []inventory.InventoryEntry, error) {
	// Resolve the inventory Secret.
	var inv *inventory.InventorySecret
	var invErr error
	switch {
	case rsf.ReleaseID != "":
		relName := rsf.ReleaseName
		if relName == "" {
			relName = rsf.ReleaseID
		}
		inv, invErr = inventory.GetInventory(ctx, client, relName, namespace, rsf.ReleaseID)
	case rsf.ReleaseName != "":
		inv, invErr = inventory.FindInventoryByReleaseName(ctx, client, rsf.ReleaseName, namespace)
	}

	if invErr != nil {
		releaseLog.Error("reading inventory", "error", invErr)
		return nil, nil, nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("reading inventory: %w", invErr)}
	}

	if inv == nil {
		name := rsf.ReleaseName
		if name == "" {
			name = rsf.ReleaseID
		}
		notFound := &kubernetes.ReleaseNotFoundError{Name: name, Namespace: namespace}
		if ignoreNotFound {
			releaseLog.Info("release not found (ignored)")
			return nil, nil, nil, nil
		}
		releaseLog.Error("release not found", "name", name, "namespace", namespace)
		return nil, nil, nil, &oerrors.ExitError{Code: oerrors.ExitNotFound, Err: notFound, Printed: true}
	}

	liveResources, missingEntries, discoverErr := inventory.DiscoverResourcesFromInventory(ctx, client, inv)
	if discoverErr != nil {
		releaseLog.Error("discovering resources from inventory", "error", discoverErr)
		return nil, nil, nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("discovering resources: %w", discoverErr)}
	}

	return inv, liveResources, missingEntries, nil
}
