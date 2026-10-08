package instance

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/charmbracelet/log"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/operator"
	"github.com/open-platform-model/cli/internal/output"
	workflowapply "github.com/open-platform-model/cli/internal/workflow/apply"
	"github.com/open-platform-model/cli/internal/workflow/query"
)

// NewInstanceDeleteCmd creates the instance delete command.
func NewInstanceDeleteCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		yesFlag     bool
		forceFlag   bool
		dryRunFlag  bool
		deleteData  bool
		timeoutFlag time.Duration
	)

	c := &cobra.Command{
		Use:   "delete <file|name|uuid>",
		Short: "Delete instance resources from cluster",
		Long: `Delete the resources belonging to an OPM instance from a Kubernetes cluster
(CRDs and Namespaces are left behind, PersistentVolumeClaims are kept).

PersistentVolumeClaims are kept by default, because deleting one deletes the
data on its volume. Each kept claim is listed with the status "kept", the
command still exits 0, and the closing output prints the 'kubectl delete pvc'
command for each one. The ModuleInstance is deleted, so OPM no longer tracks a
kept claim and no later opm command deletes it; applying the instance again
takes it back. Pass --delete-data to delete the claims and their data with the
instance; the confirmation prompt then names each claim. Claims a StatefulSet
created from volumeClaimTemplates are not tracked by OPM and are never deleted
here, with or without the flag: Kubernetes keeps them by default.

None of this holds for an operator-managed instance: there the operator
deletes what the instance tracks when spec.prune is set, and leaves all of it
running otherwise. An operator that has spec.dataPolicy keeps
PersistentVolumeClaims unless that field of the instance is Delete; an
operator without the field deletes them. The confirmation prompt says which
holds: it reads spec.prune and spec.dataPolicy from the instance, and from the
ModuleInstance CRD whether the operator has the field. --delete-data does not
change what the operator does.

CustomResourceDefinitions and Namespaces are never deleted, since deleting one
takes every custom resource of its kind, or everything inside it, with it.
Each tracked resource is also read again just before its delete, and a
resource that is no longer managed by OPM, now belongs to another instance, or
carries an opmodel.dev/adopt annotation naming another instance is left
behind. Every resource left behind is listed with its reason; remove
it with 'kubectl delete' once nothing else needs it.

Deleting an instance that deploys the operator (the instance opm-operator in
opm-operator-system, any instance of the operator module, or one whose
inventory holds the operator's CRDs) is refused in two cases, dry runs
included. While it is operator-owned: the operator never reconciles or prunes
the instance that deploys it, so set spec.owner to cli first. And while any
instance still carries the operator's opmodel.dev/cleanup finalizer: removing
the operator would leave them unable to finish deletion. Run
'opm operator uninstall --remove-finalizers' to remove that finalizer first,
which orphans those instances' workloads. --yes does not bypass either.

Arguments:
  file         Path to an instance.cue file or directory containing one.
               The instance name and namespace are read from the file's metadata.
               --namespace overrides the namespace found in the file.
  name         Instance name (use -n / --namespace to scope by namespace).
  uuid         Instance UUID.

Examples:
  # Delete by instance.cue file in the current directory
  opm instance delete .

  # Delete by instance.cue file path
  opm instance delete ./instances/jellyfin/instance.cue -n media

  # Delete by name
  opm instance delete jellyfin -n media

  # Preview what would be deleted
  opm instance delete jellyfin -n media --dry-run

  # Skip the confirmation prompt
  opm instance delete jellyfin -n media --yes

  # Also delete the PersistentVolumeClaims and the data on them
  opm instance delete jellyfin -n media --delete-data`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runDelete(c.Context(), args[0], cfg, &kf, namespace, deleteFlags{
				SkipConfirm: yesFlag || forceFlag,
				DryRun:      dryRunFlag,
				DeleteData:  deleteData,
				Timeout:     timeoutFlag,
			})
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip the confirmation prompt")
	// --force was the first spelling of --yes. On every other command --force
	// overrides a refusal, so here it stays only as a deprecated alias.
	c.Flags().BoolVar(&forceFlag, "force", false, "Skip the confirmation prompt")
	cmdutil.DeprecateFlag(c, "force", "yes")
	c.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Preview without deleting")
	c.Flags().BoolVar(&deleteData, "delete-data", false, deleteDataFlagHelp)
	c.Flags().DurationVar(&timeoutFlag, "timeout", inventory.DefaultReconcileTimeout,
		"Bound on the operator-cleanup wait (operator-managed instances only)")

	return c
}

// runDelete is what the delete command runs. It is a variable so a test can
// read what the flags resolved to without a cluster.
var runDelete = runInstanceDelete

// deleteDataFlagHelp is the help of --delete-data on instance delete.
const deleteDataFlagHelp = "Also delete PersistentVolumeClaims and the data on them (kept by default)"

// deleteFlags carries the delete command's behavior flags.
type deleteFlags struct {
	// SkipConfirm is --yes (or its deprecated alias): do not prompt.
	SkipConfirm bool
	DryRun      bool
	// DeleteData is --delete-data: delete tracked PersistentVolumeClaims too.
	DeleteData bool
	Timeout    time.Duration
}

func runInstanceDelete(ctx context.Context, identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, flags deleteFlags) error {
	target, err := cmdutil.ResolveInstanceTarget(ctx, identifier, cfg, kf, namespaceFlag)
	if err != nil {
		return err
	}
	cmdutil.LogResolvedKubernetesConfig(target.Namespace, target.K8sConfig.Kubeconfig.Value, target.K8sConfig.Context.Value)

	rsf := target.Selector
	namespace := target.Namespace
	instanceLog := output.InstanceLogger(target.LogName)

	k8sClient, err := cmdutil.NewK8sClient(target.K8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		instanceLog.Error("connecting to cluster", "error", err)
		return err
	}

	return confirmAndDelete(ctx, k8sClient, rsf, namespace, flags, os.Stdin, instanceLog)
}

// confirmAndDelete reads the instance record, asks for confirmation on in
// unless the flags skip it, and deletes. The record is read before the
// prompt, so that the prompt can name the claims --delete-data deletes and a
// missing instance is reported without a question. The read changes nothing.
func confirmAndDelete(ctx context.Context, k8sClient *kubernetes.Client, rsf *cmdutil.InstanceSelectorFlags, namespace string, flags deleteFlags, in io.Reader, instanceLog *log.Logger) error {
	inv, liveResources, _, unreadable, err := query.ResolveInventory(ctx, k8sClient, rsf, namespace, instanceLog)
	if err != nil {
		return err
	}

	// Said before the question: on an operator-managed instance the flag
	// does not change what the operator does, and the user must know that
	// when they answer.
	operatorManaged := inventory.ResolveOwnership(inv) == inventory.ModeOperatorOwned
	if operatorManaged && flags.DeleteData {
		instanceLog.Warn(workflowapply.DeleteDataOperatorManagedNote)
	}

	// Read before the question too: what the operator does with the claims
	// depends on whether its CRD has spec.dataPolicy at all.
	opClaims := readOperatorClaims(ctx, k8sClient, inv)

	if flags.DryRun {
		instanceLog.Info("dry run - no changes will be made")
	} else if !flags.SkipConfirm {
		prompt := deletePrompt(rsf.InstanceName, rsf.InstanceID, namespace, claimsToDelete(inv, liveResources, flags.DeleteData))
		if operatorManaged {
			prompt = operatorManagedDeletePrompt(rsf.InstanceName, rsf.InstanceID, namespace, inv.Prune, opClaims)
		}
		output.Prompt(prompt)
		if !readConfirmation(in) {
			instanceLog.Info("deletion canceled")
			return nil
		}
	}

	return deleteResolvedInstance(ctx, k8sClient, rsf, namespace, inv, liveResources, unreadable, opClaims, flags.Timeout, flags.DryRun, flags.DeleteData, instanceLog)
}

// claimsToDelete lists, as "<namespace>/<name>", the PersistentVolumeClaims a
// delete with these flags removes: the live tracked claims of a CLI-owned
// instance when deleteData is set, and none otherwise. An operator-managed
// instance lists none, since the operator decides what it removes.
func claimsToDelete(inv *inventory.Record, live []*unstructured.Unstructured, deleteData bool) []string {
	if !deleteData || inventory.ResolveOwnership(inv) == inventory.ModeOperatorOwned {
		return nil
	}
	var claims []string
	for _, obj := range live {
		if kubernetes.IsDataClaim(obj.GroupVersionKind().Group, obj.GetKind()) {
			claims = append(claims, obj.GetNamespace()+"/"+obj.GetName())
		}
	}
	return claims
}

// deleteResolvedInstance deletes an instance whose record has been read. It
// first guards an instance that deploys the operator, on both branches and on
// a dry run alike. Ownership is then the single branch point (0006:D18): an
// operator-owned instance is deleted by deleting its CR and letting the
// operator's finalizer prune the workloads.
//
// unreadable lists the tracked resources discovery could not read. The
// CLI-owned branch treats each as a per-resource failure and keeps the
// ModuleInstance; the operator-owned branch ignores them, since it deletes only
// the ModuleInstance and the operator prunes with its own credentials.
func deleteResolvedInstance(ctx context.Context, k8sClient *kubernetes.Client, rsf *cmdutil.InstanceSelectorFlags, namespace string,
	inv *inventory.Record, liveResources []*unstructured.Unstructured, unreadable []inventory.UnreadableEntry, opClaims operatorClaims,
	timeout time.Duration, dryRun, deleteData bool, instanceLog *log.Logger) error {
	if err := guardOperatorInstanceDelete(ctx, k8sClient, inv); err != nil {
		return err
	}

	if inventory.ResolveOwnership(inv) == inventory.ModeOperatorOwned {
		return deleteOperatorOwned(ctx, k8sClient, inv, opClaims, timeout, dryRun, instanceLog)
	}

	return executeInstanceDelete(ctx, k8sClient, rsf, namespace, inv, liveResources, unreadable, dryRun, deleteData, instanceLog)
}

// guardOperatorInstanceDelete refuses to delete an instance that deploys the
// operator (operator.DeploysOperator) in two cases; every other instance
// passes untouched.
//
// An operator-owned record is refused before any read: the operator never
// reconciles, finalizes or prunes the instance that deploys it, so the
// operator-owned branch would wait on, and report, a cleanup that does not
// happen. The remedy is to set spec.owner to cli; the CLI does not write it.
//
// Otherwise the delete is refused while any ModuleInstance carries the
// operator's cleanup finalizer: removing the operator then leaves each one
// unable to finish deletion until an operator runs again. Orphaning them is
// the explicit choice 'opm operator uninstall --remove-finalizers' already
// owns, so the refusal points there and this command offers no such flag. A
// failed list fails closed.
func guardOperatorInstanceDelete(ctx context.Context, k8sClient *kubernetes.Client, inv *inventory.Record) error {
	signal, ok := operator.DeploysOperator(inv)
	if !ok {
		return nil
	}

	if inventory.ResolveOwnership(inv) == inventory.ModeOperatorOwned {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: &operator.OwnInstanceOwnerError{
			Namespace: inv.Namespace, Name: inv.Name, Signal: signal,
		}}
	}

	armed, err := operator.CheckFinalizerGuard(ctx, k8sClient)
	if err != nil {
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err}
	}
	if len(armed) == 0 {
		return nil
	}
	return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: &operator.FinalizerGuardError{
		Armed:  armed,
		Action: fmt.Sprintf("delete %s/%s, which deploys the operator", inv.Namespace, inv.Name),
		Target: operator.ArmedInstance{Namespace: inv.Namespace, Name: inv.Name},
		Remedy: "run 'opm operator uninstall --remove-finalizers' to remove that finalizer, orphaning their workloads, then retry",
	}}
}

// deleteOperatorOwned deletes an operator-managed instance by removing its
// ModuleInstance CR and waiting for the operator's cleanup finalizer to prune
// the workloads.
//
// The readiness guard is the point of this function. A ModuleInstance carries
// the operator's cleanup finalizer, so deleting the CR with no controller
// running does not delete anything — it wedges the CR in Terminating forever,
// with its workloads orphaned and unreachable through the CLI. That is the same
// footgun `opm operator uninstall` guards from the other side, and it has no
// --force bypass here: forcing it produces the wedge, it does not avoid it.
func deleteOperatorOwned(ctx context.Context, k8sClient *kubernetes.Client, inv *inventory.Record, claims operatorClaims, timeout time.Duration, dryRun bool, instanceLog *log.Logger) error {
	if err := operator.CheckReady(ctx, k8sClient); err != nil {
		var notReady *operator.NotReadyError
		if errors.As(err, &notReady) {
			notReady.Hint = fmt.Sprintf(
				"instance %q is operator-managed, and deleting its ModuleInstance now would wedge it in Terminating on the %s finalizer with its workloads orphaned",
				inv.Name, inventory.CleanupFinalizer)
		}
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	// spec.prune decides whether removing the CR removes the workloads. It has
	// no CRD default, so it is false unless someone set it, and the operator
	// then orphans the workloads on purpose. The CLI does not write this field,
	// so a CLI-created instance orphans by default — say so rather than
	// reporting a cleanup that will not happen.
	//
	// spec.dataPolicy decides, under spec.prune, whether the operator's prune
	// takes the PersistentVolumeClaims too, for an operator that has the
	// field; claims carries what opm could read about that.
	entries := len(inv.Inventory.Entries)
	if dryRun {
		if inv.Prune {
			instanceLog.Info(fmt.Sprintf(
				"dry run complete: ModuleInstance %q would be deleted and the operator would prune its %d tracked resource(s)%s",
				inv.Name, entries, claims.outcome()))
		} else {
			instanceLog.Info(fmt.Sprintf(
				"dry run complete: ModuleInstance %q would be deleted; its %d tracked resource(s) would be left running (spec.prune is not set)",
				inv.Name, entries))
		}
		return nil
	}

	if inv.Prune {
		instanceLog.Info("deleting the ModuleInstance — the operator prunes its resources"+claims.outcome(), "instance", inv.Name)
	} else {
		instanceLog.Warn("spec.prune is not set — the operator will remove the ModuleInstance but leave its resources running",
			"instance", inv.Name, "resources", entries)
	}

	if err := inventory.DeleteCR(ctx, k8sClient, inv.Name, inv.Namespace); err != nil {
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err}
	}

	instanceLog.Info("waiting for the operator to finish cleanup")
	if err := inventory.WaitForAbsence(ctx, k8sClient, inv.Name, inv.Namespace, timeout); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}

	// The CR's disappearance proves the finalizer completed; it does not prove
	// anything was pruned. Report only what was actually established.
	if inv.Prune {
		reportOperatorPrune(inv, claims, instanceLog)
		return nil
	}

	output.Println(output.FormatCheckmark(fmt.Sprintf(
		"ModuleInstance deleted — %d resource(s) left running", entries)))
	output.Details(fmt.Sprintf(
		"The operator orphaned them because spec.prune is not set on this instance.\n"+
			"To have the operator remove workloads on delete, set it before deleting:\n"+
			"  kubectl patch moduleinstance %s -n %s --type=merge -p '{\"spec\":{\"prune\":true}}'\n"+
			"Otherwise remove them yourself with 'kubectl delete'.",
		inv.Name, inv.Namespace))
	return nil
}

// executeInstanceDelete deletes the instance's tracked workloads, then the
// ModuleInstance CR last (after all workloads are gone; skipped on dry-run).
// A tracked resource discovery could not read (unreadable) is a per-resource
// failure, so the ModuleInstance is kept and still tracks it. A ModuleInstance
// delete that fails after the workloads are gone fails the command with the
// exit code of its cause. A PersistentVolumeClaim is kept unless deleteData;
// a kept claim does not block the ModuleInstance delete, so it is left
// untracked, where no later opm command can delete it.
func executeInstanceDelete(ctx context.Context, k8sClient *kubernetes.Client, rsf *cmdutil.InstanceSelectorFlags, namespace string, inv *inventory.Record, liveResources []*unstructured.Unstructured, unreadable []inventory.UnreadableEntry, dryRun, deleteData bool, instanceLog *log.Logger) error {
	deleteResult, err := workflowapply.DeleteRecorded(ctx, workflowapply.DeleteRequest{
		Client:       k8sClient,
		InstanceName: rsf.InstanceName,
		InstanceID:   rsf.InstanceID,
		Namespace:    namespace,
		Record:       inv,
		Live:         liveResources,
		Unreadable:   inventory.UnreadableResources(unreadable),
		DryRun:       dryRun,
		DeleteData:   deleteData,
		Log:          instanceLog,
	})
	if err != nil {
		var recordErr *workflowapply.RecordDeleteError
		if errors.As(err, &recordErr) {
			instanceLog.Error(fmt.Sprintf("the tracked resources of ModuleInstance %s/%s were deleted, but the record remains",
				recordErr.Namespace, recordErr.Name), "error", recordErr.Err)
			output.Details("The ModuleInstance still lists resources that are gone.\n" +
				"Fix the cause (for example missing RBAC) and re-run; re-running is safe.")
		}
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}
	return reportInstanceDelete(deleteResult, dryRun, instanceLog)
}

// reportInstanceDelete prints the closing summary of a CLI-owned delete and
// turns per-resource errors into the command's exit error. Errors claim no
// completion: a real run kept the ModuleInstance for a re-run, and says so,
// and a dry run, which attempts no delete, could not check every resource.
func reportInstanceDelete(deleteResult *kubernetes.DeleteResult, dryRun bool, instanceLog *log.Logger) error {
	if n := len(deleteResult.Errors); n > 0 {
		format := "%d resource(s) failed to delete"
		if dryRun {
			format = "%d resource(s) could not be checked"
		} else {
			output.Details("The ModuleInstance was kept, so it still tracks these resources.\n" +
				"Fix the cause (for example missing RBAC) and re-run; re-running is safe.")
		}
		hintUnservedKinds(deleteResult)
		return &opmexit.ExitError{Code: deleteFailureExitCode(deleteResult), Err: fmt.Errorf(format, n), Printed: true}
	}

	leftBehind := len(deleteResult.LeftBehind)
	switch {
	case dryRun && leftBehind > 0:
		instanceLog.Info(fmt.Sprintf("dry run complete: %d resources would be deleted, %d left behind", deleteResult.Deleted, leftBehind))
	case dryRun:
		instanceLog.Info(fmt.Sprintf("dry run complete: %d resources would be deleted", deleteResult.Deleted))
	case leftBehind > 0:
		output.Println(output.FormatCheckmark(fmt.Sprintf("Instance deleted — %d resource(s) left behind", leftBehind)))
		output.Details("Remove them with 'kubectl delete' once nothing else needs them.")
	default:
		if len(deleteResult.Kept) == 0 {
			instanceLog.Info("all resources have been deleted")
		}
		output.Println(output.FormatCheckmark("Instance deleted"))
	}
	reportKeptClaims(deleteResult.Kept, dryRun, instanceLog)
	return nil
}

// reportKeptClaims closes a delete that kept PersistentVolumeClaims: how many,
// that OPM no longer tracks them, and the command that deletes each one. The
// claims are kept on purpose, so nothing here is a warning. A dry run says
// what a real run would keep.
func reportKeptClaims(kept []kubernetes.LeftBehindResource, dryRun bool, instanceLog *log.Logger) {
	if len(kept) == 0 {
		return
	}
	if dryRun {
		instanceLog.Info(fmt.Sprintf("dry run: %d PersistentVolumeClaim(s) would be kept with the data on them; pass --delete-data to delete them",
			len(kept)))
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Kept %d PersistentVolumeClaim(s) and the data on them. OPM no longer tracks them.\n", len(kept))
	b.WriteString("Applying the instance again takes them back. To delete a claim and its data:\n")
	for _, k := range kept {
		fmt.Fprintf(&b, "  kubectl delete pvc %s -n %s\n", k.Name, k.Namespace)
	}
	b.WriteString("To delete claims together with an instance, pass --delete-data.")
	output.Details(b.String())
}

// deletePrompt is the confirmation question of instance delete. claims are
// the PersistentVolumeClaims this run deletes ("<namespace>/<name>"); when
// there are any, the prompt names each one before the question.
func deletePrompt(instanceName, instanceID, namespace string, claims []string) string {
	var b strings.Builder
	if len(claims) > 0 {
		b.WriteString("--delete-data: these PersistentVolumeClaims and the data on them will be deleted:\n")
		for _, c := range claims {
			b.WriteString("  " + c + "\n")
		}
	}
	subject := fmt.Sprintf("instance %q", instanceName)
	if instanceName == "" {
		subject = fmt.Sprintf("instance-id %q", instanceID)
	}
	left := "CRDs and Namespaces are left behind, PersistentVolumeClaims are kept"
	if len(claims) > 0 {
		left = "CRDs and Namespaces are left behind"
	}
	fmt.Fprintf(&b, "Delete the resources for %s in namespace %q (%s)? [y/N]: ", subject, namespace, left)
	return b.String()
}

// reportOperatorPrune closes an operator-owned delete with spec.prune set. The
// ModuleInstance being gone proves that the operator's finalizer completed,
// not which resources are left. So when the instance tracked
// PersistentVolumeClaims that the operator may have kept, the closing output
// does not say that everything was pruned, and does not say that the claims
// are still there either: it names them, says which operator keeps them, and
// prints the commands that show and delete them.
func reportOperatorPrune(inv *inventory.Record, claims operatorClaims, instanceLog *log.Logger) {
	if fate := claims.fate(); fate == claimsNone || fate == claimsDeleted {
		instanceLog.Info("all resources have been deleted")
		output.Println(output.FormatCheckmark(fmt.Sprintf("Instance deleted — operator pruned %d resources", len(inv.Inventory.Entries))))
		return
	}

	output.Println(output.FormatCheckmark("Instance deleted: the operator finished its cleanup"))
	var b strings.Builder
	fmt.Fprintf(&b, "The instance tracked %d PersistentVolumeClaim(s). An operator that has spec.dataPolicy keeps them\n", len(claims.tracked))
	fmt.Fprintf(&b, "and the data on them (%s), and OPM no longer tracks them.\n", describeDataPolicy(claims.dataPolicy))
	b.WriteString("An operator older than its CRDs deleted them. To see what is left:\n")
	fmt.Fprintf(&b, "  kubectl get pvc -n %s\n", inv.Namespace)
	b.WriteString("To delete a claim and its data:\n")
	for _, c := range claims.tracked {
		fmt.Fprintf(&b, "  kubectl delete pvc %s -n %s\n", c.Name, c.Namespace)
	}
	b.WriteString("To have the operator delete claims with an instance, set spec.dataPolicy to Delete before deleting it.")
	output.Details(b.String())
}

// dataPolicyDelete is the one value of a ModuleInstance's spec.dataPolicy
// that lets the operator delete PersistentVolumeClaims; dataPolicyKeep is the
// other value the operator defines. The CLI reads the field as a string and
// does not import the operator's types.
const (
	dataPolicyField  = "dataPolicy"
	dataPolicyDelete = "Delete"
	dataPolicyKeep   = "Keep"
)

// keptClaimsHedge follows every statement that the operator keeps claims. The
// CRD shows that the operator's API has spec.dataPolicy; it does not show
// that the controller is as new as its CRDs.
const keptClaimsHedge = "The ModuleInstance CRD has spec.dataPolicy, but an operator older than its CRDs deletes the claims whatever the field says."

// undecidedClaimsNote stands in for a statement about claims when the CRD
// could not be read.
const undecidedClaimsNote = "An operator that has spec.dataPolicy keeps PersistentVolumeClaims and the data on them, an older operator deletes them, " +
	"and opm could not read the ModuleInstance CRD to tell which runs here."

// claimFate is what the operator does with the PersistentVolumeClaims of an
// instance with spec.prune set, as far as opm can read it.
type claimFate int

const (
	// claimsNone: the inventory tracks no claim.
	claimsNone claimFate = iota
	// claimsDeleted: spec.dataPolicy is Delete, or the operator's CRD has no
	// such field, which is the operator that deletes claims under spec.prune.
	claimsDeleted
	// claimsKept: the CRD has the field and its value is not Delete.
	claimsKept
	// claimsUndecided: the value is not Delete and the CRD could not be read.
	claimsUndecided
)

// operatorClaims is what an operator-owned delete knows about the instance's
// PersistentVolumeClaims: the ones its recorded inventory tracks, the
// spec.dataPolicy as written, and whether the installed ModuleInstance CRD
// has that field. The zero value tracks no claim.
type operatorClaims struct {
	tracked    []k8sinventory.Entry
	dataPolicy string
	field      operator.FieldSupport
}

// readOperatorClaims builds the operatorClaims of an operator-managed
// instance. It reads the ModuleInstance CRD, which the readiness gate of the
// delete reads too, only when the answer matters: spec.prune is set and the
// inventory tracks a claim. Every other instance costs no read.
func readOperatorClaims(ctx context.Context, k8sClient *kubernetes.Client, inv *inventory.Record) operatorClaims {
	if inventory.ResolveOwnership(inv) != inventory.ModeOperatorOwned || !inv.Prune {
		return operatorClaims{}
	}
	claims := operatorClaims{tracked: trackedClaims(inv), dataPolicy: inv.DataPolicy}
	if len(claims.tracked) > 0 {
		claims.field = operator.ModuleInstanceSpecField(ctx, k8sClient, dataPolicyField)
	}
	return claims
}

// fate decides what the operator does with the tracked claims. Only the
// exact value Delete is a request to delete: Keep, an absent value and any
// value opm does not know keep them, which is the operator's own rule. An
// operator whose CRD has no spec.dataPolicy keeps nothing.
func (c operatorClaims) fate() claimFate {
	switch {
	case len(c.tracked) == 0:
		return claimsNone
	case c.field == operator.FieldAbsent, c.dataPolicy == dataPolicyDelete:
		return claimsDeleted
	case c.field == operator.FieldPresent:
		return claimsKept
	default:
		return claimsUndecided
	}
}

// reason says why the claims have their fate: the operator has no
// spec.dataPolicy, or the value of the field as it is written.
func (c operatorClaims) reason() string {
	if c.field == operator.FieldAbsent {
		return "the operator in this cluster has no spec.dataPolicy"
	}
	return describeDataPolicy(c.dataPolicy)
}

// outcome is the clause a progress or dry-run line of an instance with
// spec.prune set appends about the claims; empty when it tracks none.
func (c operatorClaims) outcome() string {
	switch c.fate() {
	case claimsDeleted:
		return fmt.Sprintf(", PersistentVolumeClaims and the data on them included (%s)", c.reason())
	case claimsKept:
		return fmt.Sprintf(", PersistentVolumeClaims and the data on them kept (%s) unless the operator is older than its CRDs", c.reason())
	case claimsUndecided:
		return "; PersistentVolumeClaims are kept only by an operator that has spec.dataPolicy, and opm could not read the ModuleInstance CRD to tell"
	case claimsNone:
		return ""
	}
	return ""
}

// describeDataPolicy words a spec.dataPolicy value for a message, as it is
// written on the instance. A value opm does not know is quoted, which also
// makes any control character in it harmless, and is named as read as Keep.
func describeDataPolicy(dataPolicy string) string {
	switch dataPolicy {
	case "":
		return "spec.dataPolicy is not set"
	case dataPolicyKeep, dataPolicyDelete:
		return "spec.dataPolicy is " + dataPolicy
	default:
		return fmt.Sprintf("spec.dataPolicy is %q, not a value opm knows, read as %s", dataPolicy, dataPolicyKeep)
	}
}

// trackedClaims lists the PersistentVolumeClaims in the instance's recorded
// inventory.
func trackedClaims(inv *inventory.Record) []k8sinventory.Entry {
	var claims []k8sinventory.Entry
	for _, e := range inv.Inventory.Entries {
		if kubernetes.IsDataClaim(e.Group, e.Kind) {
			claims = append(claims, e)
		}
	}
	return claims
}

// operatorManagedDeletePrompt is the confirmation question for an
// operator-managed instance. opm deletes only the ModuleInstance there; the
// operator removes what the instance tracks when spec.prune is set, and
// leaves all of it running otherwise. When the instance tracks
// PersistentVolumeClaims the prompt says what the operator does with them,
// and it says that they are kept only when the installed CRD has
// spec.dataPolicy, with what an operator older than its CRDs does.
func operatorManagedDeletePrompt(instanceName, instanceID, namespace string, prune bool, claims operatorClaims) string {
	subject := fmt.Sprintf("instance %q", instanceName)
	if instanceName == "" {
		subject = fmt.Sprintf("instance-id %q", instanceID)
	}
	question := fmt.Sprintf("Delete the ModuleInstance for %s in namespace %q? [y/N]: ", subject, namespace)
	if !prune {
		return "This instance is operator-managed: spec.prune is not set, so the operator leaves its tracked resources running.\n" + question
	}

	const lead = "This instance is operator-managed: spec.prune is set"
	switch claims.fate() {
	case claimsDeleted:
		return fmt.Sprintf("%s and %s, so the operator deletes its tracked resources, PersistentVolumeClaims and the data on them included.\n",
			lead, claims.reason()) + question
	case claimsKept:
		return fmt.Sprintf("%s and %s, so the operator deletes its tracked resources and keeps PersistentVolumeClaims and the data on them.\n",
			lead, claims.reason()) + keptClaimsHedge + "\n" + question
	case claimsUndecided:
		return fmt.Sprintf("%s and %s, so the operator deletes its tracked resources.\n", lead, claims.reason()) +
			undecidedClaimsNote + "\n" + question
	case claimsNone:
	}
	return lead + ", so the operator deletes its tracked resources.\n" + question
}

// readConfirmation reads one line and reports whether it says yes. Anything
// else, a closed input included, is a no.
func readConfirmation(in io.Reader) bool {
	scanner := bufio.NewScanner(in)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}
