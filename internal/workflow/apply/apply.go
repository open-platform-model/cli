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
	"github.com/open-platform-model/cli/internal/version"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// now is the clock Execute reads the start of the apply from; tests move it
// back to stand for an apply that already spent part of its budget.
var now = time.Now

type Options struct {
	DryRun                 bool
	CreateNS               bool
	NoPrune                bool
	Force                  bool
	SuccessUpToDateMessage string
	SuccessAppliedMessage  string

	// Wait, in CLI-executor mode, blocks after a successful apply and
	// inventory write until every applied resource is healthy (see
	// kubernetes.HealthyPredicate) or Timeout runs out. Ignored on dry-run. An
	// operator-managed instance always waits for the operator, so the flag
	// changes nothing there.
	Wait bool

	// Timeout bounds the operator-reconcile wait in thin-editor mode. In
	// CLI-executor mode it bounds the CustomResourceDefinition establish wait,
	// counted from the start of the apply, and, separately and in full, the
	// readiness wait (Wait). Zero uses inventory.DefaultReconcileTimeout.
	Timeout time.Duration

	// SkipOperatorCeiling skips the operator-version ceiling gate. Only
	// `opm operator install` sets it, for the operator's own instance:
	// install replaces the operator the Platform reports rather than driving
	// it, and applies the same MAJOR.MINOR rule to the operator it installs
	// (0021:D9:R3, 0021:D9:R4). The CRD presence and field-floor gates still
	// run.
	SkipOperatorCeiling bool

	// SkipUnprovided is the command's --skip-unprovided. An operator-managed
	// instance refuses it: the operator renders that instance and never
	// skips.
	SkipUnprovided bool
}

type Request struct {
	Result    *workflowrender.Result
	K8sClient *kubernetes.Client
	Log       *log.Logger
	Options   Options
	// Admit are existing objects the first-install existence check lets
	// pass its untracked test. Only `opm operator install` sets it, to the
	// objects its migration proved; nil for every other apply.
	Admit inventory.AdmitSet
}

func Execute(ctx context.Context, req Request) error { //nolint:gocyclo // orchestration for apply flow spans gates, apply, prune, and CR spec+status writes
	result := req.Result
	instanceLog := req.Log
	namespace := result.Instance.Namespace
	name := result.Instance.Name
	instanceID := result.Instance.UUID
	dryRun := req.Options.DryRun

	wouldCreateNS, err := EnsureNamespaceIfRequested(ctx, req.K8sClient, namespace, req.Options.CreateNS, dryRun, instanceLog)
	if err != nil {
		return err
	}
	var newNamespaces []string
	if wouldCreateNS {
		newNamespaces = []string{namespace}
	}

	// The library's shared render digest, computed by the render workflow
	// over the render's single export; it leaves the managed-by value out so
	// the CLI and the operator digest one render equally (0012:D6).
	manifestDigest := result.RenderDigest
	output.Debug("render digest computed", "digest", manifestDigest)

	// Pre-apply gates 1-3 (cluster probes). Skipped entirely on dry-run — they
	// exist to protect writes, and a dry-run writes nothing (0006:D5).
	if !dryRun {
		if err := RunClusterGates(ctx, req.K8sClient, req.Options.SkipOperatorCeiling); err != nil {
			// Not Printed: the gates return bare errors without logging, so
			// claiming otherwise makes a missing CRD exit silently.
			return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
		}
	}

	// Ownership has to be resolved on dry-run too: with no record the resolver
	// below would see an absent CR and preview the CLI-executor path, promising
	// resource applies that an operator-owned instance would never receive.
	// --dry-run is server-side, so the client is connected and this read is safe.
	if dryRun && instanceID != "" {
		rec, err := inventory.GetRecord(ctx, req.K8sClient, name, namespace)
		if err != nil {
			return unreadableRecordError(name, namespace, err)
		}
		if inventory.ResolveOwnership(rec) == inventory.ModeOperatorOwned {
			return previewThinEditor(req, rec)
		}
	}

	// Load the previous inventory from the CR; when absent, look for a legacy
	// Secret to migrate. Both are read-only, so a dry-run loads them too and
	// can report what a real apply would prune.
	prevRecord, legacy, err := LoadPreviousInventory(ctx, req.K8sClient, name, namespace, instanceID, dryRun, instanceLog)
	if err != nil {
		return unreadableRecordError(name, namespace, err)
	}

	// Gate 4: ownership — the single branch point (0006:D18). An operator-owned
	// instance takes the thin-editor path and returns; everything below this
	// point is CLI-executor mode. A dry-run never takes this path: an
	// operator-owned instance was already previewed above.
	if !dryRun && inventory.ResolveOwnership(prevRecord) == inventory.ModeOperatorOwned {
		return executeThinEditor(ctx, req, prevRecord)
	}

	// Gate 5: status-RBAC pre-flight (CLI-executor mode, non-dry-run). Ensures
	// resources are never deployed without a recordable inventory.
	if !dryRun && instanceID != "" {
		if err := inventory.GateStatusRBAC(ctx, req.K8sClient, namespace); err != nil {
			// Not Printed — see RunClusterGates above.
			return &opmexit.ExitError{Code: opmexit.ExitPermissionDenied, Err: err}
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
	if err := RunPreApplyExistenceCheck(ctx, req.K8sClient, hasPrevInventory, dryRun, currentEntries, req.Admit); err != nil {
		return err
	}

	if dryRun {
		instanceLog.Info("dry run - no changes will be made")
	}
	if len(result.Resources) > 0 {
		instanceLog.Info(fmt.Sprintf("applying %d resources", len(result.Resources)))
	}

	// The CustomResourceDefinition establish wait inside the apply is charged
	// to a --timeout budget that starts with the apply; the --wait readiness
	// wait after it gets a fresh --timeout of its own.
	timeout := inventory.ResolveTimeout(req.Options.Timeout)
	budgetStart := now()

	var applyResult *kubernetes.ApplyResult
	if len(result.Resources) > 0 {
		var err error
		applyResult, err = kubernetes.Apply(ctx, req.K8sClient, result.Resources, name, kubernetes.ApplyOptions{
			DryRun:            dryRun,
			EstablishDeadline: budgetStart.Add(timeout),
			BudgetStart:       budgetStart,
			NewNamespaces:     newNamespaces,
		})
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
			instanceLog.Info(FormatDryRunSummary(applyResult))
		} else {
			instanceLog.Info(FormatApplySummary(applyResult))
		}
	}

	// CRDs and Namespaces are never pruned (kubernetes.IsProtectedKind); they
	// are listed as left behind instead, in the preview and the real run.
	prunable, protected := inventory.SplitProtected(staleSet)

	if dryRun && instanceID != "" && !req.Options.NoPrune {
		previewPrune(prunable, protected, instanceLog)
	}

	if !dryRun && instanceID != "" {
		applyHadErrors := applyResult != nil && len(applyResult.Errors) > 0
		if applyHadErrors {
			instanceLog.Warn("apply had errors — skipping pruning and inventory write")
			return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)), Printed: true}
		}

		if !req.Options.NoPrune {
			if len(prunable) > 0 {
				instanceLog.Info(fmt.Sprintf("pruning %d stale resource(s)", len(prunable)))
				if err := inventory.PruneStaleResources(ctx, req.K8sClient, prunable); err != nil {
					instanceLog.Warn("pruning stale resources failed", "error", err)
				}
			}
			if len(protected) > 0 {
				instanceLog.Warn(fmt.Sprintf("leaving %d resource(s) behind", len(protected)))
				logLeftBehind(protected, instanceLog)
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
	}

	if applyResult != nil && len(applyResult.Errors) > 0 {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)), Printed: true}
	}

	if req.Options.Wait && !dryRun {
		return waitForHealthy(ctx, req, timeout, instanceLog)
	}

	return nil
}

// previewPrune reports what a real apply would do with the stale set, without
// deleting anything: the prunable half under "would prune", then the protected
// half (inventory.SplitProtected) as left behind.
func previewPrune(prunable, protected []k8sinventory.Entry, instanceLog *log.Logger) {
	if len(prunable) > 0 {
		instanceLog.Info(fmt.Sprintf("would prune %d stale resource(s)", len(prunable)))
		for _, e := range prunable {
			instanceLog.Info(output.FormatResourceLine(e.Kind, e.Namespace, e.Name, "would prune"))
		}
	}
	if len(protected) > 0 {
		instanceLog.Info(fmt.Sprintf("would leave %d resource(s) behind", len(protected)))
		logLeftBehind(protected, instanceLog)
	}
}

// logLeftBehind prints one left-behind line per protected stale entry.
func logLeftBehind(protected []k8sinventory.Entry, instanceLog *log.Logger) {
	for _, e := range protected {
		instanceLog.Warn(output.FormatResourceLine(e.Kind, e.Namespace, e.Name, output.StatusLeftBehind),
			"reason", kubernetes.ProtectedKindReason)
	}
}

// RunClusterGates runs the read-only pre-apply cluster gates in order: CRD
// presence, CRD field floor, operator-version ceiling. skipCeiling skips the
// last, for the operator's own install only (Options.SkipOperatorCeiling).
func RunClusterGates(ctx context.Context, client *kubernetes.Client, skipCeiling bool) error {
	if err := inventory.GateCRDPresent(ctx, client); err != nil {
		return err
	}
	if err := inventory.GateCRDFieldFloor(ctx, client); err != nil {
		return err
	}
	if skipCeiling {
		output.Debug("operator-version ceiling skipped: the operator's own install checks its target instead")
		return nil
	}
	return inventory.GateOperatorVersionCeiling(ctx, client, version.Version)
}

// EnsureNamespaceIfRequested creates the instance namespace when createNS is
// set and it is missing. On a dry run it creates nothing and reports
// wouldCreate: the namespace is missing, so the objects in it cannot be
// validated by the server.
func EnsureNamespaceIfRequested(ctx context.Context, k8sClient *kubernetes.Client, namespace string, createNS, dryRun bool, instanceLog *log.Logger) (wouldCreate bool, err error) {
	if !createNS || namespace == "" {
		return false, nil
	}

	created, err := k8sClient.EnsureNamespace(ctx, namespace, dryRun)
	if err != nil {
		instanceLog.Error("ensuring namespace", "error", err)
		return false, &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
	}
	if created {
		if dryRun {
			instanceLog.Info(fmt.Sprintf("namespace %q would be created", namespace))
		} else {
			instanceLog.Info(fmt.Sprintf("namespace %q created", namespace))
		}
	}
	return created && dryRun, nil
}

// LoadPreviousInventory reads the ModuleInstance CR for an instance. When no
// CR exists, it looks for a legacy inventory Secret to migrate (0006:D6).
// Returns no record and no legacy inventory on a missing instance ID or a
// first apply with no legacy Secret. Both reads are read-only, so dryRun only
// changes the wording of the migration message.
//
// A CR read that fails with anything but NotFound is returned as the error:
// only a NotFound answer proves there is no record, and a caller that went on
// without one would run a first install over an existing instance.
func LoadPreviousInventory(ctx context.Context, k8sClient *kubernetes.Client, name, namespace, instanceID string, dryRun bool, instanceLog *log.Logger) (*inventory.Record, *inventory.LegacyInventory, error) {
	if instanceID == "" {
		return nil, nil, nil
	}

	prevRecord, err := inventory.GetRecord(ctx, k8sClient, name, namespace)
	if err != nil {
		return nil, nil, err
	}
	if prevRecord != nil {
		return prevRecord, nil, nil
	}

	legacy, err := inventory.FindLegacySecretInventory(ctx, k8sClient, name, namespace, instanceID)
	if err != nil {
		instanceLog.Warn("could not read legacy inventory Secret, proceeding as first apply", "error", err)
		return nil, nil, nil
	}
	if legacy == nil {
		return nil, nil, nil
	}
	if dryRun {
		instanceLog.Info("legacy inventory Secret would be migrated to ModuleInstance CR")
	} else {
		instanceLog.Info("migrating legacy inventory Secret to ModuleInstance CR")
	}
	return nil, legacy, nil
}

// unreadableRecordError is the refusal for a ModuleInstance read that failed
// with anything but NotFound. The exit code follows the cause: permission
// denied, connectivity, or general.
func unreadableRecordError(name, namespace string, cause error) error {
	return &opmexit.ExitError{
		Code: exitCodeFromK8sError(cause),
		Err: fmt.Errorf("cannot read the ModuleInstance record %q in namespace %q: %w\n"+
			"apply stopped: without the record it cannot tell a first install from an existing instance.\n"+
			"Check that you can read moduleinstances.%s in that namespace, then run the command again",
			name, namespace, cause, inventory.GroupOpmodel),
	}
}

// WriteInstanceRecord writes the ModuleInstance CR spec, then its status subset
// on the status subresource, then (for a migration) deletes the ported legacy
// Secret only after the status write succeeds.
func WriteInstanceRecord(ctx context.Context, req Request, prevRecord *inventory.Record, legacy *inventory.LegacyInventory, currentEntries []k8sinventory.Entry, manifestDigest string, instanceLog *log.Logger) error {
	result := req.Result
	name := result.Instance.Name
	namespace := result.Instance.Namespace
	instanceID := result.Instance.UUID

	modulePath, moduleVersion := workflowrender.CanonicalModuleRef(result.Module)

	if _, err := inventory.ApplySpec(ctx, req.K8sClient, inventory.SpecInput{
		Name:             name,
		Namespace:        namespace,
		Owner:            inventory.OwnerCLI,
		ModulePath:       modulePath,
		ModuleVersion:    moduleVersion,
		Values:           result.Values,
		SourceLocal:      result.SourceLocal,
		SkippedContracts: SkippedContracts(result),
	}); err != nil {
		instanceLog.Warn("failed to write ModuleInstance spec", "error", err)
		return &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
	}

	revision := nextRevision(prevRecord, legacy)
	statusInput := inventory.StatusInput{
		Name:      name,
		Namespace: namespace,
		Inventory: inventory.Inventory{
			Revision: revision,
			Digest:   k8sinventory.Digest(currentEntries),
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

// SkippedContracts is the render's skipped demands as the
// "<component>=<fqn>" pairs the skipped-contracts annotation records, in
// build order; nil when the render skipped nothing.
func SkippedContracts(result *workflowrender.Result) []string {
	if len(result.Skipped) == 0 {
		return nil
	}
	pairs := make([]string, 0, len(result.Skipped))
	for _, s := range result.Skipped {
		pairs = append(pairs, s.Component+"="+s.FQN)
	}
	return pairs
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

func previousEntries(prevRecord *inventory.Record, legacy *inventory.LegacyInventory) []k8sinventory.Entry {
	switch {
	case prevRecord != nil:
		return prevRecord.Inventory.Entries
	case legacy != nil:
		return legacy.Inventory.Entries
	default:
		return nil
	}
}

func CurrentInventoryEntries(resources []*unstructured.Unstructured) []k8sinventory.Entry {
	entries := make([]k8sinventory.Entry, 0, len(resources))
	for _, r := range resources {
		entries = append(entries, k8sinventory.NewEntry(r))
	}
	return entries
}

// ComputeStaleInventorySet is the library's component-blind stale set: every
// previous entry that no current entry is the same object as. A component
// rename or an API version change leaves nothing stale (0012:D7).
func ComputeStaleInventorySet(prevEntries, currentEntries []k8sinventory.Entry) []k8sinventory.Entry {
	return k8sinventory.StaleSet(prevEntries, currentEntries)
}

func GuardEmptyRender(resourceCount int, prevEntries []k8sinventory.Entry, force bool, instanceLog *log.Logger) error {
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

func RunPreApplyExistenceCheck(ctx context.Context, k8sClient *kubernetes.Client, hasPrevInventory, dryRun bool, currentEntries []k8sinventory.Entry, admit inventory.AdmitSet) error {
	if hasPrevInventory || dryRun {
		return nil
	}
	if err := inventory.PreApplyExistenceCheck(ctx, k8sClient, currentEntries, admit); err != nil {
		return fmt.Errorf("pre-apply existence check failed: %w", err)
	}
	return nil
}

// FormatDryRunSummary is the closing line of a dry run: how many resources
// would be applied and, when any were skipped, how many and why.
func FormatDryRunSummary(r *kubernetes.ApplyResult) string {
	summary := fmt.Sprintf("dry run complete: %d resources would be applied", r.Applied)
	if r.Skipped > 0 {
		summary += fmt.Sprintf(", %d skipped (their CustomResourceDefinition or Namespace is created by this apply)", r.Skipped)
	}
	return summary
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
// instead of hashing "@"); 0006:D6 guarantees a non-empty canonical reference on
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
