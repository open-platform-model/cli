package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// kindContext is the kind cluster context these tests run against. Matches
// TestMain's dummy config.cue and the workspace's `task cluster:create`.
const kindContext = "kind-opm-dev"

// pinnedInstallYAML locates the embedded operator manifest so the pinned
// image reference can be read directly, instead of duplicating it here where
// it would silently go stale after a `task operator:sync`.
const pinnedInstallYAMLRelPath = "../../internal/operator/dist/install.yaml"

var pinnedImageRe = regexp.MustCompile(`image:\s*(\S+)`)

// requireKindCluster skips the test if the kind-opm-dev cluster is not
// reachable, and returns the real (non-HOME-overridden) kubeconfig path.
func requireKindCluster(t *testing.T) string {
	t.Helper()

	realHome, err := os.UserHomeDir()
	require.NoError(t, err)
	kubeconfig := filepath.Join(realHome, ".kube", "config")
	if _, statErr := os.Stat(kubeconfig); statErr != nil {
		t.Skipf("no kubeconfig at %q; skipping operator e2e", kubeconfig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext,
		"cluster-info").Run(); err != nil {
		t.Skipf("kind cluster %q not reachable; run `task cluster:create`: %v", kindContext, err)
	}

	return kubeconfig
}

// pinnedOperatorImage reads the pinned image reference out of the embedded
// manifest, so the pull-reachability check always matches what `install`
// actually applies.
func pinnedOperatorImage(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(pinnedInstallYAMLRelPath)
	require.NoError(t, err)

	m := pinnedImageRe.FindSubmatch(data)
	require.NotNil(t, m, "could not find image: reference in %s", pinnedInstallYAMLRelPath)
	return string(m[1])
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

// resetOperatorCluster removes everything the operator manifest can create,
// including CRDs and the Namespace (which `opm operator uninstall` itself
// deliberately never touches), so each e2e run starts from a clean slate.
func resetOperatorCluster(t *testing.T, kubeconfig string) {
	t.Helper()
	stripAllModuleInstanceFinalizers(t, kubeconfig)
	kubectlDeleteIfExists(t, kubeconfig, "moduleinstances.opmodel.dev", "--all-namespaces", "--all")
	kubectlDeleteIfExists(t, kubeconfig, "crd", "moduleinstances.opmodel.dev", "modulepackages.opmodel.dev", "platforms.opmodel.dev")
	kubectlDeleteIfExists(t, kubeconfig, "namespace", "opm-operator-system")
	kubectlDeleteIfExists(t, kubeconfig, "clusterrole", "opm-cli-user",
		"opm-operator-manager-role", "opm-operator-metrics-auth-role", "opm-operator-metrics-reader",
		"opm-operator-moduleinstance-admin-role", "opm-operator-moduleinstance-editor-role", "opm-operator-moduleinstance-viewer-role")
	kubectlDeleteIfExists(t, kubeconfig, "clusterrolebinding", "opm-cli-user",
		"opm-operator-manager-rolebinding", "opm-operator-metrics-auth-rolebinding")
}

// TestE2E_Operator_InstallUninstallLifecycle exercises the full
// `opm operator install`/`uninstall` lifecycle against a real kind cluster:
// full install (waiting for readiness if the pinned image is reachable, else
// just CRD-established + Deployment-created — design risk 1), idempotent
// re-install, uninstall's finalizer guard and its --remove-finalizers
// override (CRDs/Namespace surviving throughout), and a solo --crds-only
// install onto a freshly reset cluster.
func TestE2E_Operator_InstallUninstallLifecycle(t *testing.T) {
	kubeconfig := requireKindCluster(t)
	image := pinnedOperatorImage(t)

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
		assertResourceExists(t, kubeconfig, "namespace", "", "opm-operator-system")
		assertResourceExists(t, kubeconfig, "deployment", "opm-operator-system", "opm-operator-controller-manager")
	})

	t.Run("idempotent re-install reports unchanged", func(t *testing.T) {
		_, stderr, err := runOPMWithEnv(t, tmpDir, homeDir, 30*time.Second,
			"operator", "install", "--crds-only",
			"--kubeconfig", kubeconfig, "--context", kindContext, "--timeout", "15s")
		require.NoError(t, err, "stderr=%s", stderr)
		assert.Equal(t, 3, strings.Count(stderr, "unchanged"), "stderr=%s", stderr)
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

		// CRDs and the Namespace survive uninstall.
		assertCRDEstablished(t, kubeconfig, "moduleinstances.opmodel.dev")
		assertResourceExists(t, kubeconfig, "namespace", "", "opm-operator-system")
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
