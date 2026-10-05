package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/operator"
)

// kindContext is the kind cluster context these tests run against. Matches
// TestMain's dummy config.cue and the workspace's `task cluster:create`.
const kindContext = "kind-opm-dev"

// operatorImageRepository is the repository the operator module renders the
// controller image from by default.
const operatorImageRepository = "ghcr.io/open-platform-model/opm-operator"

// requireClusterEnv opts a run into treating a missing or unusable cluster as
// a failure. CI's cluster job sets it, so a cluster-backed test cannot pass by
// skipping; unset, the suite skips cluster tests as a developer machine needs.
const requireClusterEnv = "OPM_E2E_REQUIRE_CLUSTER"

// skipOrFailf skips, or fails when the run requires the cluster. A failure
// carries the variable as a prefix, so the log says why a skip became a failure.
func skipOrFailf(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv(requireClusterEnv) == "1" {
		t.Fatalf(requireClusterEnv+"=1: "+format, args...)
	}
	t.Skipf(format, args...)
}

// requireKindCluster skips the test (or fails it, see skipOrFailf) if the
// kind-opm-dev cluster is not reachable, and returns the real
// (non-HOME-overridden) kubeconfig path.
func requireKindCluster(t *testing.T) string {
	t.Helper()

	realHome, err := os.UserHomeDir()
	require.NoError(t, err)
	kubeconfig := filepath.Join(realHome, ".kube", "config")
	if _, statErr := os.Stat(kubeconfig); statErr != nil {
		skipOrFailf(t, "no kubeconfig at %q, so context %q cannot be reached; run `task cluster:create`",
			kubeconfig, kindContext)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext,
		"cluster-info").Run(); err != nil {
		skipOrFailf(t, "kind cluster %q not reachable; run `task cluster:create`: %v", kindContext, err)
	}

	return kubeconfig
}

// pinnedOperatorImage is the image the pinned operator module deploys,
// composed from the operator version the CLI records beside the module pin,
// so the pull-reachability check matches what `install` applies.
func pinnedOperatorImage(t *testing.T) string {
	t.Helper()
	return operatorImageRepository + ":" + operator.PinnedOperatorVersion
}

// imagePullable reports whether the pinned operator image's manifest is
// reachable, without pulling any layers. Used to decide (design risk 1)
// whether the full-install test can assert a completed rollout, or must
// fall back to asserting CRD Established + Deployment created.
func imagePullable(t *testing.T, image string) bool {
	t.Helper()

	if _, err := exec.LookPath("crane"); err != nil {
		t.Logf("crane not available; treating %s as unpullable for this run", image)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "crane", "manifest", image).Run(); err != nil {
		t.Logf("crane manifest %s failed, treating as unpullable: %v", image, err)
		return false
	}
	return true
}

// kubectlOut runs kubectl against the kind-opm-dev cluster and returns
// trimmed stdout.
func kubectlOut(t *testing.T, kubeconfig string, args ...string) string {
	t.Helper()

	fullArgs := append([]string{"--kubeconfig", kubeconfig, "--context", kindContext}, args...)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", fullArgs...).Output()
	require.NoError(t, err, "kubectl %s", strings.Join(args, " "))
	return strings.TrimSpace(string(out))
}

// kubectlDeleteIfExists best-effort deletes a resource, ignoring absence.
// Waits for the delete to actually complete (default kubectl behavior) so
// callers can rely on the resource being gone once this returns — cheap
// here since it only ever runs against test fixtures with no real workloads.
func kubectlDeleteIfExists(t *testing.T, kubeconfig string, args ...string) {
	t.Helper()

	fullArgs := append([]string{"--kubeconfig", kubeconfig, "--context", kindContext, "delete", "--ignore-not-found", "--timeout=60s"}, args...)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "kubectl", fullArgs...).Run()
}

func assertCRDEstablished(t *testing.T, kubeconfig, name string) {
	t.Helper()
	status := kubectlOut(t, kubeconfig, "get", "crd", name,
		"-o", `jsonpath={.status.conditions[?(@.type=="Established")].status}`)
	assert.Equal(t, "True", status, "CRD %s should be Established", name)
}

func assertResourceExists(t *testing.T, kubeconfig, kind, namespace, name string) {
	t.Helper()
	args := []string{"get", kind, name, "-o", "name"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	out := kubectlOut(t, kubeconfig, args...)
	assert.NotEmpty(t, out, "%s/%s should exist", kind, name)
}

// stripAllModuleInstanceFinalizers force-clears finalizers on every
// ModuleInstance so a subsequent delete can't wedge waiting on one that
// nothing will ever remove (there's no real operator running in this suite
// to do it). Best-effort: silently no-ops if the CRD isn't installed.
func stripAllModuleInstanceFinalizers(t *testing.T, kubeconfig string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext,
		"get", "moduleinstances.opmodel.dev", "--all-namespaces",
		"-o", `jsonpath={range .items[*]}{.metadata.namespace}{"/"}{.metadata.name}{"\n"}{end}`).Output()
	if err != nil {
		return // CRD not installed — nothing to strip.
	}

	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		ns, name, ok := strings.Cut(line, "/")
		if !ok {
			continue
		}
		pctx, pcancel := context.WithTimeout(context.Background(), 15*time.Second)
		_ = exec.CommandContext(pctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext,
			"patch", "moduleinstance", name, "-n", ns, "--type=merge", "-p", `{"metadata":{"finalizers":[]}}`).Run()
		pcancel()
	}
}

// restoreDevOperator rebuilds the reconciling dev operator this suite's
// destructive tests tear down, by re-running the one target that owns that
// setup rather than duplicating its steps here. A failure here fails the run:
// the test's own assertions have already passed, but the cluster is left
// without an operator, and reporting that without failing would push the
// consequence into someone else's next run as an unrelated hard-fail. This runs
// from t.Cleanup, where t.Errorf still fails the test.
func restoreDevOperator(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "task", "cluster:operator")
	cmd.Dir = repoPath(t, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("could not restore the dev operator via `task cluster:operator`: %v\n%s\n"+
			"The cluster is left without a reconciling operator. Run `task cluster:operator` by "+
			"hand before the next e2e run, or the adoption tests will fail.", err, out)
	}
}

// resetOperatorCluster removes everything an operator install can create,
// including CRDs and the Namespace (which `opm operator uninstall` itself
// deliberately never touches), so each e2e run starts from a clean slate.
func resetOperatorCluster(t *testing.T, kubeconfig string) {
	t.Helper()
	stripAllModuleInstanceFinalizers(t, kubeconfig)
	kubectlDeleteIfExists(t, kubeconfig, "moduleinstances.opmodel.dev", "--all-namespaces", "--all")
	kubectlDeleteIfExists(t, kubeconfig, append([]string{"crd"}, operator.CRDNames()...)...)
	kubectlDeleteIfExists(t, kubeconfig, "namespace", "opm-operator-system")
	kubectlDeleteIfExists(t, kubeconfig, "clusterrole", "opm-cli-user",
		"opm-operator-manager-role", "opm-operator-metrics-auth-role", "opm-operator-metrics-reader",
		"opm-operator-moduleinstance-admin-role", "opm-operator-moduleinstance-editor-role", "opm-operator-moduleinstance-viewer-role",
		"opm-operator-transformerregistration-admin-role",
		"opm-operator-platform-viewer-role", "opm-operator-modulepackage-viewer-role",
		"opm-operator-transformerregistration-viewer-role")
	// The module names each binding after its role; an earlier manifest
	// install named them "-rolebinding".
	kubectlDeleteIfExists(t, kubeconfig, "clusterrolebinding", "opm-cli-user",
		"opm-operator-manager-role", "opm-operator-metrics-auth-role",
		"opm-operator-manager-rolebinding", "opm-operator-metrics-auth-rolebinding")
	// kubectl delete gives up after its timeout and the error is ignored;
	// a CRD still terminating when the next case applies the manifest
	// makes that case's install refuse to apply over it.
	waitGone(t, kubeconfig, append([]string{"crd"}, operator.CRDNames()...)...)
}

// waitGone polls until none of the named objects of a kind exists, failing
// the test after three minutes.
func waitGone(t *testing.T, kubeconfig string, kindAndNames ...string) {
	t.Helper()
	args := append(append([]string{"--kubeconfig", kubeconfig, "--context", kindContext, "get"}, kindAndNames...), "--ignore-not-found", "-o", "name")
	deadline := time.Now().Add(3 * time.Minute)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		out, err := exec.CommandContext(ctx, "kubectl", args...).Output()
		cancel()
		if err == nil && strings.TrimSpace(string(out)) == "" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("still present after 3m: %s (err %v)", out, err)
		}
		time.Sleep(2 * time.Second)
	}
}

// TestE2E_Operator_InstallUninstallLifecycle exercises the full
// `opm operator install`/`uninstall` lifecycle against a real kind cluster:
// full install of the pinned operator module (waiting for readiness if the
// pinned image is reachable, else just CRD-established + Deployment-created —
// design risk 1) recorded as the CLI-owned instance opm-operator, idempotent
// re-install, uninstall's finalizer guard and its --remove-finalizers
// override deleting from the record (CRDs/Namespace surviving throughout),
// uninstall's refusal without a record, and a solo --crds-only install onto
// a freshly reset cluster.
// This test is DESTRUCTIVE: resetOperatorCluster deletes the CRDs, the
// opm-operator-system Namespace, and every ModuleInstance, which tears down the
// reconciling operator that the operator-owned tests require. Those tests
// hard-fail without one, so the teardown is followed by a rebuild via
// `task cluster:operator` — otherwise a full suite run would poison the next
// one, and the failure would surface far from its cause.
func TestE2E_Operator_InstallUninstallLifecycle(t *testing.T) {
	kubeconfig := requireKindCluster(t)
	image := pinnedOperatorImage(t)

	// Registered before the reset cleanup so it runs after it (t.Cleanup is
	// LIFO): tear the cluster down, then put the dev operator back.
	t.Cleanup(func() { restoreDevOperator(t) })
	t.Cleanup(func() { resetOperatorCluster(t, kubeconfig) })
	resetOperatorCluster(t, kubeconfig)

	tmpDir, err := os.MkdirTemp("", "e2e-operator-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	pullable := imagePullable(t, image)
	t.Logf("pinned image %s pullable: %v", image, pullable)

	t.Run("install", func(t *testing.T) {
		waitTimeout := 15 * time.Second
		if pullable {
			waitTimeout = 150 * time.Second
		}

		stdout, stderr, err := runOPMWithEnv(t, tmpDir, homeDir, waitTimeout+30*time.Second,
			"operator", "install",
			"--kubeconfig", kubeconfig, "--context", kindContext,
			"--timeout", waitTimeout.String())

		if pullable {
			require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
			assertResourceHealthy(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager")
		} else {
			// The apply phase (before the readiness wait) still ran —
			// only the Deployment rollout wait times out.
			require.Error(t, err, "stdout=%s stderr=%s", stdout, stderr)
		}

		assertCRDEstablished(t, kubeconfig, "moduleinstances.opmodel.dev")
		assertCRDEstablished(t, kubeconfig, "modulepackages.opmodel.dev")
		assertCRDEstablished(t, kubeconfig, "platforms.opmodel.dev")
		assertCRDEstablished(t, kubeconfig, "transformerregistrations.opmodel.dev")
		assertResourceExists(t, kubeconfig, "namespace", "", "opm-operator-system")
		assertResourceExists(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager")

		// The install is recorded: a CLI-owned instance of the pinned module
		// whose inventory lists the CRDs and the Namespace too.
		owner := kubectlOut(t, kubeconfig, "get", "moduleinstance", "opm-operator", "-n", "opm-operator-system",
			"-o", "jsonpath={.spec.owner}")
		assert.Equal(t, "cli", owner)
		kinds := kubectlOut(t, kubeconfig, "get", "moduleinstance", "opm-operator", "-n", "opm-operator-system",
			"-o", "jsonpath={.status.inventory.entries[*].kind}")
		assert.Equal(t, 4, strings.Count(kinds, "CustomResourceDefinition"), "inventory kinds: %s", kinds)
		assert.Contains(t, kinds, "Namespace")
		assert.Contains(t, kinds, "Deployment")
		assert.Contains(t, stderr, operator.PinnedOperatorVersion, "the output names the operator version")
	})

	t.Run("idempotent re-install reports unchanged", func(t *testing.T) {
		_, stderr, err := runOPMWithEnv(t, tmpDir, homeDir, 30*time.Second,
			"operator", "install", "--crds-only",
			"--kubeconfig", kubeconfig, "--context", kindContext, "--timeout", "15s")
		require.NoError(t, err, "stderr=%s", stderr)
		assert.Equal(t, 4, strings.Count(stderr, "unchanged"), "stderr=%s", stderr)
	})

	t.Run("uninstall refuses while a finalizer is armed, then --remove-finalizers proceeds", func(t *testing.T) {
		applyArmedModuleInstance(t, kubeconfig, "default", "e2e-jellyfin")
		t.Cleanup(func() {
			// Strip finalizers first: with the remaining foreign finalizer,
			// a plain delete would hang forever waiting on a controller
			// that isn't running in this suite.
			stripAllModuleInstanceFinalizers(t, kubeconfig)
			kubectlDeleteIfExists(t, kubeconfig, "moduleinstance", "e2e-jellyfin", "-n", "default")
		})

		_, stderr, err := runOPMWithEnv(t, tmpDir, homeDir, 30*time.Second,
			"operator", "uninstall", "--kubeconfig", kubeconfig, "--context", kindContext)
		require.Error(t, err)
		assert.Contains(t, stderr, "default/e2e-jellyfin")
		assert.Contains(t, stderr, "--remove-finalizers")
		// Refused: the Deployment this scenario relies on for the "survives"
		// check further down must still be present.
		assertResourceExists(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager")

		_, stderr, err = runOPMWithEnv(t, tmpDir, homeDir, 30*time.Second,
			"operator", "uninstall", "--remove-finalizers",
			"--kubeconfig", kubeconfig, "--context", kindContext)
		require.NoError(t, err, "stderr=%s", stderr)

		finalizers := kubectlOut(t, kubeconfig, "get", "moduleinstance", "e2e-jellyfin", "-n", "default",
			"-o", "jsonpath={.metadata.finalizers}")
		assert.JSONEq(t, `["example.com/foreign"]`, finalizers)

		// CRDs and the Namespace survive uninstall; the record is gone.
		assertCRDEstablished(t, kubeconfig, "moduleinstances.opmodel.dev")
		assertResourceExists(t, kubeconfig, "namespace", "", "opm-operator-system")
		out := kubectlOut(t, kubeconfig, "get", "moduleinstance", "opm-operator", "-n", "opm-operator-system",
			"--ignore-not-found", "-o", "name")
		assert.Empty(t, out, "uninstall deletes the operator's instance record")
	})

	t.Run("uninstall without a record refuses and deletes nothing", func(t *testing.T) {
		_, stderr, err := runOPMWithEnv(t, tmpDir, homeDir, 30*time.Second,
			"operator", "uninstall", "--kubeconfig", kubeconfig, "--context", kindContext)
		require.Error(t, err)
		assert.Contains(t, stderr, "opm operator install")
		assertCRDEstablished(t, kubeconfig, "moduleinstances.opmodel.dev")
	})

	t.Run("install immediately after uninstall waits out the terminating Deployment", func(t *testing.T) {
		// Uninstall is fire-and-report: its foreground deletes have been
		// accepted but the Deployment is still terminating. Without the
		// guard, install applied onto it, the garbage collector then removed
		// it, and the readiness wait burned the whole timeout. No pause here
		// is the point of the test.
		waitTimeout := 15 * time.Second
		if pullable {
			waitTimeout = 150 * time.Second
		}

		stdout, stderr, err := runOPMWithEnv(t, tmpDir, homeDir, waitTimeout+30*time.Second,
			"operator", "install", "--skip-platform",
			"--kubeconfig", kubeconfig, "--context", kindContext,
			"--timeout", waitTimeout.String())

		if pullable {
			require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
			assertResourceHealthy(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager")
		} else {
			// Only the rollout wait may fail; the Deployment existing
			// post-apply (asserted below) shows the guard let apply run.
			require.Error(t, err, "stdout=%s stderr=%s", stdout, stderr)
		}
		assertResourceExists(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager")
	})

	t.Run("crds-only on a fresh cluster installs only the CRDs", func(t *testing.T) {
		resetOperatorCluster(t, kubeconfig)

		stdout, stderr, err := runOPMWithEnv(t, tmpDir, homeDir, 30*time.Second,
			"operator", "install", "--crds-only",
			"--kubeconfig", kubeconfig, "--context", kindContext, "--timeout", "15s")
		require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

		assertCRDEstablished(t, kubeconfig, "moduleinstances.opmodel.dev")
		assertCRDEstablished(t, kubeconfig, "modulepackages.opmodel.dev")
		assertCRDEstablished(t, kubeconfig, "platforms.opmodel.dev")
		assertCRDEstablished(t, kubeconfig, "transformerregistrations.opmodel.dev")

		out := kubectlOut(t, kubeconfig, "get", "namespace", "opm-operator-system", "--ignore-not-found", "-o", "name")
		assert.Empty(t, out, "no Namespace should exist after a --crds-only install")
	})
}

// assertResourceHealthy asserts a Deployment has completed its rollout.
func assertResourceHealthy(t *testing.T, kubeconfig, kind, namespace, name string) {
	t.Helper()
	ready := kubectlOut(t, kubeconfig, "get", kind, name, "-n", namespace,
		"-o", `jsonpath={.status.conditions[?(@.type=="Available")].status}`)
	assert.Equal(t, "True", ready, "%s/%s should be Available", kind, name)
}

// applyArmedModuleInstance creates a minimal ModuleInstance carrying the
// operator's cleanup finalizer plus an unrelated one, to exercise the
// uninstall finalizer guard without needing a running operator.
func applyArmedModuleInstance(t *testing.T, kubeconfig, namespace, name string) {
	t.Helper()

	manifest := `apiVersion: opmodel.dev/v1alpha1
kind: ModuleInstance
metadata:
  name: ` + name + `
  namespace: ` + namespace + `
  finalizers:
  - opmodel.dev/cleanup
  - example.com/foreign
spec:
  module:
    path: opmodel.dev/modules/e2e-fixture@v0
    version: v0.0.0
`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext, "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "applying armed ModuleInstance fixture: %s", out)
}
