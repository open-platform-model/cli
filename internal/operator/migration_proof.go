package operator

import (
	"fmt"
	"maps"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

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
// not disprove it), the Deployment also the listed selector, and none of
// the instance identity labels. An object carrying instanceUUID is the
// instance's own. The reason says why an object is unproven.
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
	return VerdictProven, ""
}

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
