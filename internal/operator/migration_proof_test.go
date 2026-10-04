package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestProveLegacy(t *testing.T) {
	sa := legacyObject("ServiceAccount", OperatorNamespace, "opm-operator-controller-manager")
	crd := legacyObject("CustomResourceDefinition", "", "moduleinstances.opmodel.dev")

	live := func(o LegacyObject, mutate func(*unstructured.Unstructured)) *unstructured.Unstructured {
		obj := legacyLive(t, o, originOPMCLI)
		if mutate != nil {
			mutate(obj)
		}
		return obj
	}
	setLabel := func(k, v string) func(*unstructured.Unstructured) {
		return func(o *unstructured.Unstructured) {
			l := o.GetLabels()
			if l == nil {
				l = map[string]string{}
			}
			l[k] = v
			o.SetLabels(l)
		}
	}
	dropLabel := func(k string) func(*unstructured.Unstructured) {
		return func(o *unstructured.Unstructured) {
			l := o.GetLabels()
			delete(l, k)
			o.SetLabels(l)
		}
	}

	withManager := func(manager string, op metav1.ManagedFieldsOperationType) func(*unstructured.Unstructured) {
		return func(o *unstructured.Unstructured) {
			o.SetManagedFields(append(o.GetManagedFields(), metav1.ManagedFieldsEntry{Manager: manager, Operation: op}))
		}
	}

	cases := []struct {
		name   string
		live   *unstructured.Unstructured
		want   LegacyObject
		verd   Verdict
		reason string
	}{
		{"absent", nil, sa, VerdictAbsent, ""},
		{"proven", live(sa, nil), sa, VerdictProven, ""},
		{"an added label still proves", live(sa, setLabel("team", "a")), sa, VerdictProven, ""},
		{"a CRD with no listed labels", live(crd, nil), crd, VerdictProven, ""},
		{"a missing listed label", live(sa, dropLabel("app.kubernetes.io/name")), sa, VerdictUnproven,
			`label app.kubernetes.io/name is missing, earlier manifests set "opm-operator"`},
		{"a changed listed label", live(sa, setLabel("app.kubernetes.io/name", "x")), sa, VerdictUnproven,
			`label app.kubernetes.io/name is "x", earlier manifests set "opm-operator"`},
		// An object an operator module release's install.yaml created
		// carries this instance's identity: it is the instance's own.
		{"this instance's identity", live(crd, func(o *unstructured.Unstructured) {
			o.SetLabels(map[string]string{
				"module-instance.opmodel.dev/uuid": testInstanceUUID,
				"module-instance.opmodel.dev/name": OperatorInstanceName,
				"app.kubernetes.io/managed-by":     "opm-cli",
			})
		}), crd, VerdictOurs, ""},
		{"another instance's identity", live(sa, func(o *unstructured.Unstructured) {
			setLabel("module-instance.opmodel.dev/uuid", "other")(o)
			setLabel("module-instance.opmodel.dev/name", "web")(o)
			setLabel("module-instance.opmodel.dev/namespace", "team-a")(o)
		}), sa, VerdictUnproven, "carries the identity of instance team-a/web"},
		{"a partial identity", live(sa, setLabel("module-instance.opmodel.dev/name", "web")), sa, VerdictUnproven,
			"carries the instance identity label module-instance.opmodel.dev/name, which no earlier manifest set"},
		{"the earlier Deployment", live(legacyDeployment, nil), legacyDeployment, VerdictProven, ""},
		{"applied client-side by kubectl", legacyLive(t, sa, originClientSide), sa, VerdictProven, ""},
		{"server-side applied by kubectl", live(sa, withManager("kubectl", metav1.ManagedFieldsOperationApply)), sa, VerdictProven, ""},
		{"labeled by hand", live(sa, withManager("kubectl-label", metav1.ManagedFieldsOperationUpdate)), sa, VerdictProven, ""},
		{"a Flux Kustomization label", live(sa, setLabel("kustomize.toolkit.fluxcd.io/name", "ops")), sa, VerdictUnproven,
			"is applied by Flux (label kustomize.toolkit.fluxcd.io/name); suspend that tool's reconciliation"},
		{"a Flux HelmRelease label", live(crd, setLabel("helm.toolkit.fluxcd.io/name", "ops")), crd, VerdictUnproven,
			"is applied by Flux (label helm.toolkit.fluxcd.io/name)"},
		{"an Argo CD instance label", live(crd, setLabel("argocd.argoproj.io/instance", "ops")), crd, VerdictUnproven,
			"is applied by Argo CD (label argocd.argoproj.io/instance)"},
		{"the Flux field manager", live(sa, withManager("kustomize-controller", metav1.ManagedFieldsOperationApply)), sa, VerdictUnproven,
			"is applied by Flux (field manager kustomize-controller)"},
		{"another server-side applier", live(sa, withManager("pulumi-kubernetes", metav1.ManagedFieldsOperationApply)), sa, VerdictUnproven,
			"is server-side applied by field manager pulumi-kubernetes"},
		{"a Deployment with another selector", live(legacyDeployment, func(o *unstructured.Unstructured) {
			_ = unstructured.SetNestedStringMap(o.Object, map[string]string{"app": "x"}, "spec", "selector", "matchLabels")
		}), legacyDeployment, VerdictUnproven, "selector is map[app:x]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			verdict, reason := ProveLegacy(c.live, c.want, testInstanceUUID)
			assert.Equal(t, c.verd, verdict)
			assert.Contains(t, reason, c.reason)
			if c.reason == "" {
				assert.Empty(t, reason)
			}
		})
	}
}
