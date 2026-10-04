package operator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestMigrationReport_FullMigration(t *testing.T) {
	fc := newFakeCluster(t, manifestObjects(t, "v1.0.0-beta.5", originOPMCLI)...)
	// The fixture render plus the roles the real module renders, so nothing
	// of beta.5 is left in place.
	render := migrationModuleObjects("")
	for _, name := range []string{
		"opm-operator-metrics-reader", "opm-operator-moduleinstance-admin-role",
		"opm-operator-moduleinstance-editor-role", "opm-operator-moduleinstance-viewer-role",
		"opm-operator-transformerregistration-admin-role",
	} {
		render = append(render, labeled(&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRole", "metadata": map[string]any{"name": name},
		}}, "admin-roles", "0.1.0"))
	}
	plan, err := PlanMigration(context.Background(), fc.client, render, testInstanceUUID, false)
	require.NoError(t, err)
	assert.Equal(t, `migrating the operator installed from an earlier release manifest
  adopted   CustomResourceDefinition/moduleinstances.opmodel.dev
  adopted   CustomResourceDefinition/modulepackages.opmodel.dev
  adopted   CustomResourceDefinition/platforms.opmodel.dev
  adopted   CustomResourceDefinition/transformerregistrations.opmodel.dev
  adopted   Namespace/opm-operator-system
  adopted   ServiceAccount/opm-operator-system/opm-operator-controller-manager
  adopted   Role/opm-operator-system/opm-operator-leader-election-role
  adopted   ClusterRole/opm-operator-manager-role
  adopted   ClusterRole/opm-operator-metrics-auth-role
  adopted   ClusterRole/opm-operator-metrics-reader
  adopted   ClusterRole/opm-operator-moduleinstance-admin-role
  adopted   ClusterRole/opm-operator-moduleinstance-editor-role
  adopted   ClusterRole/opm-operator-moduleinstance-viewer-role
  adopted   ClusterRole/opm-operator-transformerregistration-admin-role
  adopted   Service/opm-operator-system/opm-operator-controller-manager-metrics-service
  recreated Deployment/opm-operator-system/opm-operator-controller-manager: selector changed; patches made to the earlier Deployment are not carried over
  deleted   RoleBinding/opm-operator-system/opm-operator-leader-election-rolebinding: superseded by opm-operator-leader-election-role
  deleted   ClusterRoleBinding/opm-operator-manager-rolebinding: superseded by opm-operator-manager-role
  deleted   ClusterRoleBinding/opm-operator-metrics-auth-rolebinding: superseded by opm-operator-metrics-auth-role`,
		strings.Join(MigrationReport(plan), "\n"))
}

// "Leftover object of an older release is named and kept".
func TestMigrationReport_PreV1Leftovers(t *testing.T) {
	cluster := dedupe(append(manifestObjects(t, "v1.0.0-beta.5", originClientSide), manifestObjects(t, "v0.7.5", originClientSide)...))
	fc := newFakeCluster(t, cluster...)
	plan, err := PlanMigration(context.Background(), fc.client, migrationModuleObjects(""), testInstanceUUID, false)
	require.NoError(t, err)
	report := strings.Join(MigrationReport(plan), "\n")
	assert.Contains(t, report, "\n  left      ClusterRole/opm-operator-modulerelease-viewer-role: not part of the operator module")
	assert.Contains(t, report, "\n  left      CustomResourceDefinition/releases.releases.opmodel.dev: not part of the operator module")
	assert.NotContains(t, report, "left      ClusterRoleBinding", "a superseded binding is deleted, not left")
}

// "Install after a completed migration prints no migration lines" and
// "Re-running install after the migration recreates nothing".
func TestInstall_RerunAfterMigrationPrintsNothing(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	cluster := dedupe(append(manifestObjects(t, beta8, originOPMCLI), manifestObjects(t, "v0.7.5", originOPMCLI)...))
	fc := newFakeCluster(t, cluster...)
	render := &fakeRender{objs: migrationModuleObjects("")}
	_, err := install(t, fc, render, PlanOptions{})
	require.NoError(t, err)
	uid := fc.mustGet(deploymentGVR, OperatorNamespace, ControllerDeploymentName).GetUID()

	plan, err := PlanInstall(context.Background(), newEnv(fc, render), testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: 2 * time.Second})
	require.NoError(t, err)
	assert.NotEmpty(t, plan.Migration.LeftInPlace, "the v0.7 leftovers remain")
	assert.Empty(t, MigrationReport(plan.Migration))
	assert.False(t, plan.Migration.Writes())
	_, err = Install(context.Background(), newEnv(fc, render), plan)
	require.NoError(t, err)
	assert.Equal(t, uid, fc.mustGet(deploymentGVR, OperatorNamespace, ControllerDeploymentName).GetUID())
	assert.True(t, fc.exists(clusterRoleGVR, "", "opm-operator-modulerelease-viewer-role"), "a leftover is never removed")
}

func TestMigrationRefusal_NamesEveryObject(t *testing.T) {
	err := &MigrationRefusalError{Blocks: []MigrationBlock{
		{Kind: "ClusterRoleBinding", Name: "opm-operator-manager-rolebinding", Reason: "carries the identity of instance team-a/web"},
		{Kind: "ServiceAccount", Namespace: OperatorNamespace, Name: "opm-operator-controller-manager", Reason: `label app.kubernetes.io/name is "x", earlier manifests set "opm-operator"`},
	}}
	assert.Equal(t, `operator migration refused: 2 object(s) of an earlier operator manifest cannot be proven:
  ClusterRoleBinding/opm-operator-manager-rolebinding: carries the identity of instance team-a/web
  ServiceAccount/opm-operator-system/opm-operator-controller-manager: label app.kubernetes.io/name is "x", earlier manifests set "opm-operator"
nothing was changed; remove or rename these objects, then re-run 'opm operator install'`, err.Error())
}
