package e2e

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Thin-editor apply and operator-owned delete, end to end against a live kind
// cluster with a REAL reconciling operator (enhancement 0006 slice C3).
//
// These require an operator that can actually reconcile — not just the CRDs.
// Bring one up with `task cluster:operator`, which installs the operator,
// points its --registry at the local registry over kind's docker network, and
// seeds the cluster Platform. Without it these tests FAIL rather than skip:
// the operator-owned paths are the security core of the ownership model, and
// silently skipping them is how they stayed unverified in the first place.

const (
	operatorOwnedNamespace = "default"
	operatorOwnedInstance  = "e2e-operator-owned"
	// podSelector is the label the render stamps on the instance's workloads.
	podSelector = "module-instance.opmodel.dev/name=" + operatorOwnedInstance
)

// repoPath resolves a path relative to the cli repo root (tests run in tests/e2e).
func repoPath(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", rel))
	require.NoError(t, err)
	return abs
}

// runOperatorOwnedOPM runs the CLI against the kind cluster with the repo's
// hermetic dev config, so it does not depend on the developer's ~/.opm.
func runOperatorOwnedOPM(t *testing.T, timeout time.Duration, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	kubeconfig := requireKindCluster(t)
	full := make([]string, 0, len(args)+6)
	full = append(full, args...)
	full = append(full,
		"--config", repoPath(t, "hack/opm-config.cue"),
		"--kubeconfig", kubeconfig,
		"--context", kindContext,
	)
	return runOPMWithEnv(t, t.TempDir(), homeDir, timeout, full...)
}

const (
	operatorNamespace    = "opm-operator-system"
	operatorDeployment   = "opm-operator-controller-manager"
	operatorControllerSA = "system:serviceaccount:" + operatorNamespace + ":" + operatorDeployment
	defaultSAFlag        = "--default-service-account="
)

// operatorRunning reports whether the controller Deployment has an available
// replica — the difference between "the manifest was applied" and "something is
// reconciling".
func operatorRunning(t *testing.T, kubeconfig string) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext,
		"get", "deployment", operatorDeployment, "-n", operatorNamespace,
		"-o", "jsonpath={.status.availableReplicas}").Output()
	if err != nil {
		return false
	}
	replicas := strings.TrimSpace(string(out))
	return replicas != "" && replicas != "0"
}

// requireReconcilingOperator fails (does not skip) when no operator is running,
// or when the operator's effective applier identity may not patch the workload
// kinds the fixtures render in the test namespace. The second check is the
// suite's own precondition for the dev grant in hack/kind-operator-rbac.yaml,
// which only `task cluster:operator` applies: without it an operator-owned test
// would otherwise fail 90 seconds later with the operator's own
// `cannot patch resource "services"` error, pointing away from the remedy.
func requireReconcilingOperator(t *testing.T, kubeconfig string) {
	t.Helper()
	if !operatorRunning(t, kubeconfig) {
		t.Fatal("no reconciling opm-operator in the cluster — run `task cluster:operator` " +
			"(installs the operator, wires its --registry to the local registry, seeds the cluster Platform)")
	}
	requireOperatorApplierGrant(t, kubeconfig)
}

// operatorApplierIdentity returns the identity the operator applies workloads
// as, read from the controller Deployment's container args. With
// --default-service-account=<sa> the operator impersonates <sa> in the
// instance's namespace (the fixtures live in operatorOwnedNamespace); without
// it, applies run as the controller's own ServiceAccount. Probing a constant
// would report a false denial the day the dev operator runs with the flag.
func operatorApplierIdentity(t *testing.T, kubeconfig string) string {
	t.Helper()

	out := kubectlOut(t, kubeconfig, "get", "deployment", operatorDeployment, "-n", operatorNamespace,
		"-o", `jsonpath={range .spec.template.spec.containers[*].args[*]}{@}{"\n"}{end}`)
	for arg := range strings.SplitSeq(out, "\n") {
		arg = strings.TrimSpace(arg)
		if sa, ok := strings.CutPrefix(arg, defaultSAFlag); ok && sa != "" {
			return "system:serviceaccount:" + operatorOwnedNamespace + ":" + sa
		}
	}
	return operatorControllerSA
}

// requireOperatorApplierGrant asks the cluster (SubjectAccessReview via
// `kubectl auth can-i --as=<identity>`) whether the operator's applier may
// patch the two representative kinds every fixture renders. An explicit "no"
// is a preparation mistake on a reachable cluster and FAILS; anything other
// than "yes"/"no" (kubectl error, test user lacking `impersonate`) means the
// check could not be performed and follows the reachability rule: skip.
func requireOperatorApplierGrant(t *testing.T, kubeconfig string) {
	t.Helper()

	identity := operatorApplierIdentity(t, kubeconfig)
	for _, resource := range []string{"services", "deployments.apps"} {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		// kubectl exits 1 for a denial AND for a failed check; stdout tells them apart.
		out, err := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext,
			"auth", "can-i", "patch", resource, "-n", operatorOwnedNamespace, "--as="+identity).Output()
		cancel()
		answer := strings.TrimSpace(string(out))
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			answer = strings.TrimSpace(answer + " " + string(exitErr.Stderr))
		}
		switch strings.TrimSpace(string(out)) {
		case "yes":
			continue
		case "no":
			t.Fatalf("operator applier %s may not patch %s in namespace %q: the dev grant "+
				"(hack/kind-operator-rbac.yaml) is missing; `opm operator install` does not apply it. "+
				"Run `task cluster:operator`.", identity, resource, operatorOwnedNamespace)
		default:
			t.Skipf("could not check whether %s may patch %s in namespace %q (kubectl auth can-i: %v: %s); "+
				"skipping operator e2e", identity, resource, operatorOwnedNamespace, err, answer)
		}
	}
}

func instanceField(t *testing.T, kubeconfig, jsonpath string) string {
	t.Helper()
	return kubectlOut(t, kubeconfig, "get", "moduleinstance", operatorOwnedInstance,
		"-n", operatorOwnedNamespace, "-o", "jsonpath="+jsonpath)
}

// waitForNoInstancePods blocks until every pod of the fixture is gone, so a
// following apply's "before" snapshot cannot catch a straggler.
func waitForNoInstancePods(t *testing.T, kubeconfig string) {
	t.Helper()

	for range 60 {
		out := kubectlOut(t, kubeconfig, "get", "pods", "-n", operatorOwnedNamespace, "-l", podSelector,
			"-o", "name")
		if out == "" {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatal("fixture pods still present after 60s; refusing to run with a dirty namespace")
}

// resetOperatorOwnedInstance removes the CR and any workloads it left behind,
// so each test starts clean regardless of how the previous one ended (a
// prune-less delete deliberately orphans workloads).
func resetOperatorOwnedInstance(t *testing.T, kubeconfig string) {
	t.Helper()
	stripAllModuleInstanceFinalizers(t, kubeconfig)
	kubectlDeleteIfExists(t, kubeconfig, "moduleinstance", operatorOwnedInstance, "-n", operatorOwnedNamespace)
	kubectlDeleteIfExists(t, kubeconfig, "deployment,service", "-n", operatorOwnedNamespace, "-l", podSelector)
	waitForNoInstancePods(t, kubeconfig)
}

// applyCLIOwned deploys the fixture as a CLI-owned instance and waits for its
// Deployment to roll out.
func applyCLIOwned(t *testing.T, kubeconfig string) {
	t.Helper()

	stdout, stderr, err := runOperatorOwnedOPM(t, 5*time.Minute,
		"instance", "apply", repoPath(t, "tests/e2e/testdata/operator-owned/instance.cue"))
	require.NoError(t, err, "CLI apply failed: %s%s", stdout, stderr)
	require.Equal(t, "cli", instanceField(t, kubeconfig, "{.spec.owner}"))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext,
		"rollout", "status", "deployment/"+operatorOwnedInstance+"-podinfo", "-n", operatorOwnedNamespace, "--timeout=90s").CombinedOutput()
	require.NoError(t, err, "waiting for the CLI-applied Deployment: %s", out)
}

// makeOperatorOwned turns the CLI-applied fixture into an operator-owned
// instance the way one is created outside the CLI: apply CLI-owned, patch
// spec.owner to "operator" with kubectl, then wait for the operator's
// reconcile of the new generation. The CLI offers no ownership transfer, so
// the patch stands in for the out-of-band creation the ownership model
// expects (kubectl, GitOps, the operator's own fixtures).
func makeOperatorOwned(t *testing.T, kubeconfig string) {
	t.Helper()

	applyCLIOwned(t, kubeconfig)
	kubectlOut(t, kubeconfig, "patch", "moduleinstance", operatorOwnedInstance, "-n", operatorOwnedNamespace,
		"--type=merge", "-p", `{"spec":{"owner":"operator"}}`)

	generation, err := strconv.ParseInt(instanceField(t, kubeconfig, "{.metadata.generation}"), 10, 64)
	require.NoError(t, err, "the patched CR should report a generation")

	// Same criterion as inventory.Record.ReadyFor: the Ready condition's
	// observedGeneration (falling back to the CR-level one when the condition
	// states none) has caught up to the patched generation, and Ready is True.
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		observed, _ := strconv.ParseInt(instanceField(t, kubeconfig,
			`{.status.conditions[?(@.type=="Ready")].observedGeneration}`), 10, 64)
		if observed == 0 {
			observed, _ = strconv.ParseInt(instanceField(t, kubeconfig,
				"{.status.observedGeneration}"), 10, 64)
		}
		ready := instanceField(t, kubeconfig,
			`{.status.conditions[?(@.type=="Ready")].status}`)
		if observed >= generation && ready == "True" {
			require.Equal(t, "operator", instanceField(t, kubeconfig, "{.spec.owner}"))
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("operator did not reconcile generation %d of %s/%s within 3m",
		generation, operatorOwnedNamespace, operatorOwnedInstance)
}

// swapFixtureReplicas rewrites one line of the fixture and returns a function
// restoring the original bytes, so a values-edit test leaves the repo tree as
// it found it even when the test fails.
func swapFixtureReplicas(t *testing.T, path, old, replacement string) func() {
	t.Helper()

	original, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(original), old, "fixture does not contain %q — has it drifted?", old)

	edited := strings.Replace(string(original), old, replacement, 1)
	require.NoError(t, os.WriteFile(path, []byte(edited), 0o644))

	return func() {
		if err := os.WriteFile(path, original, 0o644); err != nil {
			t.Errorf("restoring fixture %s: %v", path, err)
		}
	}
}

// TestE2E_ThinEditor_ValuesRoundTrip covers D18's thin-editor mode against a
// live operator: the CLI edits spec only, and the operator acts on the edit.
func TestE2E_ThinEditor_ValuesRoundTrip(t *testing.T) {
	kubeconfig := requireKindCluster(t)
	requireReconcilingOperator(t, kubeconfig)

	t.Cleanup(func() { resetOperatorOwnedInstance(t, kubeconfig) })
	resetOperatorOwnedInstance(t, kubeconfig)
	makeOperatorOwned(t, kubeconfig)
	require.Equal(t, "1", instanceField(t, kubeconfig, "{.spec.values.replicas}"))

	// Re-apply with a changed value. The fixture is edited in place and
	// restored, so the repo tree is left as found.
	fixture := repoPath(t, "tests/e2e/testdata/operator-owned/instance.cue")
	restore := swapFixtureReplicas(t, fixture, "\treplicas: 1", "\treplicas: 2")
	t.Cleanup(restore)

	stdout, stderr, err := runOperatorOwnedOPM(t, 10*time.Minute, "instance", "apply", fixture)
	require.NoError(t, err, "thin-editor apply failed: %s%s", stdout, stderr)

	combined := stdout + stderr
	assert.Contains(t, combined, "operator-managed", "the apply should announce thin-editor mode")
	assert.NotContains(t, combined, "applying 2 resources",
		"the CLI must not apply resources itself in thin-editor mode")

	assert.Equal(t, "2", instanceField(t, kubeconfig, "{.spec.values.replicas}"),
		"the values edit should have reached the CR")
	assert.Equal(t, "operator", instanceField(t, kubeconfig, "{.spec.owner}"),
		"a thin-editor apply must never rewrite spec.owner")
	assert.Equal(t, "True", instanceField(t, kubeconfig,
		`{.status.conditions[?(@.type=="Ready")].status}`))

	// The operator, not the CLI, acted on the edit.
	assert.Equal(t, "2", kubectlOut(t, kubeconfig, "get", "deployment", operatorOwnedInstance+"-podinfo",
		"-n", operatorOwnedNamespace, "-o", "jsonpath={.spec.replicas}"),
		"the operator should have scaled the Deployment to the new value")
}

// TestE2E_Delete_OperatorOwnedDelegates covers both outcomes of an
// operator-owned delete. Whether the workloads go away depends on spec.prune,
// which has no CRD default and which the CLI does not write — so the default
// for a CLI-created instance is that the operator orphans them. The CLI must
// report which of the two actually happened.
func TestE2E_Delete_OperatorOwnedDelegates(t *testing.T) {
	kubeconfig := requireKindCluster(t)
	requireReconcilingOperator(t, kubeconfig)
	t.Cleanup(func() { resetOperatorOwnedInstance(t, kubeconfig) })

	t.Run("without spec.prune the operator orphans the workloads, and the CLI says so", func(t *testing.T) {
		resetOperatorOwnedInstance(t, kubeconfig)
		makeOperatorOwned(t, kubeconfig)

		stdout, stderr, err := runOperatorOwnedOPM(t, 10*time.Minute,
			"instance", "delete", operatorOwnedInstance, "--force")
		require.NoError(t, err, "delete failed: %s%s", stdout, stderr)

		combined := stdout + stderr
		assert.Contains(t, combined, "left running", "the CLI must not claim a prune that did not happen")
		assert.Contains(t, combined, "spec.prune")

		assert.Empty(t, kubectlOut(t, kubeconfig, "get", "moduleinstance", operatorOwnedInstance,
			"-n", operatorOwnedNamespace, "--ignore-not-found", "-o", "name"),
			"the CR should be gone once the finalizer completed")
		assert.NotEmpty(t, kubectlOut(t, kubeconfig, "get", "deployment",
			"-n", operatorOwnedNamespace, "-l", podSelector, "-o", "name"),
			"the workloads should have been orphaned, not pruned")
	})

	t.Run("with spec.prune the operator removes the workloads", func(t *testing.T) {
		resetOperatorOwnedInstance(t, kubeconfig)
		makeOperatorOwned(t, kubeconfig)

		kubectlOut(t, kubeconfig, "patch", "moduleinstance", operatorOwnedInstance, "-n", operatorOwnedNamespace,
			"--type=merge", "-p", `{"spec":{"prune":true}}`)

		stdout, stderr, err := runOperatorOwnedOPM(t, 10*time.Minute,
			"instance", "delete", operatorOwnedInstance, "--force")
		require.NoError(t, err, "delete failed: %s%s", stdout, stderr)

		assert.Contains(t, stdout+stderr, "operator pruned")
		assert.Empty(t, kubectlOut(t, kubeconfig, "get", "deployment",
			"-n", operatorOwnedNamespace, "-l", podSelector, "-o", "name"),
			"the operator should have pruned the workloads")
	})
}
