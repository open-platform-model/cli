package apply

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/charmbracelet/log"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/internal/version"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
	pkginventory "github.com/open-platform-model/cli/pkg/inventory"
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

type Request struct {
	Result    *workflowrender.Result
	K8sClient *kubernetes.Client
	Log       *log.Logger
	Options   Options
}

func Execute(ctx context.Context, req Request) error { //nolint:gocyclo // orchestration for apply flow spans gates, apply, prune, and CR spec+status writes
	result := req.Result
	instanceLog := req.Log
	namespace := result.Instance.Namespace
	name := result.Instance.Name
	instanceID := result.Instance.UUID
	dryRun := req.Options.DryRun

	if err := EnsureNamespaceIfRequested(ctx, req.K8sClient, namespace, req.Options.CreateNS, dryRun, instanceLog); err != nil {
		return err
	}

	// Operator-parity render digest, computed by the render workflow over the
	// kernel-compiled resources (0006 D9/D30 — see inventory.ComputeRenderDigest).
	manifestDigest := result.RenderDigest
	output.Debug("render digest computed", "digest", manifestDigest)

	// Pre-apply gates 1-3 (cluster probes). Skipped entirely on dry-run — they
	// exist to protect writes, and a dry-run writes nothing (enhancement 0006 D5).
	if !dryRun {
		if err := RunClusterGates(ctx, req.K8sClient); err != nil {
			return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
		}
	}

	// Load the previous inventory from the CR; when absent, look for a legacy
	// Secret to migrate. Both are read-only.
	prevRecord, legacy := LoadPreviousInventory(ctx, req.K8sClient, name, namespace, instanceID, dryRun, instanceLog)

	// Gate 4: ownership. Operator-owned instances are refused before any write.
	if inventory.ResolveOwnership(prevRecord) == inventory.ModeOperatorOwned {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: inventory.OperatorOwnedApplyError(name, namespace), Printed: true}
	}

	// Gate 5: status-RBAC pre-flight (CLI-executor mode, non-dry-run). Ensures
	// resources are never deployed without a recordable inventory.
	if !dryRun && instanceID != "" {
		if err := inventory.GateStatusRBAC(ctx, req.K8sClient, namespace); err != nil {
			return &opmexit.ExitError{Code: opmexit.ExitPermissionDenied, Err: err, Printed: true}
		}
	}

	prevEntries := previousEntries(prevRecord, legacy)
	currentEntries := CurrentInventoryEntries(result.Resources)
	staleSet := ComputeStaleInventorySet(prevEntries, currentEntries)

	if err := GuardEmptyRender(len(result.Resources), prevEntries, req.Options.Force, instanceLog); err != nil {
		return err
	}
	if len(result.Resources) == 0 && len(prevEntries) == 0 {
		return nil
	}

	// Gate 6: existence check, first-ever apply only (no previous inventory).
	hasPrevInventory := prevRecord != nil || legacy != nil
	if err := RunPreApplyExistenceCheck(ctx, req.K8sClient, hasPrevInventory, dryRun, currentEntries); err != nil {
		return err
	}

	if dryRun {
		instanceLog.Info("dry run - no changes will be made")
	}
	if len(result.Resources) > 0 {
		instanceLog.Info(fmt.Sprintf("applying %d resources", len(result.Resources)))
	}

	var applyResult *kubernetes.ApplyResult
	if len(result.Resources) > 0 {
		var err error
		applyResult, err = kubernetes.Apply(ctx, req.K8sClient, result.Resources, name, kubernetes.ApplyOptions{DryRun: dryRun})
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

		if dryRun {
			instanceLog.Info(fmt.Sprintf("dry run complete: %d resources would be applied", applyResult.Applied))
		} else {
			instanceLog.Info(FormatApplySummary(applyResult))
		}
	}

	if !dryRun && instanceID != "" {
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

		if err := WriteInstanceRecord(ctx, req, prevRecord, legacy, currentEntries, manifestDigest, instanceLog); err != nil {
			return err
		}
	}

	if applyResult != nil && len(applyResult.Errors) == 0 && !dryRun {
		if applyResult.Unchanged == applyResult.Applied {
			output.Println(output.FormatCheckmark(req.Options.SuccessUpToDateMessage))
		} else {
			output.Println(output.FormatCheckmark(req.Options.SuccessAppliedMessage))
		}

		// Solo-cluster Platform seeding (0006 D12/D22): when the render fell
		// back from the cluster to the local default platform, seed the
		// singleton cluster Platform write-if-absent so the operator adopts
		// it on install. The seeded document is the exact resolved spec the
		// render consumed (Result.PlatformSpec — no file re-read, no TOCTOU).
		// Best-effort — Ensure degrades on RBAC and no-ops on AlreadyExists;
		// only unexpected errors surface as warnings.
		if result.Platform.Source == platform.SourceLocalDefault && result.Platform.Warning != "" {
			if seedErr := platform.EnsureClusterPlatform(ctx, req.K8sClient.Dynamic, result.PlatformSpec); seedErr != nil {
				instanceLog.Warn("could not seed cluster Platform", "error", seedErr)
			}
		}
	}

	if applyResult != nil && len(applyResult.Errors) > 0 {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)), Printed: true}
	}

	return nil
}

// RunClusterGates runs the read-only pre-apply cluster gates in order: CRD
// presence, CRD field floor, operator-version ceiling.
func RunClusterGates(ctx context.Context, client *kubernetes.Client) error {
	if err := inventory.GateCRDPresent(ctx, client); err != nil {
		return err
	}
	if err := inventory.GateCRDFieldFloor(ctx, client); err != nil {
		return err
	}
	return inventory.GateOperatorVersionCeiling(ctx, client, version.Version)
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

// LoadPreviousInventory reads the ModuleInstance CR for an instance. When no CR
// exists, it looks for a legacy inventory Secret to migrate (enhancement 0006
// D6). Returns (nil, nil) on dry-run, missing instance ID, or a first apply
// with no legacy Secret.
func LoadPreviousInventory(ctx context.Context, k8sClient *kubernetes.Client, name, namespace, instanceID string, dryRun bool, instanceLog *log.Logger) (*inventory.Record, *inventory.LegacyInventory) {
	if instanceID == "" || dryRun {
		return nil, nil
	}

	prevRecord, err := inventory.GetRecord(ctx, k8sClient, name, namespace)
	if err != nil {
		instanceLog.Warn("could not read inventory CR, proceeding without it", "error", err)
		return nil, nil
	}
	if prevRecord != nil {
		return prevRecord, nil
	}

	legacy, err := inventory.FindLegacySecretInventory(ctx, k8sClient, name, namespace, instanceID)
	if err != nil {
		instanceLog.Warn("could not read legacy inventory Secret, proceeding as first apply", "error", err)
		return nil, nil
	}
	if legacy == nil {
		return nil, nil
	}
	instanceLog.Info("migrating legacy inventory Secret to ModuleInstance CR")
	return nil, legacy
}

// WriteInstanceRecord writes the ModuleInstance CR spec, then its status subset
// on the status subresource, then (for a migration) deletes the ported legacy
// Secret only after the status write succeeds.
func WriteInstanceRecord(ctx context.Context, req Request, prevRecord *inventory.Record, legacy *inventory.LegacyInventory, currentEntries []inventory.InventoryEntry, manifestDigest string, instanceLog *log.Logger) error {
	result := req.Result
	name := result.Instance.Name
	namespace := result.Instance.Namespace
	instanceID := result.Instance.UUID

	modulePath, moduleVersion := result.Module.CanonicalModuleRef()

	if err := inventory.ApplySpec(ctx, req.K8sClient, inventory.SpecInput{
		Name:          name,
		Namespace:     namespace,
		Owner:         inventory.OwnerCLI,
		ModulePath:    modulePath,
		ModuleVersion: moduleVersion,
		Values:        result.Values,
		SourceLocal:   result.SourceLocal,
	}); err != nil {
		instanceLog.Warn("failed to write ModuleInstance spec", "error", err)
		return &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
	}

	revision := nextRevision(prevRecord, legacy)
	statusInput := inventory.StatusInput{
		Name:      name,
		Namespace: namespace,
		Inventory: pkginventory.Inventory{
			Revision: revision,
			Digest:   inventory.ComputeDigest(currentEntries),
			Count:    len(currentEntries),
			Entries:  currentEntries,
		},
		InstanceUUID:            inventory.ExtractInstanceUUID(result.Resources),
		LastAppliedRenderDigest: manifestDigest,
		LastAppliedSourceDigest: sourceDigest(modulePath, moduleVersion),
		LastAppliedConfigDigest: valuesDigest(result.Values),
		LastAppliedAt:           time.Now().UTC().Format(time.RFC3339),
	}
	if statusInput.InstanceUUID == "" {
		statusInput.InstanceUUID = instanceID
	}

	if err := inventory.ApplyStatus(ctx, req.K8sClient, statusInput); err != nil {
		instanceLog.Warn("failed to write ModuleInstance status", "error", err)
		return &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
	}
	output.Debug("inventory written to ModuleInstance CR", "revision", revision)

	// Delete the migrated (or leftover) legacy Secret only after the status
	// write succeeds, so a failure leaves the Secret authoritative for a re-run.
	cleanupLegacySecret(ctx, req.K8sClient, name, namespace, instanceID, legacy, instanceLog)
	return nil
}

func cleanupLegacySecret(ctx context.Context, client *kubernetes.Client, name, namespace, instanceID string, legacy *inventory.LegacyInventory, instanceLog *log.Logger) {
	secretName := inventory.LegacySecretName(name, instanceID)
	secretNS := namespace
	if legacy != nil {
		secretName = legacy.SecretName
		secretNS = legacy.SecretNamespace
	}
	if err := inventory.DeleteLegacySecret(ctx, client, secretName, secretNS); err != nil {
		instanceLog.Warn("could not delete legacy inventory Secret", "error", err)
	}
}

func nextRevision(prevRecord *inventory.Record, legacy *inventory.LegacyInventory) int {
	prev := 0
	switch {
	case prevRecord != nil:
		prev = prevRecord.Inventory.Revision
	case legacy != nil:
		prev = legacy.Inventory.Revision
	}
	if prev < 0 {
		prev = 0
	}
	return prev + 1
}

func previousEntries(prevRecord *inventory.Record, legacy *inventory.LegacyInventory) []inventory.InventoryEntry {
	switch {
	case prevRecord != nil:
		return prevRecord.Inventory.Entries
	case legacy != nil:
		return legacy.Inventory.Entries
	default:
		return nil
	}
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

func RunPreApplyExistenceCheck(ctx context.Context, k8sClient *kubernetes.Client, hasPrevInventory, dryRun bool, currentEntries []inventory.InventoryEntry) error {
	if hasPrevInventory || dryRun {
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

// sourceDigest returns a deterministic digest identifying the module source of
// this apply, derived from the canonical module reference (path@version). For
// any non-empty reference this is byte-identical to the operator's
// ModuleSourceDigest (opm-operator internal/status): on the CUE-native
// resolution path BOTH actors use the reference-identity digest — there is no
// Flux artifact content digest here. Do not change one side without the
// other. The empty-reference guard below is CLI-only (omits the status field
// instead of hashing "@"); D6 guarantees a non-empty canonical reference on
// every real apply, so the divergence is unreachable in practice.
func sourceDigest(modulePath, moduleVersion string) string {
	if modulePath == "" && moduleVersion == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(modulePath + "@" + moduleVersion))
	return fmt.Sprintf("sha256:%x", sum)
}

// valuesDigest returns a deterministic digest of the unified values blob.
// Canonical-JSON semantics match the operator's ConfigDigest, including the
// empty case (SHA-256 of no bytes), so the field is cross-actor comparable.
func valuesDigest(values map[string]any) string {
	if len(values) == 0 {
		sum := sha256.Sum256(nil)
		return fmt.Sprintf("sha256:%x", sum)
	}
	b, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return fmt.Sprintf("sha256:%x", sum)
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
