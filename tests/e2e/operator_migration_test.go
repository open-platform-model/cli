package e2e

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// legacyManifest is the install manifest of the last operator release that
// attaches one, as a client-side `kubectl apply` or an older
// `opm operator install` put it on clusters.
const legacyManifest = "tests/e2e/testdata/legacy-operator/install-v1.0.0-beta.8.yaml"

// supersededBindings are the earlier manifest's role bindings the operator
// module replaces under other names.
var supersededBindings = [][2]string{
	{"clusterrolebinding", "opm-operator-manager-rolebinding"},
	{"clusterrolebinding", "opm-operator-metrics-auth-rolebinding"},
	{"rolebinding", "opm-operator-leader-election-rolebinding"},
}

// kubectlRun runs kubectl against the kind cluster, failing the test on error.
func kubectlRun(t *testing.T, kubeconfig string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	full := append([]string{"--kubeconfig", kubeconfig, "--context", kindContext}, args...)
	out, err := exec.CommandContext(ctx, "kubectl", full...).CombinedOutput()
	require.NoError(t, err, "kubectl %s: %s", strings.Join(args, " "), out)
}

func uidOf(t *testing.T, kubeconfig, kind, namespace, name string) string {
	t.Helper()
	args := []string{"get", kind, name, "-o", "jsonpath={.metadata.uid}"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	return kubectlOut(t, kubeconfig, args...)
}

func existsOnCluster(t *testing.T, kubeconfig, kind, namespace, name string) bool {
	t.Helper()
	args := []string{"get", kind, name, "--ignore-not-found", "-o", "name"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	return kubectlOut(t, kubeconfig, args...) != ""
}

// installOperator runs `opm operator install` with a budget that fits
// whether or not the cluster can pull the operator image, and returns its
// stderr. With an unpullable image only the rollout wait may fail.
func installOperator(t *testing.T, kubeconfig, workDir string, pullable bool, extra ...string) (string, error) {
	t.Helper()
	budget := 45 * time.Second
	if pullable {
		budget = 150 * time.Second
	}
	args := append([]string{"operator", "install", "--skip-platform",
		"--kubeconfig", kubeconfig, "--context", kindContext, "--timeout", budget.String()}, extra...)
	stdout, stderr, err := runOPMWithEnv(t, workDir, homeDir, budget+60*time.Second, args...)
	if pullable {
		require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	} else if err != nil {
		require.Contains(t, stderr, "did not complete its rollout", "only the rollout may fail: stdout=%s", stdout)
	}
	return stderr, err
}

// applyCLIOwnedInstance creates a ModuleInstance the operator never
// reconciles, standing in for an instance the migration must not touch.
func applyCLIOwnedInstance(t *testing.T, kubeconfig string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext, "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(`apiVersion: opmodel.dev/v1alpha1
kind: ModuleInstance
metadata:
  name: e2e-migrate-keep
  namespace: default
spec:
  owner: cli
  module:
    path: opmodel.dev/modules/e2e-fixture@v0
    version: v0.0.0
`)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "applying the ModuleInstance fixture: %s", out)
}

// TestE2E_Operator_MigratesManifestInstall installs the operator from the
// last release manifest, as earlier CLIs and kubectl did, and then runs the
// module install over it: a client-side origin taken through --crds-only
// first, a refused migration that changes nothing, and an opm-cli origin
// left as an interrupted migration leaves it. DESTRUCTIVE like the
// lifecycle test, and restores the dev operator the same way.
func TestE2E_Operator_MigratesManifestInstall(t *testing.T) {
	kubeconfig := requireKindCluster(t)
	manifest := repoPath(t, legacyManifest)

	t.Cleanup(func() { restoreDevOperator(t) })
	t.Cleanup(func() {
		kubectlDeleteIfExists(t, kubeconfig, "moduleinstance", "e2e-migrate-keep", "-n", "default")
		resetOperatorCluster(t, kubeconfig)
	})

	tmpDir, err := os.MkdirTemp("", "e2e-operator-migration-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	pullable := imagePullable(t, pinnedOperatorImage(t))

	t.Run("client-side kubectl install, crds-only first", func(t *testing.T) {
		resetOperatorCluster(t, kubeconfig)
		kubectlRun(t, kubeconfig, "apply", "-f", manifest)
		applyCLIOwnedInstance(t, kubeconfig)
		before := map[string]string{
			"crd":            uidOf(t, kubeconfig, "crd", "", "moduleinstances.opmodel.dev"),
			"namespace":      uidOf(t, kubeconfig, "namespace", "", "opm-operator-system"),
			"serviceaccount": uidOf(t, kubeconfig, "serviceaccount", "opm-operator-system", "opm-operator-controller-manager"),
			"clusterrole":    uidOf(t, kubeconfig, "clusterrole", "", "opm-operator-manager-role"),
			"deployment":     uidOf(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager"),
			"instance":       uidOf(t, kubeconfig, "moduleinstance", "default", "e2e-migrate-keep"),
		}
		instanceGen := kubectlOut(t, kubeconfig, "get", "moduleinstance", "e2e-migrate-keep", "-n", "default", "-o", "jsonpath={.metadata.generation}")
		deployRV := kubectlOut(t, kubeconfig, "get", "deployment", "opm-operator-controller-manager", "-n", "opm-operator-system", "-o", "jsonpath={.metadata.resourceVersion}")

		// --crds-only proves and applies the CRDs and touches nothing else.
		stdout, stderr, err := runOPMWithEnv(t, tmpDir, homeDir, 60*time.Second,
			"operator", "install", "--crds-only", "--kubeconfig", kubeconfig, "--context", kindContext, "--timeout", "30s")
		require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
		assert.NotContains(t, stderr, "migrating the operator")
		assert.Equal(t, deployRV, kubectlOut(t, kubeconfig, "get", "deployment", "opm-operator-controller-manager", "-n", "opm-operator-system", "-o", "jsonpath={.metadata.resourceVersion}"))
		for _, b := range supersededBindings {
			ns := ""
			if b[0] == "rolebinding" {
				ns = "opm-operator-system"
			}
			assert.True(t, existsOnCluster(t, kubeconfig, b[0], ns, b[1]), "%s survives --crds-only", b[1])
		}

		// The full install migrates the rest.
		stderr, _ = installOperator(t, kubeconfig, tmpDir, pullable)
		assert.Contains(t, stderr, "migrating the operator installed from an earlier release manifest")
		assert.Contains(t, stderr, "recreated Deployment/opm-operator-system/opm-operator-controller-manager")
		assert.Contains(t, stderr, "deleted   ClusterRoleBinding/opm-operator-manager-rolebinding: superseded by opm-operator-manager-role")

		for key, kind := range map[string][3]string{
			"crd":            {"crd", "", "moduleinstances.opmodel.dev"},
			"namespace":      {"namespace", "", "opm-operator-system"},
			"serviceaccount": {"serviceaccount", "opm-operator-system", "opm-operator-controller-manager"},
			"clusterrole":    {"clusterrole", "", "opm-operator-manager-role"},
			"instance":       {"moduleinstance", "default", "e2e-migrate-keep"},
		} {
			assert.Equal(t, before[key], uidOf(t, kubeconfig, kind[0], kind[1], kind[2]), "%s keeps its uid", key)
		}
		assert.NotEqual(t, before["deployment"], uidOf(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager"), "the Deployment is recreated")
		assert.Equal(t, instanceGen, kubectlOut(t, kubeconfig, "get", "moduleinstance", "e2e-migrate-keep", "-n", "default", "-o", "jsonpath={.metadata.generation}"))
		for _, b := range supersededBindings {
			ns := ""
			if b[0] == "rolebinding" {
				ns = "opm-operator-system"
			}
			assert.False(t, existsOnCluster(t, kubeconfig, b[0], ns, b[1]), "%s is deleted", b[1])
		}
		assert.True(t, existsOnCluster(t, kubeconfig, "clusterrolebinding", "", "opm-operator-manager-role"))

		// The labels and annotation of the client-side apply are gone, and
		// no field is left to its manager.
		nsLabels := kubectlOut(t, kubeconfig, "get", "namespace", "opm-operator-system", "-o", "jsonpath={.metadata.labels}")
		assert.NotContains(t, nsLabels, "kustomize")
		assert.NotContains(t, nsLabels, "control-plane")
		for _, obj := range [][2]string{{"namespace", "opm-operator-system"}, {"crd", "moduleinstances.opmodel.dev"}} {
			meta := kubectlOut(t, kubeconfig, "get", obj[0], obj[1], "--show-managed-fields", "-o", "jsonpath={.metadata}")
			assert.NotContains(t, meta, "last-applied-configuration", obj[1])
			assert.NotContains(t, meta, "kubectl-client-side-apply", obj[1])
		}

		kinds := kubectlOut(t, kubeconfig, "get", "moduleinstance", "opm-operator", "-n", "opm-operator-system",
			"-o", "jsonpath={.status.inventory.entries[*].kind}")
		assert.Equal(t, 4, strings.Count(kinds, "CustomResourceDefinition"), "inventory kinds: %s", kinds)
		for _, k := range []string{"Namespace", "ServiceAccount", "Deployment", "ClusterRoleBinding", "RoleBinding", "Service"} {
			assert.Contains(t, kinds, k)
		}

		// A re-run has nothing to migrate and recreates nothing.
		uid := uidOf(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager")
		stderr, _ = installOperator(t, kubeconfig, tmpDir, pullable)
		assert.NotContains(t, stderr, "migrating the operator")
		assert.Equal(t, uid, uidOf(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager"))
	})

	t.Run("an unproven object refuses and changes nothing", func(t *testing.T) {
		resetOperatorCluster(t, kubeconfig)
		kubectlRun(t, kubeconfig, "apply", "--server-side", "--field-manager=opm-cli", "-f", manifest)
		kubectlRun(t, kubeconfig, "label", "clusterrolebinding", "opm-operator-manager-rolebinding",
			"module-instance.opmodel.dev/uuid=00000000-0000-0000-0000-000000000000")
		crdRV := kubectlOut(t, kubeconfig, "get", "crd", "moduleinstances.opmodel.dev", "-o", "jsonpath={.metadata.resourceVersion}")
		deployUID := uidOf(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager")

		_, stderr, err := runOPMWithEnv(t, tmpDir, homeDir, 90*time.Second, "operator", "install", "--skip-platform",
			"--kubeconfig", kubeconfig, "--context", kindContext, "--timeout", "30s")
		require.Error(t, err)
		assert.Contains(t, stderr, "operator migration refused")
		assert.Contains(t, stderr, "ClusterRoleBinding/opm-operator-manager-rolebinding")
		assert.Equal(t, crdRV, kubectlOut(t, kubeconfig, "get", "crd", "moduleinstances.opmodel.dev", "-o", "jsonpath={.metadata.resourceVersion}"), "the CRD step did not run")
		assert.Equal(t, deployUID, uidOf(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager"))
	})

	t.Run("an interrupted migration completes on the next run", func(t *testing.T) {
		resetOperatorCluster(t, kubeconfig)
		// An older 'opm operator install' applied the manifest as opm-cli;
		// a migration then stopped after deleting the Deployment and one
		// binding.
		kubectlRun(t, kubeconfig, "apply", "--server-side", "--field-manager=opm-cli", "-f", manifest)
		kubectlRun(t, kubeconfig, "delete", "deployment", "opm-operator-controller-manager", "-n", "opm-operator-system", "--wait=true")
		kubectlRun(t, kubeconfig, "delete", "clusterrolebinding", "opm-operator-manager-rolebinding")
		nsUID := uidOf(t, kubeconfig, "namespace", "", "opm-operator-system")

		stderr, _ := installOperator(t, kubeconfig, tmpDir, pullable)
		assert.Contains(t, stderr, "deleted   ClusterRoleBinding/opm-operator-metrics-auth-rolebinding")
		assert.NotContains(t, stderr, "recreated Deployment", "the Deployment was already gone")
		selector := kubectlOut(t, kubeconfig, "get", "deployment", "opm-operator-controller-manager", "-n", "opm-operator-system",
			"-o", `jsonpath={.spec.selector.matchLabels.module-instance\.opmodel\.dev/name}`)
		assert.Equal(t, "opm-operator", selector, "the module created the Deployment with its own selector")
		assert.Equal(t, nsUID, uidOf(t, kubeconfig, "namespace", "", "opm-operator-system"))
		for _, b := range supersededBindings {
			ns := ""
			if b[0] == "rolebinding" {
				ns = "opm-operator-system"
			}
			assert.False(t, existsOnCluster(t, kubeconfig, b[0], ns, b[1]), "%s is deleted", b[1])
		}
		owner := kubectlOut(t, kubeconfig, "get", "moduleinstance", "opm-operator", "-n", "opm-operator-system", "-o", "jsonpath={.spec.owner}")
		assert.Equal(t, "cli", owner)
	})
}
