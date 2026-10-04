package operator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
)

const beta8 = "v1.0.0-beta.8"

// planOn runs PlanMigration over a fake cluster holding objs against the
// render, and asserts it wrote nothing.
func planOn(t *testing.T, render []*unstructured.Unstructured, crdsOnly bool, objs ...*unstructured.Unstructured) (*MigrationPlan, error) {
	t.Helper()
	fc := newFakeCluster(t, objs...)
	if crdsOnly {
		render = renderedCRDs(render)
	}
	plan, err := PlanMigration(context.Background(), fc.client, render, testInstanceUUID, crdsOnly)
	assert.Empty(t, fc.Writes(), "the proof writes nothing")
	return plan, err
}

func bindingNames(bs []SupersededBinding) map[string]string {
	out := map[string]string{}
	for _, b := range bs {
		out[b.Live.GetName()] = b.Replacement
	}
	return out
}

var wantDeletes = map[string]string{
	"opm-operator-manager-rolebinding":         "opm-operator-manager-role",
	"opm-operator-metrics-auth-rolebinding":    "opm-operator-metrics-auth-role",
	"opm-operator-leader-election-rolebinding": "opm-operator-leader-election-role",
}

func TestPlanMigration_FreshCluster(t *testing.T) {
	plan, err := planOn(t, migrationModuleObjects(""), false)
	require.NoError(t, err)
	assert.Equal(t, &MigrationPlan{}, plan)
	assert.False(t, plan.Migrates())
	assert.False(t, plan.Writes())
}

func TestPlanMigration_ManifestOrigins(t *testing.T) {
	for _, origin := range []string{originOPMCLI, originClientSide} {
		t.Run(origin, func(t *testing.T) {
			plan, err := planOn(t, migrationModuleObjects(""), false, manifestObjects(t, beta8, origin)...)
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{
				"CustomResourceDefinition/moduleinstances.opmodel.dev",
				"CustomResourceDefinition/modulepackages.opmodel.dev",
				"CustomResourceDefinition/platforms.opmodel.dev",
				"CustomResourceDefinition/transformerregistrations.opmodel.dev",
				"Namespace/opm-operator-system",
				"ClusterRole/opm-operator-manager-role",
				"ClusterRole/opm-operator-metrics-auth-role",
				"ServiceAccount/opm-operator-system/opm-operator-controller-manager",
				"Role/opm-operator-system/opm-operator-leader-election-role",
				"Service/opm-operator-system/opm-operator-controller-manager-metrics-service",
			}, paths(plan.Adopt))
			assert.Empty(t, plan.Ours)
			require.NotNil(t, plan.RecreateDeployment)
			assert.Equal(t, ControllerDeploymentName, plan.RecreateDeployment.GetName())
			assert.Equal(t, wantDeletes, bindingNames(plan.DeleteBindings))
			// The fixture render lacks the viewer and admin roles, so the
			// earlier ones are left in place.
			assert.Len(t, plan.LeftInPlace, 8)
			if origin == originClientSide {
				assert.ElementsMatch(t, paths(plan.Adopt), paths(plan.MoveOwnership))
			} else {
				assert.Empty(t, plan.MoveOwnership)
			}
			assert.True(t, plan.Migrates())
		})
	}
}

// "Install over a module-rendered kubectl install": every object is the
// instance's own; nothing is adopted, recreated or deleted.
func TestPlanMigration_ModuleRenderedKubectlInstall(t *testing.T) {
	render := migrationModuleObjects("")
	live := make([]*unstructured.Unstructured, 0, len(render))
	for _, o := range render {
		l := o.DeepCopy()
		l.SetManagedFields(legacyLive(t, LegacyObject{Kind: "ConfigMap", Name: "x"}, originClientSide).GetManagedFields())
		live = append(live, l)
	}
	plan, err := planOn(t, render, false, live...)
	require.NoError(t, err)
	assert.Len(t, plan.Ours, len(render))
	assert.Empty(t, plan.Adopt)
	assert.Nil(t, plan.RecreateDeployment)
	assert.Empty(t, plan.DeleteBindings)
	assert.Empty(t, plan.LeftInPlace)
	assert.Len(t, plan.MoveOwnership, len(render), "a client-side apply of the module's manifest leaves its manager behind")
	assert.False(t, plan.Migrates())
}

// A cluster upgraded through kubectl since v0.7 still holds the older
// release's objects; the module renders none of them.
func TestPlanMigration_PreV1Leftovers(t *testing.T) {
	cluster := append(manifestObjects(t, beta8, originClientSide), manifestObjects(t, "v0.7.5", originClientSide)...)
	cluster = dedupe(cluster)
	plan, err := planOn(t, migrationModuleObjects(""), false, cluster...)
	require.NoError(t, err)
	left := make([]string, 0, len(plan.LeftInPlace))
	for _, o := range plan.LeftInPlace {
		left = append(left, o.Kind+"/"+o.Name)
	}
	assert.Subset(t, left, []string{
		"ClusterRole/opm-operator-modulerelease-viewer-role",
		"CustomResourceDefinition/releases.releases.opmodel.dev",
		"CustomResourceDefinition/platforms.releases.opmodel.dev",
	})
	assert.Len(t, plan.DeleteBindings, 3)
}

func dedupe(objs []*unstructured.Unstructured) []*unstructured.Unstructured {
	seen := map[objKey]bool{}
	var out []*unstructured.Unstructured
	for _, o := range objs {
		if !seen[keyOf(o)] {
			seen[keyOf(o)] = true
			out = append(out, o)
		}
	}
	return out
}

func TestPlanMigration_Refusals(t *testing.T) {
	otherInstance := func(o *unstructured.Unstructured) {
		l := o.GetLabels()
		l["module-instance.opmodel.dev/uuid"] = "other-uuid"
		l["module-instance.opmodel.dev/name"] = "web"
		l["module-instance.opmodel.dev/namespace"] = "team-a"
		o.SetLabels(l)
	}
	mutated := func(kind, name string, f func(*unstructured.Unstructured)) []*unstructured.Unstructured {
		objs := manifestObjects(t, beta8, originOPMCLI)
		f(findObj(objs, kind, name))
		return objs
	}
	cases := []struct {
		name    string
		cluster []*unstructured.Unstructured
		render  []*unstructured.Unstructured
		want    []string
	}{
		{
			name:    "an unproven binding",
			cluster: mutated("ClusterRoleBinding", "opm-operator-manager-rolebinding", otherInstance),
			render:  migrationModuleObjects(""),
			want:    []string{"1 object(s)", "ClusterRoleBinding/opm-operator-manager-rolebinding: carries the identity of instance team-a/web", "nothing was changed"},
		},
		{
			name:    "a proven binding with no replacement in the render",
			cluster: manifestObjects(t, beta8, originOPMCLI),
			render:  migrationModuleObjects("opm-operator-manager-role"),
			want:    []string{"ClusterRoleBinding/opm-operator-manager-rolebinding: the module renders no ClusterRoleBinding to ClusterRole/opm-operator-manager-role"},
		},
		{
			name:    "an unproven rendered object",
			cluster: mutated("Namespace", OperatorNamespace, func(o *unstructured.Unstructured) { o.SetLabels(nil) }),
			render:  migrationModuleObjects(""),
			want:    []string{`Namespace/opm-operator-system: label app.kubernetes.io/managed-by is missing, earlier manifests set "kustomize"`},
		},
		{
			name: "two unproven objects are both named",
			cluster: func() []*unstructured.Unstructured {
				objs := mutated("Deployment", ControllerDeploymentName, otherInstance)
				otherInstance(findObj(objs, "RoleBinding", "opm-operator-leader-election-rolebinding"))
				return objs
			}(),
			render: migrationModuleObjects(""),
			want: []string{"2 object(s)",
				"Deployment/opm-operator-system/opm-operator-controller-manager: carries the identity",
				"RoleBinding/opm-operator-system/opm-operator-leader-election-rolebinding: carries the identity"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := planOn(t, c.render, false, c.cluster...)
			var refusal *MigrationRefusalError
			require.ErrorAs(t, err, &refusal)
			for _, w := range c.want {
				assert.Contains(t, err.Error(), w)
			}
		})
	}
}

// An unproven leftover the migration neither adopts nor deletes is left
// alone and not reported.
func TestPlanMigration_UnprovenLeftoverDoesNotRefuse(t *testing.T) {
	cluster := dedupe(append(manifestObjects(t, beta8, originOPMCLI), manifestObjects(t, "v0.7.5", originOPMCLI)...))
	findObj(cluster, "ClusterRole", "opm-operator-modulerelease-viewer-role").SetLabels(nil)
	plan, err := planOn(t, migrationModuleObjects(""), false, cluster...)
	require.NoError(t, err)
	for _, o := range plan.LeftInPlace {
		assert.NotEqual(t, "opm-operator-modulerelease-viewer-role", o.Name)
	}
}

func TestPlanMigration_UnreadableEntryRefuses(t *testing.T) {
	for _, denied := range []error{
		apierrors.NewForbidden(schema.GroupResource{Group: "rbac.authorization.k8s.io", Resource: "clusterrolebindings"}, "x", errors.New("no")),
		apierrors.NewUnauthorized("no"),
	} {
		fc := newFakeCluster(t, manifestObjects(t, beta8, originOPMCLI)...)
		fc.fake.PrependReactor("get", "clusterrolebindings", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, denied
		})
		_, err := PlanMigration(context.Background(), fc.client, migrationModuleObjects(""), testInstanceUUID, false)
		var readErr *MigrationReadError
		require.ErrorAs(t, err, &readErr)
		assert.Contains(t, err.Error(), "operator migration refused: cannot read ClusterRoleBinding/")
		assert.True(t, apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err), "the command maps it to exit 4")
		assert.Empty(t, fc.Writes())
	}
}

// A run interrupted after the Deployment and one binding were deleted plans
// the rest.
func TestPlanMigration_ResumedRun(t *testing.T) {
	cluster := withoutKey(manifestObjects(t, beta8, originOPMCLI), kindDeployment, ControllerDeploymentName)
	cluster = withoutKey(cluster, "ClusterRoleBinding", "opm-operator-manager-rolebinding")
	plan, err := planOn(t, migrationModuleObjects(""), false, cluster...)
	require.NoError(t, err)
	assert.Nil(t, plan.RecreateDeployment)
	assert.Len(t, plan.DeleteBindings, 2)
	assert.NotEmpty(t, plan.Adopt)
}

// --crds-only proves the rendered CRDs only and plans no write.
func TestPlanMigration_CRDsOnly(t *testing.T) {
	plan, err := planOn(t, migrationModuleObjects(""), true, manifestObjects(t, beta8, originClientSide)...)
	require.NoError(t, err)
	assert.Len(t, plan.Adopt, 4)
	for _, o := range plan.Adopt {
		assert.Equal(t, kindCustomResourceDefinition, o.GetKind())
	}
	assert.Nil(t, plan.RecreateDeployment)
	assert.Empty(t, plan.DeleteBindings)
	assert.Empty(t, plan.LeftInPlace)
	assert.Empty(t, plan.MoveOwnership)
	assert.False(t, plan.Writes())

	// An unproven CRD refuses.
	cluster := manifestObjects(t, beta8, originOPMCLI)
	findObj(cluster, kindCustomResourceDefinition, "platforms.opmodel.dev").SetLabels(map[string]string{"module-instance.opmodel.dev/uuid": "other"})
	_, err = planOn(t, migrationModuleObjects(""), true, cluster...)
	var refusal *MigrationRefusalError
	require.ErrorAs(t, err, &refusal)
}

// Until the plan is admitted (the guard still runs without it), a proven
// manifest install is refused by the guard and an unproven one by the
// proof; neither writes.
func TestPlanInstall_ProofRunsBeforeTheGuard(t *testing.T) {
	cluster := manifestObjects(t, beta8, originOPMCLI)
	findObj(cluster, "ClusterRoleBinding", "opm-operator-manager-rolebinding").SetLabels(map[string]string{"module-instance.opmodel.dev/uuid": "other"})
	fc := newFakeCluster(t, cluster...)
	_, err := PlanInstall(context.Background(), newEnv(fc, &fakeRender{objs: migrationModuleObjects("")}),
		testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: time.Second})
	var refusal *MigrationRefusalError
	require.ErrorAs(t, err, &refusal)
	assert.Empty(t, fc.Writes())
}
