package cmdutil

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// EnsureNamespaceIfRequested creates the target namespace when requested.
func EnsureNamespaceIfRequested(ctx context.Context, k8sClient *kubernetes.Client, namespace string, createNS, dryRun bool, releaseLog *log.Logger) error {
	if !createNS || namespace == "" {
		return nil
	}

	created, err := k8sClient.EnsureNamespace(ctx, namespace, dryRun)
	if err != nil {
		releaseLog.Error("ensuring namespace", "error", err)
		return &oerrors.ExitError{Code: ExitCodeFromK8sError(err), Err: err, Printed: true}
	}
	if created {
		if dryRun {
			releaseLog.Info(fmt.Sprintf("namespace %q would be created", namespace))
		} else {
			releaseLog.Info(fmt.Sprintf("namespace %q created", namespace))
		}
	}
	return nil
}

// LoadPreviousInventory reads the latest inventory when available.
func LoadPreviousInventory(ctx context.Context, k8sClient *kubernetes.Client, releaseName, namespace, releaseID string, dryRun bool, releaseLog *log.Logger) *inventory.InventorySecret {
	if releaseID == "" || dryRun {
		return nil
	}

	prevInventory, err := inventory.GetInventory(ctx, k8sClient, releaseName, namespace, releaseID)
	if err != nil {
		releaseLog.Warn("could not read inventory, proceeding without it", "error", err)
		return nil
	}
	return prevInventory
}

// PreviousInventoryEntries extracts the latest tracked entries from inventory.
func PreviousInventoryEntries(prevInventory *inventory.InventorySecret) []inventory.InventoryEntry {
	if prevInventory == nil || len(prevInventory.Index) == 0 {
		return nil
	}
	latestChangeID := prevInventory.Index[0]
	if latestChangeID == "" {
		return nil
	}
	latestChange := prevInventory.Changes[latestChangeID]
	if latestChange == nil {
		return nil
	}
	return latestChange.Inventory.Entries
}

// CurrentInventoryEntries builds inventory entries from rendered resources.
func CurrentInventoryEntries(resources []*unstructured.Unstructured) []inventory.InventoryEntry {
	entries := make([]inventory.InventoryEntry, 0, len(resources))
	for _, r := range resources {
		entries = append(entries, inventory.NewEntryFromResource(r))
	}
	return entries
}

// ComputeStaleInventorySet derives the prunable stale set after safety checks.
func ComputeStaleInventorySet(prevEntries, currentEntries []inventory.InventoryEntry) []inventory.InventoryEntry {
	staleSet := inventory.ComputeStaleSet(prevEntries, currentEntries)
	return inventory.ApplyComponentRenameSafetyCheck(staleSet, currentEntries)
}

// GuardEmptyRender enforces the empty-render safety gate.
func GuardEmptyRender(resourceCount int, prevEntries []inventory.InventoryEntry, force bool, releaseLog *log.Logger) error {
	if resourceCount != 0 {
		return nil
	}
	if len(prevEntries) > 0 && !force {
		return fmt.Errorf("render produced 0 resources but previous inventory has %d entries — this would prune all resources; use --force to proceed or --no-prune to skip pruning", len(prevEntries))
	}
	if len(prevEntries) == 0 {
		releaseLog.Info("no resources to apply")
	}
	return nil
}

// RunPreApplyExistenceCheck performs the first-apply collision check.
func RunPreApplyExistenceCheck(ctx context.Context, k8sClient *kubernetes.Client, prevInventory *inventory.InventorySecret, dryRun bool, currentEntries []inventory.InventoryEntry) error {
	if prevInventory != nil || dryRun {
		return nil
	}
	if err := inventory.PreApplyExistenceCheck(ctx, k8sClient, currentEntries); err != nil {
		return fmt.Errorf("pre-apply existence check failed: %w", err)
	}
	return nil
}
