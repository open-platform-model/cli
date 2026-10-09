package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/library/opm/k8s/object"
)

// ApplyOptions configures an apply operation.
type ApplyOptions struct {
	// DryRun performs a server-side dry run without persisting changes.
	DryRun bool

	// EstablishDeadline bounds Apply's wait for the first stage's
	// CustomResourceDefinitions to report Established=True. Zero means
	// defaultEstablishTimeout from the start of the wait. ApplyOne ignores it.
	EstablishDeadline time.Time

	// BudgetStart is when the budget behind EstablishDeadline started (the
	// start of the apply), so a timeout reports the time spent against that
	// budget rather than only the wait's share. Zero means the start of the
	// wait. ApplyOne ignores it.
	BudgetStart time.Time

	// NewNamespaces names namespaces a dry run treats as created by this
	// apply although no Namespace object of it says so: the instance
	// namespace that --create-namespace would create. Apply adds every
	// Namespace of its first stage whose pre-apply read returned NotFound.
	// Ignored outside a dry run, and by ApplyOne.
	NewNamespaces []string
}

// defaultEstablishTimeout bounds the CustomResourceDefinition wait when the
// caller sets no deadline; it matches the apply commands' --timeout default.
const defaultEstablishTimeout = 5 * time.Minute

// ApplyResult contains the outcome of an apply operation.
type ApplyResult struct {
	// Applied is the number of resources successfully applied.
	Applied int

	// Created is the number of resources that were newly created.
	Created int

	// Configured is the number of resources that were modified.
	Configured int

	// Unchanged is the number of resources that had no changes.
	Unchanged int

	// Skipped counts the objects a dry run did not send because the same
	// apply would create their CustomResourceDefinition or their namespace.
	Skipped int

	// Errors contains per-resource errors (non-fatal).
	Errors []resourceError
}

// resourceError captures an error for a specific resource.
type resourceError struct {
	// Resource identifies the resource.
	Kind      string
	Name      string
	Namespace string

	// Err is the error.
	Err error
}

func (e *resourceError) Error() string {
	if e.Namespace != "" {
		return fmt.Sprintf("%s/%s in %s: %v", e.Kind, e.Name, e.Namespace, e.Err)
	}
	return fmt.Sprintf("%s/%s: %v", e.Kind, e.Name, e.Err)
}

// stageOutcome is one object a stage applied without error. absentBefore
// records that the pre-apply read returned NotFound: the apply creates it.
type stageOutcome struct {
	obj          *unstructured.Unstructured
	absentBefore bool
}

// Apply performs server-side apply for a set of rendered resources, in two
// stages. It sorts a copy of resources ascending by resource weight (stable,
// so the input order breaks ties; the caller's slice is not reordered). The
// first stage applies the CustomResourceDefinitions and Namespaces; outside a
// dry run, Apply then waits until every CustomResourceDefinition of that stage
// that applied reports Established=True, bounded by opts.EstablishDeadline,
// and returns an error without applying the second stage if the wait fails.
// The second stage applies everything else. A dry run waits for nothing, and
// does not send a custom resource whose CustomResourceDefinition the same
// apply creates, nor a namespaced object whose namespace the same apply
// creates (a Namespace of the first stage that did not exist, or one named in
// opts.NewNamespaces): the server cannot validate it yet, so it is logged as
// skipped and counted in Skipped. A resource that fails to apply is logged and
// recorded in Errors, and the remaining resources are still applied.
// instanceName is used for logging only.
func Apply(ctx context.Context, client *Client, resources []*unstructured.Unstructured, instanceName string, opts ApplyOptions) (*ApplyResult, error) {
	result := &ApplyResult{}
	instanceLog := output.InstanceLogger(instanceName)

	sorted := append([]*unstructured.Unstructured(nil), resources...)
	SortObjects(sorted, object.Ascending)
	definitions, rest := splitClusterDefinitions(sorted)

	applied, err := applyStage(ctx, client, definitions, opts, dryRunSkips{}, result, instanceLog)
	if err != nil {
		return result, err
	}

	var skips dryRunSkips
	if opts.DryRun {
		skips = dryRunSkips{kinds: kindsOfNewCRDs(applied), namespaces: newNamespaces(applied, opts.NewNamespaces)}
	} else if err := waitEstablished(ctx, client, applied, rest, opts.EstablishDeadline, opts.BudgetStart, instanceLog); err != nil {
		return result, err
	}

	if _, err := applyStage(ctx, client, rest, opts, skips, result, instanceLog); err != nil {
		return result, err
	}
	return result, nil
}

// isCRD reports whether gvk is a CustomResourceDefinition.
func isCRD(gvk schema.GroupVersionKind) bool {
	return gvk.Group == groupAPIExtensions && gvk.Kind == kindCustomResourceDefinition
}

// splitClusterDefinitions partitions objs, keeping their order, into the
// cluster definitions and everything else. The cluster definitions are the
// protected kinds (IsProtectedKind): a CustomResourceDefinition, which its
// custom resources cannot be applied without, and a Namespace, which the
// objects in it cannot.
func splitClusterDefinitions(objs []*unstructured.Unstructured) (definitions, rest []*unstructured.Unstructured) {
	for _, obj := range objs {
		gvk := obj.GroupVersionKind()
		if IsProtectedKind(gvk.Group, gvk.Kind) {
			definitions = append(definitions, obj)
		} else {
			rest = append(rest, obj)
		}
	}
	return definitions, rest
}

// dryRunSkips is what a dry run's second stage does not send. kinds maps
// each group and kind to the name of the CustomResourceDefinition this apply
// creates for it; namespaces holds the namespaces this apply creates.
type dryRunSkips struct {
	kinds      map[schema.GroupKind]string
	namespaces map[string]struct{}
}

// reason returns the warning for an object the dry run does not send, or ""
// when it is sent. A custom resource with both reasons is reported once, for
// its CustomResourceDefinition.
func (s dryRunSkips) reason(res *unstructured.Unstructured) string {
	if crdName, ok := s.kinds[res.GroupVersionKind().GroupKind()]; ok {
		return fmt.Sprintf("skipping %s: its CustomResourceDefinition %s is created by this apply, so a dry run cannot validate it",
			describeObjects([]*unstructured.Unstructured{res}), crdName)
	}
	if ns := res.GetNamespace(); ns != "" {
		if _, ok := s.namespaces[ns]; ok {
			return fmt.Sprintf("skipping %s/%s in %s: namespace %s is created by this apply, so a dry run cannot validate it",
				res.GetKind(), res.GetName(), ns, ns)
		}
	}
	return ""
}

// applyStage applies objs in order, logging one line per object and
// recording counts and per-resource errors in result. An object skips names
// is not sent: it is logged as skipped and counted. It returns the objects
// applied without error. A failed API discovery request is not a
// per-resource error: it stops the stage and is returned, since the cluster
// cannot say where the remaining objects go.
func applyStage(ctx context.Context, client *Client, objs []*unstructured.Unstructured, opts ApplyOptions, skips dryRunSkips, result *ApplyResult, instanceLog *log.Logger) ([]stageOutcome, error) {
	var applied []stageOutcome
	for _, res := range objs {
		kind := res.GetKind()
		name := res.GetName()
		ns := res.GetNamespace()

		if reason := skips.reason(res); reason != "" {
			instanceLog.Warn(reason)
			result.Skipped++
			continue
		}

		status, absentBefore, err := applyOne(ctx, client, res, opts)
		if IsDiscoveryFailure(err) {
			return applied, fmt.Errorf("applying %s/%s: %w", kind, name, err)
		}
		if err != nil {
			instanceLog.Warn(fmt.Sprintf("applying %s/%s: %v", kind, name, err))
			result.Errors = append(result.Errors, resourceError{
				Kind:      kind,
				Name:      name,
				Namespace: ns,
				Err:       err,
			})
			continue
		}

		result.Applied++
		switch status {
		case output.StatusCreated:
			result.Created++
		case output.StatusConfigured:
			result.Configured++
		case output.StatusUnchanged:
			result.Unchanged++
		}
		instanceLog.Info(output.FormatResourceLine(kind, ns, name, status))
		applied = append(applied, stageOutcome{obj: res, absentBefore: absentBefore})
	}
	return applied, nil
}

// waitEstablished waits until every CustomResourceDefinition among applied
// reports Established=True, and then until API discovery serves each kind of
// rest that one of them defines, or deadline passes (zero:
// defaultEstablishTimeout from now). since is when the budget behind deadline
// started (zero: now); a timeout reports the time elapsed from it.
func waitEstablished(ctx context.Context, client *Client, applied []stageOutcome, rest []*unstructured.Unstructured, deadline, since time.Time, instanceLog *log.Logger) error {
	var crds []*unstructured.Unstructured
	for _, o := range applied {
		if isCRD(o.obj.GroupVersionKind()) {
			crds = append(crds, o.obj)
		}
	}
	if len(crds) == 0 {
		return nil
	}

	start := time.Now()
	if deadline.IsZero() {
		deadline = start.Add(defaultEstablishTimeout)
	}
	if since.IsZero() {
		since = start
	}
	instanceLog.Info(fmt.Sprintf("waiting for %d CustomResourceDefinition(s) to be established", len(crds)))

	waitCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if err := Wait(waitCtx, client, crds, CRDEstablishedPredicate, since); err != nil {
		return fmt.Errorf("waiting for CustomResourceDefinitions to be established: %w", err)
	}
	return waitServed(waitCtx, client, kindsDefinedBy(crds, rest), since)
}

// kindsDefinedBy returns, in first-use order and once each, the kinds of
// objs that one of crds defines and serves: spec.group and spec.names.kind
// match, and spec.versions lists the object's version with served true. An
// object at a version its definition does not serve is left out, so the
// apply fails on it at once instead of waiting for a kind that never comes.
func kindsDefinedBy(crds, objs []*unstructured.Unstructured) []schema.GroupVersionKind {
	defined := make(map[schema.GroupVersionKind]struct{})
	for _, crd := range crds {
		group, _, _ := unstructured.NestedString(crd.Object, "spec", "group")        //nolint:errcheck // a malformed CRD is ignored
		kind, _, _ := unstructured.NestedString(crd.Object, "spec", "names", "kind") //nolint:errcheck // a malformed CRD is ignored
		versions, _, _ := unstructured.NestedSlice(crd.Object, "spec", "versions")   //nolint:errcheck // a malformed CRD is ignored
		if group == "" || kind == "" {
			continue
		}
		for _, v := range versions {
			version, ok := v.(map[string]any)
			if !ok {
				continue
			}
			name, _ := version["name"].(string)   //nolint:errcheck // a malformed version is ignored
			served, _ := version["served"].(bool) //nolint:errcheck // a malformed version is ignored
			if name != "" && served {
				defined[schema.GroupVersionKind{Group: group, Version: name, Kind: kind}] = struct{}{}
			}
		}
	}

	var kinds []schema.GroupVersionKind
	seen := make(map[schema.GroupVersionKind]struct{})
	for _, obj := range objs {
		gvk := obj.GroupVersionKind()
		if _, ok := defined[gvk]; !ok {
			continue
		}
		if _, dup := seen[gvk]; dup {
			continue
		}
		seen[gvk] = struct{}{}
		kinds = append(kinds, gvk)
	}
	return kinds
}

// waitServed polls API discovery until it serves every kind of kinds, or ctx
// ends. The API server lists a kind in discovery shortly after its
// CustomResourceDefinition is Established, not in the same step. Only "not
// served" keeps the wait going: a discovery request that fails is returned.
// since is when the budget behind ctx's deadline started.
func waitServed(ctx context.Context, client *Client, kinds []schema.GroupVersionKind, since time.Time) error {
	ticker := time.NewTicker(WaitPollInterval)
	defer ticker.Stop()

	for _, gvk := range kinds {
		for {
			_, err := client.ResourceFor(ctx, gvk)
			if err == nil {
				break
			}
			if !IsKindNotServed(err) {
				return err
			}
			select {
			case <-ctx.Done():
				if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return ctx.Err()
				}
				return fmt.Errorf("timed out after %s waiting for API discovery to list a kind its CustomResourceDefinition serves: %w",
					time.Since(since).Round(time.Second), err)
			case <-ticker.C:
			}
		}
	}
	return nil
}

// kindsOfNewCRDs maps the group and kind served by each CustomResourceDefinition
// among applied that did not exist before the apply to that definition's name.
// A definition without spec.group or spec.names.kind is ignored.
func kindsOfNewCRDs(applied []stageOutcome) map[schema.GroupKind]string {
	kinds := make(map[schema.GroupKind]string)
	for _, o := range applied {
		if !o.absentBefore || !isCRD(o.obj.GroupVersionKind()) {
			continue
		}
		group, _, _ := unstructured.NestedString(o.obj.Object, "spec", "group")        //nolint:errcheck // a malformed CRD is ignored
		kind, _, _ := unstructured.NestedString(o.obj.Object, "spec", "names", "kind") //nolint:errcheck // a malformed CRD is ignored
		if group == "" || kind == "" {
			continue
		}
		kinds[schema.GroupKind{Group: group, Kind: kind}] = o.obj.GetName()
	}
	return kinds
}

// newNamespaces returns the namespaces a dry run treats as created by this
// apply: named, plus every core Namespace among applied that did not exist
// before the apply.
func newNamespaces(applied []stageOutcome, named []string) map[string]struct{} {
	namespaces := make(map[string]struct{}, len(named))
	for _, ns := range named {
		if ns != "" {
			namespaces[ns] = struct{}{}
		}
	}
	for _, o := range applied {
		gvk := o.obj.GroupVersionKind()
		if o.absentBefore && gvk.Group == "" && gvk.Kind == "Namespace" {
			namespaces[o.obj.GetName()] = struct{}{}
		}
	}
	return namespaces
}

// ApplyOne performs server-side apply for a single resource.
// Returns the status of the operation (created, configured, or unchanged).
func ApplyOne(ctx context.Context, client *Client, obj *unstructured.Unstructured, opts ApplyOptions) (string, error) {
	status, _, err := applyOne(ctx, client, obj, opts)
	return status, err
}

// applyOne is ApplyOne that also reports whether the pre-apply read returned
// NotFound, which, unlike the "created" status, no other read error implies.
func applyOne(ctx context.Context, client *Client, obj *unstructured.Unstructured, opts ApplyOptions) (status string, absentBefore bool, err error) {
	gvr, err := client.ResourceFor(ctx, obj.GroupVersionKind())
	if err != nil {
		return "", false, err
	}
	ns := obj.GetNamespace()

	// Check if resource already exists to determine status after apply.
	var existingVersion string
	existing, getErr := client.ResourceClient(gvr, ns).Get(ctx, obj.GetName(), metav1.GetOptions{})
	if getErr == nil {
		existingVersion = existing.GetResourceVersion()
		obj = guardPVCResize(ctx, client, obj, existing)
	}
	// If GET fails (NotFound or other), existingVersion stays empty -> "created"
	absentBefore = apierrors.IsNotFound(getErr)

	data, err := json.Marshal(obj)
	if err != nil {
		return "", absentBefore, fmt.Errorf("marshaling resource: %w", err)
	}

	patchOpts := metav1.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        output.BoolPtr(true),
	}

	if opts.DryRun {
		patchOpts.DryRun = []string{metav1.DryRunAll}
	}

	result, patchErr := client.ResourceClient(gvr, ns).Patch(
		ctx, obj.GetName(), types.ApplyPatchType, data, patchOpts,
	)

	if patchErr != nil {
		return "", absentBefore, patchErr
	}

	// Determine status from before/after comparison.
	if existingVersion == "" {
		return output.StatusCreated, absentBefore, nil
	}
	if opts.DryRun {
		// A dry-run persists nothing, so the response carries the live
		// resourceVersion whether or not the apply would change the object.
		// Compare the response body with the live object instead.
		if result != nil && dryRunUnchanged(existing, result) {
			return output.StatusUnchanged, absentBefore, nil
		}
		return output.StatusConfigured, absentBefore, nil
	}
	if result != nil && result.GetResourceVersion() == existingVersion {
		return output.StatusUnchanged, absentBefore, nil
	}
	return output.StatusConfigured, absentBefore, nil
}

// dryRunUnchanged reports whether a server-side dry-run apply response equals
// the live object once the fields the server rewrites on every read or write
// are set aside.
func dryRunUnchanged(live, dryRun *unstructured.Unstructured) bool {
	return equality.Semantic.DeepEqual(normalizedContent(live), normalizedContent(dryRun))
}

// normalizedContent returns a copy of obj's content without the volatile fields that
// differ between a live object and its dry-run projection regardless of
// whether the apply changes anything: managedFields (timestamps and the
// applied field sets), resourceVersion, generation and status.
func normalizedContent(obj *unstructured.Unstructured) map[string]any {
	content := obj.DeepCopy().Object
	unstructured.RemoveNestedField(content, "metadata", "managedFields")
	unstructured.RemoveNestedField(content, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(content, "metadata", "generation")
	unstructured.RemoveNestedField(content, "status")
	return content
}
