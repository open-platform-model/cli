package operator

import (
	"context"
	"fmt"
	"maps"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
)

// clientSideApplyManager is the field manager a client-side `kubectl apply`
// records its fields under.
const clientSideApplyManager = "kubectl-client-side-apply"

// objKey identifies a Kubernetes object by group, kind, namespace and name.
type objKey = inventory.K8sIdentity

func keyOf(obj *unstructured.Unstructured) objKey {
	return objKey{Group: obj.GroupVersionKind().Group, Kind: obj.GetKind(), Namespace: obj.GetNamespace(), Name: obj.GetName()}
}

func legacyKey(o LegacyObject) objKey {
	return objKey{Group: o.Group, Kind: o.Kind, Namespace: o.Namespace, Name: o.Name}
}

// objPath is an object as install's lines name it: Kind/namespace/name, or
// Kind/name for a cluster-scoped object.
func objPath(kind, namespace, name string) string {
	if namespace != "" {
		return kind + "/" + namespace + "/" + name
	}
	return kind + "/" + name
}

// SupersededBinding is a proven earlier role binding the migration deletes,
// with the rendered binding that replaces it.
type SupersededBinding struct {
	Live        *unstructured.Unstructured
	Replacement string
}

// MigrationPlan is install's migration of an operator installed from an
// earlier release manifest, computed from the cluster with no write. Every
// object it names was read during the check phase.
type MigrationPlan struct {
	// Adopt are proven objects the module renders: the guard admits them.
	Adopt []*unstructured.Unstructured
	// Ours are rendered objects that already carry the instance's identity.
	Ours []*unstructured.Unstructured
	// MoveOwnership are objects of Adopt or Ours that hold fields of a
	// client-side kubectl apply, handed to opm-cli before the instance apply.
	MoveOwnership []*unstructured.Unstructured
	// RecreateDeployment is the proven earlier controller Deployment, whose
	// selector the module changes; nil when there is none.
	RecreateDeployment *unstructured.Unstructured
	// DeleteBindings are the proven superseded role bindings.
	DeleteBindings []SupersededBinding
	// LeftInPlace are proven earlier-manifest objects the module does not
	// render and the migration does not delete; they are only reported.
	LeftInPlace []LegacyObject
}

// Migrates reports whether the plan adopts, recreates or deletes anything,
// which is when install prints its migration lines.
func (p *MigrationPlan) Migrates() bool {
	return p != nil && (len(p.Adopt) > 0 || p.RecreateDeployment != nil || len(p.DeleteBindings) > 0)
}

// Admit is the set the apply guard admits: the adopted objects, the
// Deployment the migration recreates, and the rendered objects that already
// carry the instance's identity, so a resumed run's set is complete. Nil
// for a nil plan.
func (p *MigrationPlan) Admit() inventory.AdmitSet {
	if p == nil {
		return nil
	}
	set := inventory.AdmitSet{}
	objs := append(append([]*unstructured.Unstructured(nil), p.Adopt...), p.Ours...)
	if p.RecreateDeployment != nil {
		objs = append(objs, p.RecreateDeployment)
	}
	for _, obj := range objs {
		set[keyOf(obj)] = struct{}{}
	}
	return set
}

// Writes reports whether the plan makes any migration write.
func (p *MigrationPlan) Writes() bool {
	return p != nil && (len(p.MoveOwnership) > 0 || p.RecreateDeployment != nil || len(p.DeleteBindings) > 0)
}

// MigrationBlock is one object that blocks the migration, with the reason.
type MigrationBlock struct {
	Kind, Namespace, Name string
	Reason                string
}

// MigrationRefusalError refuses an install whose migration cannot be proven;
// nothing was written.
type MigrationRefusalError struct{ Blocks []MigrationBlock }

func (e *MigrationRefusalError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "operator migration refused: %d object(s) of an earlier operator manifest cannot be proven:", len(e.Blocks))
	for _, blk := range e.Blocks {
		fmt.Fprintf(&b, "\n  %s: %s", objPath(blk.Kind, blk.Namespace, blk.Name), blk.Reason)
	}
	b.WriteString("\nnothing was changed; remove or rename these objects, then re-run 'opm operator install'")
	return b.String()
}

// MigrationReadError refuses an install whose migration cannot read an
// object it must prove; it wraps the Kubernetes error, whose exit code the
// command maps.
type MigrationReadError struct {
	Kind, Namespace, Name string
	Err                   error
}

func (e *MigrationReadError) Error() string {
	where := ""
	if e.Namespace != "" {
		where = " in " + e.Namespace
	}
	return fmt.Sprintf("operator migration refused: cannot read %s/%s%s: %v", e.Kind, e.Name, where, e.Err)
}

func (e *MigrationReadError) Unwrap() error { return e.Err }

// PlanMigration proves, with reads only, which objects of an earlier
// operator manifest the install takes over (0012:D8:R6/R7). It reads every
// proof-list entry and every object of rendered, the objects the install
// applies, and returns the plan or a *MigrationRefusalError naming every object
// that blocks it. crdsOnly restricts it to the rendered CRDs and plans no
// write. A read error other than NotFound is a *MigrationReadError.
func PlanMigration(ctx context.Context, client *kubernetes.Client, rendered []*unstructured.Unstructured, instanceUUID string, crdsOnly bool) (*MigrationPlan, error) {
	byKey := make(map[objKey]*unstructured.Unstructured, len(rendered))
	for _, obj := range rendered {
		byKey[keyOf(obj)] = obj
	}

	plan := &MigrationPlan{}
	var blocks []MigrationBlock
	listed := map[objKey]bool{}
	for _, want := range LegacyObjects {
		key := legacyKey(want)
		listed[key] = true
		_, isRendered := byKey[key]
		if crdsOnly && !isRendered {
			continue
		}
		live, err := getLive(ctx, client, legacyGVR(want), want.Namespace, want.Name, want.Kind)
		if err != nil {
			return nil, err
		}
		verdict, reason := ProveLegacy(live, want, instanceUUID)
		if blk, blocked := plan.classify(want, live, verdict, reason, isRendered, rendered); blocked {
			blocks = append(blocks, blk)
		}
	}

	// Rendered objects outside the list: only this instance's own matter.
	for _, obj := range rendered {
		key := keyOf(obj)
		if listed[key] {
			continue
		}
		live, err := getLive(ctx, client, kubernetes.GVRFromUnstructured(obj), obj.GetNamespace(), obj.GetName(), obj.GetKind())
		if err != nil {
			return nil, err
		}
		if verdict, _ := ProveLegacy(live, LegacyObject{}, instanceUUID); verdict == VerdictOurs {
			plan.Ours = append(plan.Ours, live)
		}
	}

	if len(blocks) > 0 {
		return nil, &MigrationRefusalError{Blocks: blocks}
	}
	if !crdsOnly {
		for _, live := range append(append([]*unstructured.Unstructured(nil), plan.Adopt...), plan.Ours...) {
			if hasClientSideApply(live) {
				plan.MoveOwnership = append(plan.MoveOwnership, live)
			}
		}
	}
	return plan, nil
}

// classify sorts one proof-list entry into the plan, or returns the block
// it raises: an unproven entry the migration would adopt, recreate or
// delete, or a proven superseded binding the render does not replace.
func (p *MigrationPlan) classify(want LegacyObject, live *unstructured.Unstructured, verdict Verdict, reason string, isRendered bool, rendered []*unstructured.Unstructured) (MigrationBlock, bool) {
	block := func(r string) (MigrationBlock, bool) {
		return MigrationBlock{Kind: want.Kind, Namespace: want.Namespace, Name: want.Name, Reason: r}, true
	}
	switch {
	case verdict == VerdictAbsent:
	case isRendered && verdict == VerdictOurs:
		p.Ours = append(p.Ours, live)
	case isRendered && verdict == VerdictUnproven:
		return block(reason)
	case isRendered && legacyKey(want) == legacyKey(legacyDeployment):
		p.RecreateDeployment = live
	case isRendered:
		p.Adopt = append(p.Adopt, live)
	case isSuperseded(want):
		if verdict == VerdictUnproven {
			return block(reason)
		}
		if verdict != VerdictProven {
			return MigrationBlock{}, false
		}
		repl, ref := replacementBinding(live, rendered)
		if repl == "" {
			return block(fmt.Sprintf("the module renders no %s to %s", want.Kind, ref))
		}
		p.DeleteBindings = append(p.DeleteBindings, SupersededBinding{Live: live, Replacement: repl})
	case verdict == VerdictProven:
		p.LeftInPlace = append(p.LeftInPlace, want)
	}
	return MigrationBlock{}, false
}

func isSuperseded(o LegacyObject) bool {
	for _, s := range SupersededBindings {
		if legacyKey(s) == legacyKey(o) {
			return true
		}
	}
	return false
}

// replacementBinding returns the name of the rendered binding of the live
// binding's kind and namespace whose roleRef equals the live one, or "",
// and the live roleRef as Kind/name for a refusal.
func replacementBinding(live *unstructured.Unstructured, rendered []*unstructured.Unstructured) (replacement, liveRef string) {
	ref := roleRef(live)
	liveRef = ref["kind"] + "/" + ref["name"]
	for _, obj := range rendered {
		if obj.GetKind() != live.GetKind() || obj.GetNamespace() != live.GetNamespace() {
			continue
		}
		if got := roleRef(obj); len(got) > 0 && maps.Equal(got, ref) {
			return obj.GetName(), liveRef
		}
	}
	return "", liveRef
}

// roleRef is a binding's roleRef; empty when it has none a string map holds.
func roleRef(obj *unstructured.Unstructured) map[string]string {
	ref, found, err := unstructured.NestedStringMap(obj.Object, "roleRef")
	if err != nil || !found {
		return nil
	}
	return ref
}

// hasClientSideApply reports whether a client-side kubectl apply owns
// fields of the object.
func hasClientSideApply(obj *unstructured.Unstructured) bool {
	for _, mf := range obj.GetManagedFields() {
		if mf.Manager == clientSideApplyManager && mf.Operation == metav1.ManagedFieldsOperationUpdate && mf.Subresource == "" {
			return true
		}
	}
	return false
}

// legacyGVR is a proof-list entry's resource; every kind on the list is
// served at v1 of its group.
func legacyGVR(o LegacyObject) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: o.Group, Version: "v1", Resource: kubernetes.KindToResource(o.Kind)}
}

// getLive reads one object: nil when it does not exist, a
// *MigrationReadError on any other failure.
func getLive(ctx context.Context, client *kubernetes.Client, gvr schema.GroupVersionResource, namespace, name, kind string) (*unstructured.Unstructured, error) {
	live, err := client.ResourceClient(gvr, namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, &MigrationReadError{Kind: kind, Namespace: namespace, Name: name, Err: err}
	}
	return live, nil
}
