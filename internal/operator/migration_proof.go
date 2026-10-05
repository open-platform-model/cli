package operator

import (
	"fmt"
	"maps"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/kubernetes"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

// Verdict is what the migration's proof says about one live object on the
// proof list.
type Verdict int

const (
	// VerdictAbsent: the object does not exist.
	VerdictAbsent Verdict = iota
	// VerdictProven: the object came from an earlier operator manifest.
	VerdictProven
	// VerdictOurs: the object carries the operator instance's own identity,
	// so it needs no proof.
	VerdictOurs
	// VerdictUnproven: the object exists and fails the proof.
	VerdictUnproven
)

// identityLabels are the OPM instance identity labels; an object carrying
// any of them is never proven to come from an earlier manifest.
var identityLabels = []string{
	pkgcore.LabelModuleInstanceUUID,
	pkgcore.LabelModuleInstanceName,
	pkgcore.LabelModuleInstanceNamespace,
}

// ProveLegacy proves a live object against its proof-list entry (0012:D8:R6):
// it is proven when it carries every label the entry lists (added labels do
// not disprove it), the Deployment also the listed selector, none of the
// instance identity labels, and no mark of another tool that keeps applying
// it (see gitOpsOwner). An object carrying instanceUUID is the instance's
// own. The reason says why an object is unproven.
func ProveLegacy(live *unstructured.Unstructured, want LegacyObject, instanceUUID string) (verdict Verdict, reason string) {
	if live == nil {
		return VerdictAbsent, ""
	}
	labels := live.GetLabels()
	if uuid := labels[pkgcore.LabelModuleInstanceUUID]; uuid != "" && uuid == instanceUUID {
		return VerdictOurs, ""
	}
	if uuid := labels[pkgcore.LabelModuleInstanceUUID]; uuid != "" {
		return VerdictUnproven, fmt.Sprintf("carries the identity of instance %s", instanceRef(labels))
	}
	for _, key := range identityLabels {
		if _, ok := labels[key]; ok {
			return VerdictUnproven, fmt.Sprintf("carries the instance identity label %s, which no earlier manifest set", key)
		}
	}
	for _, key := range sortedKeys(want.Labels) {
		got, ok := labels[key]
		if !ok {
			return VerdictUnproven, fmt.Sprintf("label %s is missing, earlier manifests set %q", key, want.Labels[key])
		}
		if got != want.Labels[key] {
			return VerdictUnproven, fmt.Sprintf("label %s is %q, earlier manifests set %q", key, got, want.Labels[key])
		}
	}
	if want.Selector != nil {
		sel, _, err := unstructured.NestedStringMap(live.Object, "spec", "selector", "matchLabels")
		if err != nil {
			return VerdictUnproven, fmt.Sprintf("selector cannot be read: %v", err)
		}
		if !maps.Equal(sel, want.Selector) {
			return VerdictUnproven, fmt.Sprintf("selector is %v, earlier manifests set %v", sel, want.Selector)
		}
	}
	if owner := gitOpsOwner(live); owner != "" {
		return VerdictUnproven, owner + "; suspend that tool's reconciliation of the operator first, or it re-applies what install changes"
	}
	return VerdictProven, ""
}

// gitOpsLabels and gitOpsAnnotations are the tracking marks Flux and Argo CD
// set on the objects they apply; gitOpsManagers are their field managers.
var (
	gitOpsLabels = map[string]string{
		"kustomize.toolkit.fluxcd.io/name": "Flux",
		"helm.toolkit.fluxcd.io/name":      "Flux",
		"argocd.argoproj.io/instance":      "Argo CD",
	}
	gitOpsAnnotations = map[string]string{
		"argocd.argoproj.io/tracking-id": "Argo CD",
	}
	gitOpsManagers = map[string]string{
		"kustomize-controller":          "Flux",
		"helm-controller":               "Flux",
		"argocd-controller":             "Argo CD",
		"argocd-application-controller": "Argo CD",
	}
)

// gitOpsOwner names the tool that keeps applying a live object, or "": a
// Flux or Argo CD tracking label or annotation, a Flux or Argo CD field
// manager, or a server-side apply by any manager other than opm-cli or
// kubectl. Install would fight such a tool over the objects it adopts,
// recreates or deletes.
func gitOpsOwner(live *unstructured.Unstructured) string {
	labels, annotations := live.GetLabels(), live.GetAnnotations()
	for _, key := range sortedKeys(gitOpsLabels) {
		if _, ok := labels[key]; ok {
			return fmt.Sprintf("is applied by %s (label %s)", gitOpsLabels[key], key)
		}
	}
	for _, key := range sortedKeys(gitOpsAnnotations) {
		if _, ok := annotations[key]; ok {
			return fmt.Sprintf("is applied by %s (annotation %s)", gitOpsAnnotations[key], key)
		}
	}
	for _, mf := range live.GetManagedFields() {
		if tool, ok := gitOpsManagers[mf.Manager]; ok {
			return fmt.Sprintf("is applied by %s (field manager %s)", tool, mf.Manager)
		}
		if mf.Operation == metav1.ManagedFieldsOperationApply && mf.Manager != kubernetes.FieldManager && mf.Manager != kubectlApplyManager {
			return fmt.Sprintf("is server-side applied by field manager %s, not by opm-cli or kubectl", mf.Manager)
		}
	}
	return ""
}

// kubectlApplyManager is the field manager of a `kubectl apply
// --server-side` without --field-manager.
const kubectlApplyManager = "kubectl"

// instanceRef names the instance an object's identity labels point at.
func instanceRef(labels map[string]string) string {
	name, ns := labels[pkgcore.LabelModuleInstanceName], labels[pkgcore.LabelModuleInstanceNamespace]
	switch {
	case name != "" && ns != "":
		return ns + "/" + name
	case name != "":
		return name
	default:
		return labels[pkgcore.LabelModuleInstanceUUID]
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
