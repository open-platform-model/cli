package apply

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
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

	// WarnUnrecorded makes a first install (no ModuleInstance record) warn
	// when rendered resources already exist in the cluster under OPM
	// management, on a real run and on a dry run. `opm instance apply` and
	// `opm module apply` set it. `opm operator install` does not: it applies
	// the render's CRDs itself before this workflow runs, so a fresh install
	// always finds them.
	WarnUnrecorded bool

	// AfterCallerWrites says that the caller wrote to the cluster in the same
	// command before this workflow runs, so a refusal here cannot say that
	// nothing was changed. Only `opm operator install` sets it: it applies
	// the render's CRDs and its migration first.
	AfterCallerWrites bool
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

	// --create-namespace only reads here. The namespace is created after the
	// last check that can refuse the apply, so that a refusal has changed
	// nothing. A missing namespace holds no record and no resource, so the
	// checks below send no read into it: the API would answer such a read
	// with Forbidden, not NotFound, for a caller whose rights come from a
	// RoleBinding the namespace does not have yet.
	createNamespace, err := namespaceToCreate(ctx, req.K8sClient, namespace, req.Options.CreateNS, dryRun, instanceLog)
	if err != nil {
		return err
	}
	var newNamespaces []string
	if createNamespace && dryRun {
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
	nothingChanged := !req.Options.AfterCallerWrites
	if dryRun && instanceID != "" && !createNamespace {
		rec, err := inventory.GetRecord(ctx, req.K8sClient, name, namespace)
		if err != nil {
			return unreadableRecordError(name, namespace, err, nothingChanged)
		}
		if inventory.ResolveOwnership(rec) == inventory.ModeOperatorOwned {
			return previewThinEditor(req, rec)
		}
	}

	// Load the previous inventory from the CR. The read is read-only, so a
	// dry-run loads it too and can report what a real apply would prune.
	var prevRecord *inventory.Record
	if !createNamespace {
		prevRecord, err = LoadPreviousInventory(ctx, req.K8sClient, name, namespace, instanceID)
		if err != nil {
			return unreadableRecordError(name, namespace, err, nothingChanged)
		}
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

	prevEntries := previousEntries(prevRecord)
	currentEntries := CurrentInventoryEntries(result.Resources)
	staleSet := ComputeStaleInventorySet(prevEntries, currentEntries)

	if err := GuardEmptyRender(len(result.Resources), prevEntries, req.Options.Force, instanceLog); err != nil {
		return err
	}
	if len(result.Resources) == 0 && len(prevEntries) == 0 {
		// Nothing to apply and nothing left that can refuse: the namespace
		// the flag asks for is still created.
		return ensureNamespace(ctx, req.K8sClient, namespace, createNamespace && !dryRun, instanceLog)
	}

	// Gate 6: existence check, first-ever apply only (no previous inventory).
	// Objects in a namespace this apply creates cannot exist yet.
	checkEntries := currentEntries
	if createNamespace {
		checkEntries = entriesOutside(currentEntries, namespace)
	}
	alreadyManaged, err := RunPreApplyExistenceCheck(ctx, req.K8sClient, prevRecord != nil, dryRun, checkEntries, req.Admit, nothingChanged)
	if err != nil {
		return err
	}
	if req.Options.WarnUnrecorded {
		if dryRun && prevRecord == nil {
			alreadyManaged = previewAlreadyManaged(ctx, req.K8sClient, checkEntries)
		}
		if len(alreadyManaged) > 0 {
			instanceLog.Warn(unrecordedResourcesWarning(len(alreadyManaged), len(currentEntries), name, instanceID, dryRun))
		}
	}

	// The first write of the apply: every check that can refuse has passed.
	if err := ensureNamespace(ctx, req.K8sClient, namespace, createNamespace && !dryRun, instanceLog); err != nil {
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

		// Entries prune failed to delete are still in the cluster: they stay
		// in the record, so the next apply finds them stale and retries.
		recordEntries := currentEntries
		var notPruned []k8sinventory.Entry
		pruneExit := opmexit.ExitGeneralError
		if !req.Options.NoPrune {
			if len(prunable) > 0 {
				instanceLog.Info(fmt.Sprintf("pruning %d stale resource(s)", len(prunable)))
				var err error
				notPruned, pruneExit, err = pruneStale(ctx, req.K8sClient, prunable, instanceLog)
				if err != nil {
					return err
				}
				recordEntries = append(append([]k8sinventory.Entry{}, currentEntries...), notPruned...)
			}
			if len(protected) > 0 {
				instanceLog.Warn(fmt.Sprintf("leaving %d resource(s) behind", len(protected)))
				logLeftBehind(protected, instanceLog)
			}
		}

		if err := WriteInstanceRecord(ctx, req, prevRecord, recordEntries, manifestDigest, instanceLog); err != nil {
			return err
		}

		if len(notPruned) > 0 {
			err := fmt.Errorf("%d stale resource(s) could not be pruned and stay in the inventory; fix the cause and run apply again to retry", len(notPruned))
			instanceLog.Error(err.Error())
			return &opmexit.ExitError{Code: pruneExit, Err: err, Printed: true}
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

// pruneStale deletes the prunable stale resources. It returns the entries it
// could not delete, each already reported on its own line with the delete
// error; the caller keeps them in the record and fails the command after the
// write with exitCode: 1, or, when a failed API discovery request stopped the
// prune, the code of that failure (4 denied, 3 unavailable). The error result
// is for a failure that names no entries, where the caller must stop before
// the write so that no entry is dropped unseen.
func pruneStale(ctx context.Context, client *kubernetes.Client, prunable []k8sinventory.Entry, instanceLog *log.Logger) (notPruned []k8sinventory.Entry, exitCode int, err error) {
	exitCode = opmexit.ExitGeneralError
	err = inventory.PruneStaleResources(ctx, client, prunable)
	if err == nil {
		return nil, exitCode, nil
	}
	var pruneErr *inventory.PruneError
	if !errors.As(err, &pruneErr) {
		instanceLog.Error("pruning stale resources failed", "error", err)
		return nil, exitCode, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err, Printed: true}
	}
	for i, e := range pruneErr.Failed {
		instanceLog.Error(output.FormatResourceLine(e.Kind, e.Namespace, e.Name, statusPruneFailed), "error", pruneErr.Errs[i])
		if kubernetes.IsDiscoveryFailure(pruneErr.Errs[i]) {
			exitCode = exitCodeFromK8sError(pruneErr.Errs[i])
		}
	}
	return pruneErr.Failed, exitCode, nil
}

// statusPruneFailed is the status of a stale resource whose delete failed.
const statusPruneFailed = "prune failed"

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

// namespaceToCreate reports whether --create-namespace has a namespace to
// create: createNS is set and the instance namespace is missing. It only
// reads. A dry run says here that the namespace would be created: the objects
// in it cannot be validated by the server.
func namespaceToCreate(ctx context.Context, k8sClient *kubernetes.Client, namespace string, createNS, dryRun bool, instanceLog *log.Logger) (bool, error) {
	if !createNS || namespace == "" {
		return false, nil
	}

	// The dry-run form of EnsureNamespace is the read: it creates nothing.
	missing, err := k8sClient.EnsureNamespace(ctx, namespace, true)
	if err != nil {
		instanceLog.Error("ensuring namespace", "error", err)
		return false, &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
	}
	if missing && dryRun {
		instanceLog.Info(fmt.Sprintf("namespace %q would be created", namespace))
	}
	return missing, nil
}

// ensureNamespace creates the instance namespace when create is set, which
// the caller derives from namespaceToCreate on a real run. The checks before
// it looked at nothing inside the namespace because it was missing, so a
// namespace that appeared since that read stops the apply: it may hold a
// record or resources that nothing checked.
func ensureNamespace(ctx context.Context, k8sClient *kubernetes.Client, namespace string, create bool, instanceLog *log.Logger) error {
	if !create {
		return nil
	}
	created, err := k8sClient.EnsureNamespace(ctx, namespace, false)
	if err != nil {
		instanceLog.Error("ensuring namespace", "error", err)
		return &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
	}
	if !created {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf(
			"namespace %q was created by someone else while apply was checking the cluster, and apply has not looked inside it\n"+
				"apply stopped before any change. Run the command again", namespace)}
	}
	instanceLog.Info(fmt.Sprintf("namespace %q created", namespace))
	return nil
}

// entriesOutside is the entries that are not in namespace, in order.
func entriesOutside(entries []k8sinventory.Entry, namespace string) []k8sinventory.Entry {
	out := make([]k8sinventory.Entry, 0, len(entries))
	for _, e := range entries {
		if e.Namespace != namespace {
			out = append(out, e)
		}
	}
	return out
}

// LoadPreviousInventory reads the ModuleInstance CR for an instance. It
// returns no record on a missing instance ID or when no CR exists, which is a
// first apply. The CR is the only inventory: an inventory Secret that an opm
// release before v1.0.0-alpha.2 wrote is never read.
//
// A CR read that fails with anything but NotFound is returned as the error:
// only a NotFound answer proves there is no record, and a caller that went on
// without one would run a first install over an existing instance.
func LoadPreviousInventory(ctx context.Context, k8sClient *kubernetes.Client, name, namespace, instanceID string) (*inventory.Record, error) {
	if instanceID == "" {
		return nil, nil
	}
	return inventory.GetRecord(ctx, k8sClient, name, namespace)
}

// unreadableRecordError is the refusal for a ModuleInstance read that failed
// with anything but NotFound. The exit code follows the cause: permission
// denied, connectivity, or general. nothingChanged adds that the apply
// stopped before any change: every call site is ahead of the apply's first
// write, the namespace create included, so it is false only when the caller
// wrote before the apply (Options.AfterCallerWrites).
func unreadableRecordError(name, namespace string, cause error, nothingChanged bool) error {
	stopped := "apply stopped"
	if nothingChanged {
		stopped += " before any change"
	}
	return &opmexit.ExitError{
		Code: exitCodeFromK8sError(cause),
		Err: fmt.Errorf("cannot read the ModuleInstance record %q in namespace %q: %w\n"+
			"%s: without the record it cannot tell a first install from an existing instance.\n"+
			"Check that you can read moduleinstances.%s in that namespace, then run the command again",
			name, namespace, cause, stopped, inventory.GroupOpmodel),
	}
}

// WriteInstanceRecord writes the ModuleInstance CR spec, then its status subset
// on the status subresource.
func WriteInstanceRecord(ctx context.Context, req Request, prevRecord *inventory.Record, currentEntries []k8sinventory.Entry, manifestDigest string, instanceLog *log.Logger) error {
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

	revision := nextRevision(prevRecord)
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

func nextRevision(prevRecord *inventory.Record) int {
	prev := 0
	if prevRecord != nil {
		prev = prevRecord.Inventory.Revision
	}
	if prev < 0 {
		prev = 0
	}
	return prev + 1
}

func previousEntries(prevRecord *inventory.Record) []k8sinventory.Entry {
	if prevRecord == nil {
		return nil
	}
	return prevRecord.Inventory.Entries
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

// RunPreApplyExistenceCheck runs the first-install existence check: never on
// a dry run and never when a previous inventory exists. It returns the
// rendered entries that already exist under OPM management
// (inventory.FirstInstallCheck). nothingChanged makes a refusal say that the
// apply stopped before any change.
func RunPreApplyExistenceCheck(ctx context.Context, k8sClient *kubernetes.Client, hasPrevInventory, dryRun bool, currentEntries []k8sinventory.Entry, admit inventory.AdmitSet, nothingChanged bool) ([]k8sinventory.Entry, error) {
	if hasPrevInventory || dryRun {
		return nil, nil
	}
	managed, err := inventory.FirstInstallCheck(ctx, k8sClient, currentEntries, admit)
	if err != nil {
		// An object the check could not read carries the API error, so the
		// exit code follows it; an untracked or terminating object maps to
		// the general code.
		stopped := ""
		if nothingChanged {
			stopped = "\napply stopped before any change"
		}
		return nil, &opmexit.ExitError{
			Code: exitCodeFromK8sError(err),
			Err:  fmt.Errorf("pre-apply existence check failed: %w%s", err, stopped),
		}
	}
	return managed, nil
}

// previewAlreadyManaged is the read-only half of the first-install check for
// a dry run: the rendered entries that already exist under OPM management. A
// dry run refuses nothing here, so an object the check would refuse on a real
// run only ends the look.
func previewAlreadyManaged(ctx context.Context, k8sClient *kubernetes.Client, currentEntries []k8sinventory.Entry) []k8sinventory.Entry {
	managed, err := inventory.FirstInstallCheck(ctx, k8sClient, currentEntries, nil)
	if err != nil {
		output.Debug("first-install preview stopped", "error", err)
		return nil
	}
	return managed
}

// lastMigratingRelease is the last opm release that moved an inventory kept
// in a Secret into the ModuleInstance record.
const lastMigratingRelease = "v1.0.0-beta.10"

// unrecordedResourcesWarning is the warning of a first install that found
// existing of its rendered resources already in the cluster under OPM
// management. No record lists them, so the apply cannot know what else an
// earlier apply created.
//
// The two runs give different advice. A dry run has written nothing, so the
// instance can still be applied with the release that migrates. A real run
// writes the record next, and that release then deletes the Secret without
// reading it: the Secret is the only list of the old inventory, so the text
// says to keep it and names it.
func unrecordedResourcesWarning(existing, rendered int, instanceName, instanceID string, dryRun bool) string {
	head := fmt.Sprintf("%d of %d rendered resource(s) already exist and are managed by OPM, but the instance has no ModuleInstance record. ", existing, rendered)
	if dryRun {
		return head + fmt.Sprintf("A real apply would update them in place and record them; it would prune nothing, so a resource an earlier apply created and this render no longer produces would stay in the cluster untracked. "+
			"If opm v1.0.0-alpha.1 or older last applied this instance, its inventory is in a Secret this release does not read: apply the instance once with opm %s before you apply it with this release",
			lastMigratingRelease)
	}
	return head + fmt.Sprintf("This apply updates them in place and records them; it prunes nothing, so a resource an earlier apply created and this render no longer produces stays in the cluster untracked. "+
		"If opm v1.0.0-alpha.1 or older last applied this instance, the Secret %q in this namespace still lists what it owned: keep it, do not apply this instance with an older opm, and remove the leftovers as the opm docs page \"Legacy inventory Secret\" says",
		"opm."+instanceName+"."+instanceID)
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
