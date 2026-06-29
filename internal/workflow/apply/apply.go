package apply

import (
	"context"
	"fmt"
	"strings"
	"time"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/charmbracelet/log"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	workflowrender "github.com/opmodel/cli/internal/workflow/render"
	pkginventory "github.com/opmodel/cli/pkg/inventory"
	"github.com/opmodel/cli/pkg/ownership"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Options struct {
	DryRun                 bool
	CreateNS               bool
	NoPrune                bool
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

func Execute(ctx context.Context, req Request) error { //nolint:gocyclo // orchestration for apply flow spans validation, apply, prune, and inventory write
	result := req.Result
	instanceLog := req.Log
	namespace := result.Instance.Namespace

	if err := EnsureNamespaceIfRequested(ctx, req.K8sClient, namespace, req.Options.CreateNS, req.Options.DryRun, instanceLog); err != nil {
		return err
	}

	manifestDigest := inventory.ComputeManifestDigest(result.Resources)
	output.Debug("manifest digest computed", "digest", manifestDigest)

	instanceID := result.Instance.UUID
	prevInventory := LoadPreviousInventory(ctx, req.K8sClient, result.Instance.Name, namespace, instanceID, req.Options.DryRun, instanceLog)
	if prevInventory != nil {
		if err := ownership.EnsureCLIMutable(string(prevInventory.NormalizedCreatedBy()), prevInventory.InstanceMetadata.InstanceName, prevInventory.InstanceMetadata.InstanceNamespace); err != nil {
			return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
		}
	}
	prevEntries := PreviousInventoryEntries(prevInventory)
	currentEntries := CurrentInventoryEntries(result.Resources)
	staleSet := ComputeStaleInventorySet(prevEntries, currentEntries)

	if err := GuardEmptyRender(len(result.Resources), prevEntries, req.Options.Force, instanceLog); err != nil {
		return err
	}
	if len(result.Resources) == 0 && len(prevEntries) == 0 {
		return nil
	}

	if err := RunPreApplyExistenceCheck(ctx, req.K8sClient, prevInventory, req.Options.DryRun, currentEntries); err != nil {
		return err
	}

	if req.Options.DryRun {
		instanceLog.Info("dry run - no changes will be made")
	}
	if len(result.Resources) > 0 {
		instanceLog.Info(fmt.Sprintf("applying %d resources", len(result.Resources)))
	}

	var applyResult *kubernetes.ApplyResult
	if len(result.Resources) > 0 {
		var err error
		applyResult, err = kubernetes.Apply(ctx, req.K8sClient, result.Resources, result.Instance.Name, kubernetes.ApplyOptions{DryRun: req.Options.DryRun})
		if err != nil {
			instanceLog.Error("apply failed", "error", err)
			return &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
		}

		if len(applyResult.Errors) > 0 {
			instanceLog.Warn(fmt.Sprintf("%d resource(s) had errors", len(applyResult.Errors)))
			for _, e := range applyResult.Errors {
				instanceLog.Error(e.Error())
			}
		}

		if req.Options.DryRun {
			instanceLog.Info(fmt.Sprintf("dry run complete: %d resources would be applied", applyResult.Applied))
		} else {
			instanceLog.Info(FormatApplySummary(applyResult))
		}
	}

	if !req.Options.DryRun && instanceID != "" {
		applyHadErrors := applyResult != nil && len(applyResult.Errors) > 0
		if applyHadErrors {
			instanceLog.Warn("apply had errors — skipping pruning and inventory write")
			return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)), Printed: true}
		}

		if len(staleSet) > 0 && !req.Options.NoPrune {
			instanceLog.Info(fmt.Sprintf("pruning %d stale resource(s)", len(staleSet)))
			if err := inventory.PruneStaleResources(ctx, req.K8sClient, staleSet); err != nil {
				instanceLog.Warn("pruning stale resources failed", "error", err)
			}
		}

		newOrUpdatedInventory := prevInventory
		if newOrUpdatedInventory == nil {
			newOrUpdatedInventory = &inventory.InstanceInventoryRecord{
				InstanceMetadata: inventory.InstanceMetadata{
					Kind:              "ModuleInstance",
					APIVersion:        inventory.APIVersionV1Alpha1,
					InstanceName:      result.Instance.Name,
					InstanceNamespace: namespace,
					InstanceID:        instanceID,
				},
			}
		}

		revision := 1
		if prevInventory != nil && prevInventory.Inventory.Revision > 0 {
			revision = prevInventory.Inventory.Revision + 1
		}
		newOrUpdatedInventory.InstanceMetadata.LastTransitionTime = time.Now().UTC().Format(time.RFC3339)
		newOrUpdatedInventory.Inventory = pkginventory.Inventory{
			Revision: revision,
			Digest:   inventory.ComputeDigest(currentEntries),
			Count:    len(currentEntries),
			Entries:  currentEntries,
		}

		if err := inventory.WriteInventory(ctx, req.K8sClient, newOrUpdatedInventory, req.ModuleName, req.ModuleUUID, req.Change.Version, inventory.CreatedByCLI); err != nil {
			instanceLog.Warn("failed to write inventory Secret", "error", err)
		} else {
			output.Debug("inventory written", "revision", revision)
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

func EnsureNamespaceIfRequested(ctx context.Context, k8sClient *kubernetes.Client, namespace string, createNS, dryRun bool, instanceLog *log.Logger) error {
	if !createNS || namespace == "" {
		return nil
	}

	created, err := k8sClient.EnsureNamespace(ctx, namespace, dryRun)
	if err != nil {
		instanceLog.Error("ensuring namespace", "error", err)
		return &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
	}
	if created {
		if dryRun {
			instanceLog.Info(fmt.Sprintf("namespace %q would be created", namespace))
		} else {
			instanceLog.Info(fmt.Sprintf("namespace %q created", namespace))
		}
	}
	return nil
}

func LoadPreviousInventory(ctx context.Context, k8sClient *kubernetes.Client, instanceName, namespace, instanceID string, dryRun bool, instanceLog *log.Logger) *inventory.InstanceInventoryRecord {
	if instanceID == "" || dryRun {
		return nil
	}

	prevInventory, err := inventory.GetInventory(ctx, k8sClient, instanceName, namespace, instanceID)
	if err != nil {
		instanceLog.Warn("could not read inventory, proceeding without it", "error", err)
		return nil
	}
	return prevInventory
}

func PreviousInventoryEntries(prevInventory *inventory.InstanceInventoryRecord) []inventory.InventoryEntry {
	if prevInventory == nil {
		return nil
	}
	return prevInventory.Inventory.Entries
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

func GuardEmptyRender(resourceCount int, prevEntries []inventory.InventoryEntry, force bool, instanceLog *log.Logger) error {
	if resourceCount != 0 {
		return nil
	}
	if len(prevEntries) > 0 && !force {
		return fmt.Errorf("render produced 0 resources but previous inventory has %d entries — this would prune all resources; use --force to proceed or --no-prune to skip pruning", len(prevEntries))
	}
	if len(prevEntries) == 0 {
		instanceLog.Info("no resources to apply")
	}
	return nil
}

func RunPreApplyExistenceCheck(ctx context.Context, k8sClient *kubernetes.Client, prevInventory *inventory.InstanceInventoryRecord, dryRun bool, currentEntries []inventory.InventoryEntry) error {
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
