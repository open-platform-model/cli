package operator

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/inventory"
)

var (
	crbGVR = schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}
	rbGVR  = schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}
)

// eventLog records every write the fake receives, in order, naming the
// object and, for a patch, whether it was an apply or a JSON patch.
type eventLog struct {
	mu     sync.Mutex
	events []string
}

func (l *eventLog) attach(fc *fakeCluster) {
	fc.fake.PrependReactor("*", "*", func(a k8stesting.Action) (bool, runtime.Object, error) {
		var ev string
		switch act := a.(type) {
		case k8stesting.PatchAction:
			kind := "apply"
			if act.GetPatchType() != types.ApplyPatchType {
				kind = "jsonpatch"
			}
			ev = kind + " " + a.GetResource().Resource + "/" + act.GetName()
		case k8stesting.DeleteAction:
			ev = "delete " + a.GetResource().Resource + "/" + act.GetName()
		default:
			if a.GetVerb() != "create" && a.GetVerb() != "update" {
				return false, nil, nil
			}
			ev = a.GetVerb() + " " + a.GetResource().Resource
		}
		l.mu.Lock()
		l.events = append(l.events, ev)
		l.mu.Unlock()
		return false, nil, nil
	})
}

func (l *eventLog) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.events...)
}

// index is the position of the first event with prefix, or -1.
func index(events []string, prefix string) int {
	for i, e := range events {
		if strings.HasPrefix(e, prefix) {
			return i
		}
	}
	return -1
}

func lastIndex(events []string, prefix string) int {
	for i := len(events) - 1; i >= 0; i-- {
		if strings.HasPrefix(events[i], prefix) {
			return i
		}
	}
	return -1
}

func planFor(t *testing.T, fc *fakeCluster) *MigrationPlan {
	t.Helper()
	plan, err := PlanMigration(context.Background(), fc.client, migrationModuleObjects(""), testInstanceUUID, false)
	require.NoError(t, err)
	return plan
}

// The Deployment goes first and is waited out, then the bindings; a second
// call has nothing left to do; nothing outside the plan is touched.
func TestDeleteSuperseded_OrderAndIdempotence(t *testing.T) {
	fastPolling(t)
	fc := newFakeCluster(t, manifestObjects(t, beta8, originOPMCLI)...)
	log := &eventLog{}
	log.attach(fc)
	plan := planFor(t, fc)

	require.NoError(t, MoveOwnership(context.Background(), fc.client, plan))
	require.NoError(t, DeleteSuperseded(context.Background(), fc.client, plan, time.Now()))
	assert.Equal(t, []string{
		"delete deployments/opm-operator-controller-manager",
		"delete rolebindings/opm-operator-leader-election-rolebinding",
		"delete clusterrolebindings/opm-operator-manager-rolebinding",
		"delete clusterrolebindings/opm-operator-metrics-auth-rolebinding",
	}, log.all(), "an opm-cli origin moves no ownership; only the planned objects are deleted")
	assert.False(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName))
	assert.False(t, fc.exists(crbGVR, "", "opm-operator-manager-rolebinding"))
	assert.True(t, fc.exists(namespaceGVR, "", OperatorNamespace))

	// Again with the same plan: every delete reads NotFound and is done.
	require.NoError(t, DeleteSuperseded(context.Background(), fc.client, plan, time.Now()))
	// A recomputed plan has nothing to delete.
	again := planFor(t, fc)
	assert.Nil(t, again.RecreateDeployment)
	assert.Empty(t, again.DeleteBindings)
}

// A client-side origin hands kubectl's fields to opm-cli, one JSON patch per
// planned object, and a second call sends none.
func TestMoveOwnership_ClientSideOrigin(t *testing.T) {
	fc := newFakeCluster(t, manifestObjects(t, beta8, originClientSide)...)
	log := &eventLog{}
	log.attach(fc)
	plan := planFor(t, fc)
	require.NotEmpty(t, plan.MoveOwnership)

	require.NoError(t, MoveOwnership(context.Background(), fc.client, plan))
	events := log.all()
	assert.Len(t, events, len(plan.MoveOwnership))
	for _, e := range events {
		assert.True(t, strings.HasPrefix(e, "jsonpatch "), e)
	}
	ns := fc.mustGet(namespaceGVR, "", OperatorNamespace)
	require.Len(t, ns.GetManagedFields(), 1)
	assert.Equal(t, "opm-cli", ns.GetManagedFields()[0].Manager)
	assert.Equal(t, "kustomize", ns.GetLabels()["app.kubernetes.io/managed-by"], "the move changes managedFields only")

	require.NoError(t, MoveOwnership(context.Background(), fc.client, plan))
	assert.Len(t, log.all(), len(events), "nothing is left to move")
}

// "Run interrupted between the deletes": the second binding delete fails;
// a recomputed plan and a second run complete the migration.
func TestDeleteSuperseded_InterruptedRunCompletes(t *testing.T) {
	fastPolling(t)
	fc := newFakeCluster(t, manifestObjects(t, beta8, originOPMCLI)...)
	fail := true
	fc.fake.PrependReactor("delete", "clusterrolebindings", func(a k8stesting.Action) (bool, runtime.Object, error) {
		if fail && a.(k8stesting.DeleteAction).GetName() == "opm-operator-metrics-auth-rolebinding" {
			return true, nil, errors.New("connection reset")
		}
		return false, nil, nil
	})

	err := DeleteSuperseded(context.Background(), fc.client, planFor(t, fc), time.Now())
	var stopped *MigrationStoppedError
	require.ErrorAs(t, err, &stopped)
	assert.Contains(t, err.Error(), "operator migration stopped after it began: deleting ClusterRoleBinding/opm-operator-metrics-auth-rolebinding")
	assert.Contains(t, err.Error(), "re-run 'opm operator install' to complete it")
	assert.False(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName))
	assert.False(t, fc.exists(crbGVR, "", "opm-operator-manager-rolebinding"))
	assert.True(t, fc.exists(crbGVR, "", "opm-operator-metrics-auth-rolebinding"))

	fail = false
	resumed := planFor(t, fc)
	assert.Nil(t, resumed.RecreateDeployment)
	assert.Equal(t, map[string]string{
		"opm-operator-metrics-auth-rolebinding": "opm-operator-metrics-auth-role",
	}, bindingNames(resumed.DeleteBindings))
	require.NoError(t, DeleteSuperseded(context.Background(), fc.client, resumed, time.Now()))
	assert.False(t, fc.exists(crbGVR, "", "opm-operator-metrics-auth-rolebinding"))
	assert.False(t, fc.exists(rbGVR, OperatorNamespace, "opm-operator-leader-election-rolebinding"))
}

// The check phase admits the proven objects of the beta.8 manifest, and the
// install writes in order: CRD step, ownership moves, Deployment delete,
// binding deletes, instance apply. Every rendered object is recorded.
func TestInstall_MigratesAManifestInstall(t *testing.T) {
	for _, origin := range []string{originOPMCLI, originClientSide} {
		t.Run(origin, func(t *testing.T) {
			releasedCLI(t)
			fastPolling(t)
			fc := newFakeCluster(t, manifestObjects(t, beta8, origin)...)
			log := &eventLog{}
			log.attach(fc)
			render := &fakeRender{objs: migrationModuleObjects("")}

			saUID := fc.mustGet(schema.GroupVersionResource{Version: "v1", Resource: "serviceaccounts"}, OperatorNamespace, "opm-operator-controller-manager").GetUID()
			result, err := install(t, fc, render, PlanOptions{})
			require.NoError(t, err)
			assert.True(t, result.Recorded)

			ev := log.all()
			lastCRD := lastIndex(ev[:8], "apply customresourcedefinitions/")
			require.Equal(t, 3, lastCRD, "the CRD step comes first: %v", ev)
			delDeploy := index(ev, "delete deployments/")
			delBinding := index(ev, "delete clusterrolebindings/")
			lastDelete := lastIndex(ev, "delete rolebindings/")
			instance := index(ev, "apply namespaces/")
			require.Positive(t, delDeploy)
			assert.Less(t, lastCRD, delDeploy)
			assert.Less(t, delDeploy, delBinding)
			assert.Less(t, lastDelete, instance, "every delete precedes the instance apply")
			if origin == originClientSide {
				firstMove := index(ev, "jsonpatch ")
				assert.Less(t, lastCRD, firstMove)
				assert.Less(t, lastIndex(ev, "jsonpatch "), delDeploy)
			} else {
				assert.Equal(t, -1, index(ev, "jsonpatch "))
			}

			names := entryNames(fc.record())
			assert.Len(t, names, len(render.objs))
			sa := fc.mustGet(schema.GroupVersionResource{Version: "v1", Resource: "serviceaccounts"}, OperatorNamespace, "opm-operator-controller-manager")
			assert.Equal(t, saUID, sa.GetUID(), "an adopted object keeps its uid")
			d := fc.mustGet(deploymentGVR, OperatorNamespace, ControllerDeploymentName)
			assert.NotEqual(t, "legacy-deployment-"+ControllerDeploymentName, string(d.GetUID()), "the Deployment is recreated")
			assert.False(t, fc.exists(crbGVR, "", "opm-operator-manager-rolebinding"))
			assert.True(t, fc.exists(crbGVR, "", "opm-operator-manager-role"))
		})
	}
}

// The check-phase guard admits proven kustomize-labeled objects.
func TestPlanInstall_AdmitsProvenObjects(t *testing.T) {
	fc := newFakeCluster(t, manifestObjects(t, beta8, originOPMCLI)...)
	plan, err := PlanInstall(context.Background(), newEnv(fc, &fakeRender{objs: migrationModuleObjects("")}),
		testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: time.Second})
	require.NoError(t, err)
	assert.True(t, plan.Migration.Migrates())
	admit := plan.Migration.Admit()
	assert.Contains(t, admit, inventory.K8sIdentity{Kind: "Namespace", Name: OperatorNamespace})
	assert.Contains(t, admit, inventory.K8sIdentity{Group: "apps", Kind: "Deployment", Namespace: OperatorNamespace, Name: ControllerDeploymentName})
	assert.Empty(t, fc.Writes())
}

// "CRDs-only over a manifest install" and "Full install after CRDs-only
// completes the migration".
func TestInstall_CRDsOnlyThenFullInstallMigrates(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t, manifestObjects(t, beta8, originClientSide)...)
	log := &eventLog{}
	log.attach(fc)
	render := &fakeRender{objs: migrationModuleObjects("")}
	deployRV := fc.mustGet(deploymentGVR, OperatorNamespace, ControllerDeploymentName).GetResourceVersion()

	result, err := install(t, fc, render, PlanOptions{CRDsOnly: true})
	require.NoError(t, err)
	assert.Equal(t, 4, result.CRDs)
	for _, e := range log.all() {
		assert.True(t, strings.HasPrefix(e, "apply customresourcedefinitions/"), "--crds-only writes only the CRDs: %s", e)
	}
	assert.Equal(t, deployRV, fc.mustGet(deploymentGVR, OperatorNamespace, ControllerDeploymentName).GetResourceVersion())
	assert.True(t, fc.exists(crbGVR, "", "opm-operator-manager-rolebinding"))

	_, err = install(t, fc, render, PlanOptions{})
	require.NoError(t, err)
	names := entryNames(fc.record())
	assert.Len(t, names, len(render.objs))
	for _, crd := range CRDNames() {
		assert.Contains(t, names, "CustomResourceDefinition/"+crd)
	}
	assert.False(t, fc.exists(crbGVR, "", "opm-operator-manager-rolebinding"))
}

// "Fields of another manager are left alone": only the client-side apply's
// entry moves to opm-cli.
func TestMoveOwnership_LeavesOtherManagers(t *testing.T) {
	cluster := manifestObjects(t, beta8, originClientSide)
	ns := findObj(cluster, "Namespace", OperatorNamespace)
	ns.SetManagedFields(append(ns.GetManagedFields(), metav1.ManagedFieldsEntry{
		Manager: "kubectl-label", Operation: metav1.ManagedFieldsOperationUpdate, APIVersion: "v1", FieldsType: "FieldsV1",
		FieldsV1: metav1.NewFieldsV1(`{"f:metadata":{"f:labels":{"f:team":{}}}}`),
	}))
	fc := newFakeCluster(t, cluster...)
	require.NoError(t, MoveOwnership(context.Background(), fc.client, planFor(t, fc)))

	fields := fc.mustGet(namespaceGVR, "", OperatorNamespace).GetManagedFields()
	managers := make([]string, 0, len(fields))
	for _, mf := range fields {
		managers = append(managers, mf.Manager+":"+string(mf.Operation))
	}
	assert.ElementsMatch(t, []string{"opm-cli:Apply", "kubectl-label:Update"}, managers)
}

// A run whose instance apply stamped the identity on some rendered objects
// (the Namespace, the ServiceAccount and the new Deployment) and then
// stopped before writing the record resumes: the stamped objects are the
// instance's own, the rest are still adopted, the remaining bindings are
// deleted, and the record names every rendered object.
func TestInstall_ResumesAfterAPartialInstanceApply(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	render := &fakeRender{objs: migrationModuleObjects("")}
	cluster := withoutKey(manifestObjects(t, beta8, originOPMCLI), kindDeployment, ControllerDeploymentName)
	cluster = withoutKey(cluster, "ClusterRoleBinding", "opm-operator-manager-rolebinding")
	for _, o := range cluster {
		if o.GetKind() == "Namespace" || o.GetKind() == "ServiceAccount" {
			labeled(o, "controller-manager", "0.1.0")
		}
	}
	deploy := findObj(render.objs, kindDeployment, ControllerDeploymentName).DeepCopy()
	deploy.SetUID("partial-deployment")
	require.NoError(t, unstructured.SetNestedMap(deploy.Object, map[string]any{
		"observedGeneration": int64(0), "replicas": int64(1), "updatedReplicas": int64(1), "availableReplicas": int64(1),
	}, "status"))
	cluster = append(cluster, deploy)
	fc := newFakeCluster(t, cluster...)
	require.Nil(t, fc.record(), "the stopped run wrote no record")

	plan, err := PlanInstall(context.Background(), newEnv(fc, render), testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: 2 * time.Second})
	require.NoError(t, err, "the guard admits a mix of the instance's own and adopted objects")
	assert.Nil(t, plan.Migration.RecreateDeployment, "the Deployment is already the instance's")
	assert.NotEmpty(t, plan.Migration.Ours)
	assert.NotEmpty(t, plan.Migration.Adopt)

	result, err := Install(context.Background(), newEnv(fc, render), plan)
	require.NoError(t, err)
	assert.True(t, result.Recorded)
	assert.Len(t, entryNames(fc.record()), len(render.objs))
	assert.Equal(t, "partial-deployment", string(fc.mustGet(deploymentGVR, OperatorNamespace, ControllerDeploymentName).GetUID()))
	assert.False(t, fc.exists(crbGVR, "", "opm-operator-metrics-auth-rolebinding"))
	assert.False(t, fc.exists(rbGVR, OperatorNamespace, "opm-operator-leader-election-rolebinding"))
}

// Each delete of the migration carries the UID the plan's delete verdict
// judged, so an object replaced since the check phase is not deleted.
func TestDeleteSuperseded_SendsTheJudgedUID(t *testing.T) {
	fastPolling(t)
	cluster := manifestObjects(t, beta8, originOPMCLI)
	findObj(cluster, kindDeployment, ControllerDeploymentName).SetUID("uid-deployment")
	findObj(cluster, "ClusterRoleBinding", "opm-operator-manager-rolebinding").SetUID("uid-binding")
	fc := newFakeCluster(t, cluster...)
	plan := planFor(t, fc)

	require.NoError(t, DeleteSuperseded(context.Background(), fc.client, plan, time.Now()))

	uids := map[string]string{}
	for _, a := range fc.fake.Actions() {
		d, ok := a.(k8stesting.DeleteAction)
		if !ok {
			continue
		}
		if pre := d.GetDeleteOptions().Preconditions; pre != nil && pre.UID != nil {
			uids[d.GetName()] = string(*pre.UID)
		}
	}
	assert.Equal(t, "uid-deployment", uids[ControllerDeploymentName])
	assert.Equal(t, "uid-binding", uids["opm-operator-manager-rolebinding"])
}

// An object the plan holds no delete verdict for is never deleted.
func TestDeleteProven_WithoutAVerdictStops(t *testing.T) {
	cluster := manifestObjects(t, beta8, originOPMCLI)
	fc := newFakeCluster(t, cluster...)
	deployment := findObj(cluster, kindDeployment, ControllerDeploymentName)

	err := deleteProven(context.Background(), fc.client, &MigrationPlan{}, deployment)

	var stopped *MigrationStoppedError
	require.ErrorAs(t, err, &stopped)
	assert.Empty(t, fc.Writes())
	assert.True(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName))
}
