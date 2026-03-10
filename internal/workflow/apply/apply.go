package apply

import (
	"context"
	"fmt"
	"strings"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/charmbracelet/log"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	workflowrender "github.com/opmodel/cli/internal/workflow/render"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Options struct {
	DryRun                 bool
	CreateNS               bool
	NoPrune                bool
	MaxHistory             int
	Force                  bool
	SuccessUpToDateMessage string
	SuccessAppliedMessage  string
}

type ChangeDescriptor struct {
	Path      string
	ValuesStr string
	Version   string
	Local     bool
}

type Request struct {
	Result     *workflowrender.Result
	K8sClient  *kubernetes.Client
	Log        *log.Logger
	Options    Options
	Change     ChangeDescriptor
	ModuleName string
	ModuleUUID string
}

func Execute(ctx context.Context, req Request) error { //nolint:gocyclo
	result := req.Result
	releaseLog := req.Log
	namespace := result.Release.Namespace

	if err := EnsureNamespaceIfRequested(ctx, req.K8sClient, namespace, req.Options.CreateNS, req.Options.DryRun, releaseLog); err != nil {
		return err
	}

	manifestDigest := inventory.ComputeManifestDigest(result.Resources)
	output.Debug("manifest digest computed", "digest", manifestDigest)

	changeID := inventory.ComputeChangeID(req.Change.Path, req.Change.Version, req.Change.ValuesStr, manifestDigest)
	output.Debug("change ID computed", "changeID", changeID)

	releaseID := result.Release.UUID
	prevInventory := LoadPreviousInventory(ctx, req.K8sClient, result.Release.Name, namespace, releaseID, req.Options.DryRun, releaseLog)
	prevEntries := PreviousInventoryEntries(prevInventory)
	currentEntries := CurrentInventoryEntries(result.Resources)
	staleSet := ComputeStaleInventorySet(prevEntries, currentEntries)

	if err := GuardEmptyRender(len(result.Resources), prevEntries, req.Options.Force, releaseLog); err != nil {
		return err
	}
	if len(result.Resources) == 0 && len(prevEntries) == 0 {
		return nil
	}

	if err := RunPreApplyExistenceCheck(ctx, req.K8sClient, prevInventory, req.Options.DryRun, currentEntries); err != nil {
		return err
	}

	if req.Options.DryRun {
		releaseLog.Info("dry run - no changes will be made")
	}
	if len(result.Resources) > 0 {
		releaseLog.Info(fmt.Sprintf("applying %d resources", len(result.Resources)))
	}

	var applyResult *kubernetes.ApplyResult
	if len(result.Resources) > 0 {
		var err error
		applyResult, err = kubernetes.Apply(ctx, req.K8sClient, result.Resources, result.Release.Name, kubernetes.ApplyOptions{DryRun: req.Options.DryRun})
		if err != nil {
			releaseLog.Error("apply failed", "error", err)
			return &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
		}

		if len(applyResult.Errors) > 0 {
			releaseLog.Warn(fmt.Sprintf("%d resource(s) had errors", len(applyResult.Errors)))
			for _, e := range applyResult.Errors {
				releaseLog.Error(e.Error())
			}
		}

		if req.Options.DryRun {
			releaseLog.Info(fmt.Sprintf("dry run complete: %d resources would be applied", applyResult.Applied))
		} else {
			releaseLog.Info(FormatApplySummary(applyResult))
		}
	}

	if !req.Options.DryRun && releaseID != "" {
		applyHadErrors := applyResult != nil && len(applyResult.Errors) > 0
		if applyHadErrors {
			releaseLog.Warn("apply had errors — skipping pruning and inventory write")
			return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)), Printed: true}
		}

		if len(staleSet) > 0 && !req.Options.NoPrune {
			releaseLog.Info(fmt.Sprintf("pruning %d stale resource(s)", len(staleSet)))
			if err := inventory.PruneStaleResources(ctx, req.K8sClient, staleSet); err != nil {
				releaseLog.Warn("pruning stale resources failed", "error", err)
			}
		}

		alreadyCurrent := prevInventory != nil && len(prevInventory.Index) > 0 && prevInventory.Index[0] == changeID
		if alreadyCurrent {
			output.Debug("inventory unchanged, skipping write", "changeID", changeID)
		} else {
			newOrUpdatedInventory := prevInventory
			if newOrUpdatedInventory == nil {
				newOrUpdatedInventory = &inventory.InventorySecret{
					ReleaseMetadata: inventory.ReleaseMetadata{
						Kind:             "ModuleRelease",
						APIVersion:       "core.opmodel.dev/v1alpha1",
						ReleaseName:      result.Release.Name,
						ReleaseNamespace: namespace,
						ReleaseID:        releaseID,
					},
					Index:   []string{},
					Changes: map[string]*inventory.ChangeEntry{},
				}
			}

			source := inventory.ChangeSource{
				Path:        req.Change.Path,
				Version:     req.Change.Version,
				ReleaseName: result.Release.Name,
				Local:       req.Change.Local,
			}

			computedChangeID, changeEntry := inventory.PrepareChange(source, req.Change.ValuesStr, manifestDigest, currentEntries)
			newOrUpdatedInventory.Changes[computedChangeID] = changeEntry
			newOrUpdatedInventory.Index = inventory.UpdateIndex(newOrUpdatedInventory.Index, computedChangeID)
			newOrUpdatedInventory.ReleaseMetadata.LastTransitionTime = changeEntry.Timestamp
			inventory.PruneHistory(newOrUpdatedInventory, req.Options.MaxHistory)

			if err := inventory.WriteInventory(ctx, req.K8sClient, newOrUpdatedInventory, req.ModuleName, req.ModuleUUID); err != nil {
				releaseLog.Warn("failed to write inventory Secret", "error", err)
			} else {
				output.Debug("inventory written", "changeID", changeID)
			}
		}
	}

	if applyResult != nil && len(applyResult.Errors) == 0 && !req.Options.DryRun {
		if applyResult.Unchanged == applyResult.Applied {
			output.Println(output.FormatCheckmark(req.Options.SuccessUpToDateMessage))
		} else {
			output.Println(output.FormatCheckmark(req.Options.SuccessAppliedMessage))
		}
	}

	if applyResult != nil && len(applyResult.Errors) > 0 {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)), Printed: true}
	}

	return nil
}

func EnsureNamespaceIfRequested(ctx context.Context, k8sClient *kubernetes.Client, namespace string, createNS, dryRun bool, releaseLog *log.Logger) error {
	if !createNS || namespace == "" {
		return nil
	}

	created, err := k8sClient.EnsureNamespace(ctx, namespace, dryRun)
	if err != nil {
		releaseLog.Error("ensuring namespace", "error", err)
		return &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
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

func CurrentInventoryEntries(resources []*unstructured.Unstructured) []inventory.InventoryEntry {
	entries := make([]inventory.InventoryEntry, 0, len(resources))
	for _, r := range resources {
		entries = append(entries, inventory.NewEntryFromResource(r))
	}
	return entries
}

func ComputeStaleInventorySet(prevEntries, currentEntries []inventory.InventoryEntry) []inventory.InventoryEntry {
	staleSet := inventory.ComputeStaleSet(prevEntries, currentEntries)
	return inventory.ApplyComponentRenameSafetyCheck(staleSet, currentEntries)
}

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

func RunPreApplyExistenceCheck(ctx context.Context, k8sClient *kubernetes.Client, prevInventory *inventory.InventorySecret, dryRun bool, currentEntries []inventory.InventoryEntry) error {
	if prevInventory != nil || dryRun {
		return nil
	}
	if err := inventory.PreApplyExistenceCheck(ctx, k8sClient, currentEntries); err != nil {
		return fmt.Errorf("pre-apply existence check failed: %w", err)
	}
	return nil
}

func FormatApplySummary(r *kubernetes.ApplyResult) string {
	var parts []string
	if r.Created > 0 {
		parts = append(parts, fmt.Sprintf("%d created", r.Created))
	}
	if r.Configured > 0 {
		parts = append(parts, fmt.Sprintf("%d configured", r.Configured))
	}
	if r.Unchanged > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", r.Unchanged))
	}
	summary := fmt.Sprintf("applied %d resources successfully", r.Applied)
	if len(parts) > 0 {
		summary += fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
	}
	return summary
}

func exitCodeFromK8sError(err error) int {
	switch {
	case apierrors.IsNotFound(err):
		return opmexit.ExitNotFound
	case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
		return opmexit.ExitPermissionDenied
	case apierrors.IsServerTimeout(err), apierrors.IsServiceUnavailable(err):
		return opmexit.ExitConnectivityError
	default:
		return opmexit.ExitGeneralError
	}
}
