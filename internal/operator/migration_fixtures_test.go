package operator

import (
	"maps"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// The install origins of an earlier manifest.
const (
	originOPMCLI     = "opm-cli"     // server-side apply as opm-cli (an earlier 'opm operator install')
	originClientSide = "client-side" // client-side 'kubectl apply -f install.yaml'
)

// legacyRoleRefs are the roles the earlier manifests' bindings grant.
var legacyRoleRefs = map[string][2]string{
	"opm-operator-manager-rolebinding":         {"ClusterRole", "opm-operator-manager-role"},
	"opm-operator-metrics-auth-rolebinding":    {"ClusterRole", "opm-operator-metrics-auth-role"},
	"opm-operator-leader-election-rolebinding": {"Role", "opm-operator-leader-election-role"},
}

// shippedIn reports whether a proof-list entry's release range holds tag.
func shippedIn(o LegacyObject, tag string) bool {
	first, last, _ := strings.Cut(o.Releases, "..")
	return semver.Compare(first, tag) <= 0 && semver.Compare(tag, last) <= 0
}

// manifestObjects are the live objects an operator release's manifest left
// on a cluster, installed from origin: each with the manifest's labels, a
// stable uid, and the field manager that origin records.
func manifestObjects(t *testing.T, tag, origin string) []*unstructured.Unstructured {
	t.Helper()
	var objs []*unstructured.Unstructured
	for _, o := range LegacyObjects {
		if !shippedIn(o, tag) {
			continue
		}
		objs = append(objs, legacyLive(t, o, origin))
	}
	require.NotEmpty(t, objs, tag)
	return objs
}

// legacyLive is one proof-list entry as an earlier manifest left it.
func legacyLive(t *testing.T, o LegacyObject, origin string) *unstructured.Unstructured {
	t.Helper()
	apiVersion := "v1"
	if o.Group != "" {
		apiVersion = o.Group + "/v1"
	}
	obj := &unstructured.Unstructured{Object: map[string]any{"apiVersion": apiVersion, "kind": o.Kind}}
	obj.SetName(o.Name)
	obj.SetNamespace(o.Namespace)
	if len(o.Labels) > 0 {
		obj.SetLabels(maps.Clone(o.Labels))
	}
	obj.SetUID(types.UID("legacy-" + strings.ToLower(o.Kind) + "-" + o.Name))
	obj.SetResourceVersion("1")
	switch o.Kind {
	case kindDeployment:
		sel := map[string]any{}
		for k, v := range o.Selector {
			sel[k] = v
		}
		obj.Object["spec"] = map[string]any{"selector": map[string]any{"matchLabels": sel}}
	case "ClusterRoleBinding", "RoleBinding":
		ref, ok := legacyRoleRefs[o.Name]
		require.True(t, ok, o.Name)
		obj.Object["roleRef"] = map[string]any{"apiGroup": "rbac.authorization.k8s.io", "kind": ref[0], "name": ref[1]}
	}
	manager, op := originOPMCLI, metav1.ManagedFieldsOperationApply
	if origin == originClientSide {
		manager, op = clientSideApplyManager, metav1.ManagedFieldsOperationUpdate
		obj.SetAnnotations(map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "{}"})
	}
	obj.SetManagedFields([]metav1.ManagedFieldsEntry{{
		Manager: manager, Operation: op, APIVersion: apiVersion, FieldsType: "FieldsV1",
		FieldsV1: metav1.NewFieldsV1(`{"f:metadata":{"f:labels":{}}}`),
	}})
	return obj
}

// moduleBinding is a binding the operator module renders, named after its
// role as the catalog's role abstraction names it.
func moduleBinding(kind, namespace, roleKind, role string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "rbac.authorization.k8s.io/v1", "kind": kind,
		"metadata": map[string]any{"name": role},
		"roleRef":  map[string]any{"apiGroup": "rbac.authorization.k8s.io", "kind": roleKind, "name": role},
	}}
	if namespace != "" {
		obj.SetNamespace(namespace)
	}
	return labeled(obj, "rbac", "0.1.0")
}

// migrationModuleObjects is the fixture render plus the bindings, roles and
// Service of the operator module, so a migration finds a replacement for
// each superseded binding. Without dropBinding, every binding is rendered.
func migrationModuleObjects(dropBinding string) []*unstructured.Unstructured {
	objs := moduleObjects(renderOpts{})
	objs = append(objs,
		labeled(&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRole",
			"metadata": map[string]any{"name": "opm-operator-metrics-auth-role"}, "rules": []any{},
		}}, "metrics-auth-rbac", "0.1.0"),
		labeled(&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "Role",
			"metadata": map[string]any{"name": "opm-operator-leader-election-role", "namespace": OperatorNamespace}, "rules": []any{},
		}}, "leader-election", "0.1.0"),
		labeled(&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "Service",
			"metadata": map[string]any{"name": "opm-operator-controller-manager-metrics-service", "namespace": OperatorNamespace},
		}}, "controller-manager", "0.1.0"),
	)
	for _, b := range []*unstructured.Unstructured{
		moduleBinding("ClusterRoleBinding", "", "ClusterRole", "opm-operator-manager-role"),
		moduleBinding("ClusterRoleBinding", "", "ClusterRole", "opm-operator-metrics-auth-role"),
		moduleBinding("RoleBinding", OperatorNamespace, "Role", "opm-operator-leader-election-role"),
	} {
		if b.GetName() != dropBinding {
			objs = append(objs, b)
		}
	}
	return objs
}

// withoutKey drops the object of a kind and name from objs.
func withoutKey(objs []*unstructured.Unstructured, kind, name string) []*unstructured.Unstructured {
	var out []*unstructured.Unstructured
	for _, o := range objs {
		if o.GetKind() == kind && o.GetName() == name {
			continue
		}
		out = append(out, o)
	}
	return out
}

// findObj returns the object of a kind and name in objs, or nil.
func findObj(objs []*unstructured.Unstructured, kind, name string) *unstructured.Unstructured {
	for _, o := range objs {
		if o.GetKind() == kind && o.GetName() == name {
			return o
		}
	}
	return nil
}

// paths names objects as install's lines do, for comparisons.
func paths(objs []*unstructured.Unstructured) []string {
	out := make([]string, 0, len(objs))
	for _, o := range objs {
		out = append(out, objPath(o.GetKind(), o.GetNamespace(), o.GetName()))
	}
	return out
}

// writesAnything reports whether a plan makes any migration write.
func writesAnything(p *MigrationPlan) bool {
	return len(p.MoveOwnership) > 0 || p.RecreateDeployment != nil || len(p.DeleteBindings) > 0
}
