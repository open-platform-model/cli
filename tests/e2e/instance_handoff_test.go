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
)

// Handoff, thin-editor apply, and operator-owned delete, end to end against a
// live kind cluster with a REAL reconciling operator (enhancement 0006 slice
// C3, tasks 5.2/5.3).
//
// These require an operator that can actually reconcile — not just the CRDs.
// Bring one up with `task cluster:operator`, which installs the operator,
// points its --registry at the local registry over kind's docker network, and
// seeds the cluster Platform. Without it these tests FAIL rather than skip:
// the adoption paths are the security core of handoff, and silently skipping
// them is how they stayed unverified in the first place.

const (
	handoffNamespace = "default"
	handoffInstance  = "e2e-handoff"
	// podSelector is the label the render stamps on the instance's workloads.
	podSelector = "module-instance.opmodel.dev/name=" + handoffInstance
)

// repoPath resolves a path relative to the cli repo root (tests run in tests/e2e).
func repoPath(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", rel))
	require.NoError(t, err)
	return abs
}

// runHandoffOPM runs the CLI against the kind cluster with the repo's hermetic
// dev config, so it does not depend on the developer's ~/.opm.
func runHandoffOPM(t *testing.T, timeout time.Duration, args ...string) (stdout, stderr string, err error) {
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

// operatorRunning reports whether the controller Deployment has an available
// replica — the difference between "the manifest was applied" and "something is
// reconciling".
func operatorRunning(t *testing.T, kubeconfig string) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext,
		"get", "deployment", "opm-operator-controller-manager", "-n", "opm-operator-system",
		"-o", "jsonpath={.status.availableReplicas}").Output()
	if err != nil {
		return false
	}
	replicas := strings.TrimSpace(string(out))
	return replicas != "" && replicas != "0"
}

// requireReconcilingOperator fails (does not skip) when no operator is running.
func requireReconcilingOperator(t *testing.T, kubeconfig string) {
	t.Helper()
	if !operatorRunning(t, kubeconfig) {
		t.Fatal("no reconciling opm-operator in the cluster — run `task cluster:operator` " +
			"(installs the operator, wires its --registry to the local registry, seeds the cluster Platform)")
	}
}

func instanceField(t *testing.T, kubeconfig, jsonpath string) string {
	t.Helper()
	return kubectlOut(t, kubeconfig, "get", "moduleinstance", handoffInstance,
		"-n", handoffNamespace, "-o", "jsonpath="+jsonpath)
}

// podUIDs returns the UIDs of the instance's live pods. Stable UIDs across a
// handoff are the observable behind "no workload changes" — a restart mints new
// ones.
//
// Pods still terminating from an earlier fixture are excluded. They linger for
// seconds after their Deployment is deleted, and counting one would make a
// perfectly stable handoff look like it had replaced a pod.
func podUIDs(t *testing.T, kubeconfig string) []string {
	t.Helper()

	out := kubectlOut(t, kubeconfig, "get", "pods", "-n", handoffNamespace, "-l", podSelector,
		"-o", `jsonpath={range .items[*]}{.metadata.uid}|{.metadata.deletionTimestamp}{"\n"}{end}`)

	var uids []string
	for _, line := range strings.Fields(out) {
		uid, deletionTimestamp, ok := strings.Cut(line, "|")
		if !ok || uid == "" || deletionTimestamp != "" {
			continue
		}
		uids = append(uids, uid)
	}
	return uids
}

// waitForNoInstancePods blocks until every pod of the fixture is gone, so a
// following apply's "before" snapshot cannot catch a straggler.
func waitForNoInstancePods(t *testing.T, kubeconfig string) {
	t.Helper()

	for range 60 {
		out := kubectlOut(t, kubeconfig, "get", "pods", "-n", handoffNamespace, "-l", podSelector,
			"-o", "name")
		if out == "" {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatal("fixture pods still present after 60s; refusing to run with a dirty namespace")
}

// inventoryEntryIDs returns the CR's inventory as comparable identity strings.
func inventoryEntryIDs(t *testing.T, kubeconfig string) []string {
	t.Helper()

	out := instanceField(t, kubeconfig,
		`{range .status.inventory.entries[*]}{.kind}/{.namespace}/{.name}{"\n"}{end}`)
	if out == "" {
		return nil
	}
	return strings.Fields(out)
}

// resetHandoffInstance removes the CR and any workloads it left behind, so each
// test starts clean regardless of how the previous one ended (a prune-less
// delete deliberately orphans workloads).
func resetHandoffInstance(t *testing.T, kubeconfig string) {
	t.Helper()
	stripAllModuleInstanceFinalizers(t, kubeconfig)
	kubectlDeleteIfExists(t, kubeconfig, "moduleinstance", handoffInstance, "-n", handoffNamespace)
	kubectlDeleteIfExists(t, kubeconfig, "deployment,service", "-n", handoffNamespace, "-l", podSelector)
	waitForNoInstancePods(t, kubeconfig)
}

// applyCLIOwned deploys the fixture as a CLI-owned instance and waits for its
// Deployment to roll out.
func applyCLIOwned(t *testing.T, kubeconfig string) {
	t.Helper()

	stdout, stderr, err := runHandoffOPM(t, 5*time.Minute,
		"instance", "apply", repoPath(t, "tests/e2e/testdata/handoff/instance.cue"))
	require.NoError(t, err, "CLI apply failed: %s%s", stdout, stderr)
	require.Equal(t, "cli", instanceField(t, kubeconfig, "{.spec.owner}"))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kindContext,
		"rollout", "status", "deployment/"+handoffInstance+"-podinfo", "-n", handoffNamespace, "--timeout=90s").CombinedOutput()
	require.NoError(t, err, "waiting for the CLI-applied Deployment: %s", out)
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

// TestE2E_Handoff_Adoption is task 5.2: a CLI-owned instance is handed to the
// operator, which adopts it without disturbing a single workload.
func TestE2E_Handoff_Adoption(t *testing.T) {
	kubeconfig := requireKindCluster(t)
	requireReconcilingOperator(t, kubeconfig)

	t.Cleanup(func() { resetHandoffInstance(t, kubeconfig) })
	resetHandoffInstance(t, kubeconfig)
	applyCLIOwned(t, kubeconfig)

	beforeUIDs := podUIDs(t, kubeconfig)
	beforeEntries := inventoryEntryIDs(t, kubeconfig)
	beforeRevision := instanceField(t, kubeconfig, "{.status.inventory.revision}")
	require.NotEmpty(t, beforeUIDs, "the fixture should be running pods before handoff")
	require.NotEmpty(t, beforeEntries, "the CLI apply should have recorded an inventory")
	require.Equal(t, "opm-cli", kubectlOut(t, kubeconfig, "get", "deployment", handoffInstance+"-podinfo",
		"-n", handoffNamespace, "-o", `jsonpath={.metadata.labels.app\.kubernetes\.io/managed-by}`))

	stdout, stderr, err := runHandoffOPM(t, 10*time.Minute, "instance", "handoff", handoffInstance)
	require.NoError(t, err, "handoff failed: %s%s", stdout, stderr)

	combined := stdout + stderr
	assert.Contains(t, combined, "verification render matches the deployed state")
	assert.Contains(t, combined, "adopted", "the report should state the adoption")

	// D40's inventory-stable criterion, observed from outside the CLI.
	assert.Equal(t, "operator", instanceField(t, kubeconfig, "{.spec.owner}"))
	assert.Equal(t, "True", instanceField(t, kubeconfig,
		`{.status.conditions[?(@.type=="Ready")].status}`))
	assert.ElementsMatch(t, beforeEntries, inventoryEntryIDs(t, kubeconfig),
		"handoff must not change which resources the instance owns")
	assert.NotEqual(t, beforeRevision, instanceField(t, kubeconfig, "{.status.inventory.revision}"),
		"the operator should have recorded a new inventory revision")

	// The relabel is expected; a restart is not.
	assert.ElementsMatch(t, beforeUIDs, podUIDs(t, kubeconfig),
		"pods must not be restarted by a handoff — same UIDs before and after")
	assert.Equal(t, "opm-controller", kubectlOut(t, kubeconfig, "get", "deployment", handoffInstance+"-podinfo",
		"-n", handoffNamespace, "-o", `jsonpath={.metadata.labels.app\.kubernetes\.io/managed-by}`),
		"the operator should have relabelled the resources it adopted")
}

// TestE2E_Handoff_DigestGate is task 5.3's core: the verification digest must
// abort a handoff, and --force must be the one thing that overrides it.
//
// The mismatch is induced by patching status.lastAppliedRenderDigest rather
// than republishing the module — it exercises the same comparison with no
// registry mutation.
func TestE2E_Handoff_DigestGate(t *testing.T) {
	kubeconfig := requireKindCluster(t)
	requireReconcilingOperator(t, kubeconfig)

	t.Cleanup(func() { resetHandoffInstance(t, kubeconfig) })
	resetHandoffInstance(t, kubeconfig)
	applyCLIOwned(t, kubeconfig)

	realDigest := instanceField(t, kubeconfig, "{.status.lastAppliedRenderDigest}")
	require.NotEmpty(t, realDigest, "the CLI apply should have recorded a render digest")

	const bogus = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	kubectlOut(t, kubeconfig, "patch", "moduleinstance", handoffInstance, "-n", handoffNamespace,
		"--subresource=status", "--type=merge",
		"-p", `{"status":{"lastAppliedRenderDigest":"`+bogus+`"}}`)

	t.Run("mismatch aborts and leaves ownership untouched", func(t *testing.T) {
		stdout, stderr, err := runHandoffOPM(t, 10*time.Minute, "instance", "handoff", handoffInstance)
		require.Error(t, err, "handoff must abort on a digest mismatch")

		combined := stdout + stderr
		assert.Contains(t, combined, "does not reproduce the deployed state")
		assert.Contains(t, combined, bogus, "the deployed digest should be shown")
		assert.Contains(t, combined, realDigest, "the published digest should be shown")
		assert.Equal(t, "cli", instanceField(t, kubeconfig, "{.spec.owner}"),
			"an aborted gate must not flip ownership")
	})

	t.Run("--force overrides the digest gate", func(t *testing.T) {
		stdout, stderr, err := runHandoffOPM(t, 10*time.Minute,
			"instance", "handoff", handoffInstance, "--force")
		require.NoError(t, err, "--force should proceed: %s%s", stdout, stderr)

		assert.Contains(t, stdout+stderr, "proceeding despite a verification digest mismatch")
		assert.Equal(t, "operator", instanceField(t, kubeconfig, "{.spec.owner}"))
	})
}

// TestE2E_ThinEditor_ValuesRoundTrip covers D18's thin-editor mode against a
// live operator: the CLI edits spec only, and the operator acts on the edit.
func TestE2E_ThinEditor_ValuesRoundTrip(t *testing.T) {
	kubeconfig := requireKindCluster(t)
	requireReconcilingOperator(t, kubeconfig)

	t.Cleanup(func() { resetHandoffInstance(t, kubeconfig) })
	resetHandoffInstance(t, kubeconfig)
	applyCLIOwned(t, kubeconfig)

	stdout, stderr, err := runHandoffOPM(t, 10*time.Minute, "instance", "handoff", handoffInstance)
	require.NoError(t, err, "handoff failed: %s%s", stdout, stderr)
	require.Equal(t, "1", instanceField(t, kubeconfig, "{.spec.values.replicas}"))

	// Re-apply with a changed value. The fixture is edited in place and
	// restored, so the repo tree is left as found.
	fixture := repoPath(t, "tests/e2e/testdata/handoff/instance.cue")
	restore := swapFixtureReplicas(t, fixture, "\treplicas: 1", "\treplicas: 2")
	t.Cleanup(restore)

	stdout, stderr, err = runHandoffOPM(t, 10*time.Minute, "instance", "apply", fixture)
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
	assert.Equal(t, "2", kubectlOut(t, kubeconfig, "get", "deployment", handoffInstance+"-podinfo",
		"-n", handoffNamespace, "-o", "jsonpath={.spec.replicas}"),
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
	t.Cleanup(func() { resetHandoffInstance(t, kubeconfig) })

	t.Run("without spec.prune the operator orphans the workloads, and the CLI says so", func(t *testing.T) {
		resetHandoffInstance(t, kubeconfig)
		applyCLIOwned(t, kubeconfig)
		_, _, err := runHandoffOPM(t, 10*time.Minute, "instance", "handoff", handoffInstance)
		require.NoError(t, err)

		stdout, stderr, err := runHandoffOPM(t, 10*time.Minute,
			"instance", "delete", handoffInstance, "--force")
		require.NoError(t, err, "delete failed: %s%s", stdout, stderr)

		combined := stdout + stderr
		assert.Contains(t, combined, "left running", "the CLI must not claim a prune that did not happen")
		assert.Contains(t, combined, "spec.prune")

		assert.Empty(t, kubectlOut(t, kubeconfig, "get", "moduleinstance", handoffInstance,
			"-n", handoffNamespace, "--ignore-not-found", "-o", "name"),
			"the CR should be gone once the finalizer completed")
		assert.NotEmpty(t, kubectlOut(t, kubeconfig, "get", "deployment",
			"-n", handoffNamespace, "-l", podSelector, "-o", "name"),
			"the workloads should have been orphaned, not pruned")
	})

	t.Run("with spec.prune the operator removes the workloads", func(t *testing.T) {
		resetHandoffInstance(t, kubeconfig)
		applyCLIOwned(t, kubeconfig)
		_, _, err := runHandoffOPM(t, 10*time.Minute, "instance", "handoff", handoffInstance)
		require.NoError(t, err)

		kubectlOut(t, kubeconfig, "patch", "moduleinstance", handoffInstance, "-n", handoffNamespace,
			"--type=merge", "-p", `{"spec":{"prune":true}}`)

		stdout, stderr, err := runHandoffOPM(t, 10*time.Minute,
			"instance", "delete", handoffInstance, "--force")
		require.NoError(t, err, "delete failed: %s%s", stdout, stderr)

		assert.Contains(t, stdout+stderr, "operator pruned")
		assert.Empty(t, kubectlOut(t, kubeconfig, "get", "deployment",
			"-n", handoffNamespace, "-l", podSelector, "-o", "name"),
			"the operator should have pruned the workloads")
	})
}

// TestE2E_Handoff_PreconditionRefusals covers the gates that abort before the
// flip. Each must leave spec.owner untouched — a gate that fails after changing
// ownership would be worse than no gate.
func TestE2E_Handoff_PreconditionRefusals(t *testing.T) {
	kubeconfig := requireKindCluster(t)
	requireReconcilingOperator(t, kubeconfig)

	t.Cleanup(func() { resetHandoffInstance(t, kubeconfig) })

	t.Run("--platform is rejected", func(t *testing.T) {
		_, stderr, err := runHandoffOPM(t, time.Minute,
			"instance", "handoff", handoffInstance, "--platform", "./platform.cue")
		require.Error(t, err)
		assert.Contains(t, stderr, "cluster Platform")
	})

	t.Run("local-provenance annotation refuses before any render", func(t *testing.T) {
		resetHandoffInstance(t, kubeconfig)
		applyCLIOwned(t, kubeconfig)
		kubectlOut(t, kubeconfig, "annotate", "moduleinstance", handoffInstance, "-n", handoffNamespace,
			"module-instance.opmodel.dev/source=local", "--overwrite")

		_, stderr, err := runHandoffOPM(t, 5*time.Minute, "instance", "handoff", handoffInstance)
		require.Error(t, err, "handoff must refuse a locally-rendered instance")
		assert.Contains(t, stderr, "local")
		assert.Contains(t, stderr, "publish", "the refusal must name the remedy")
		assert.Equal(t, "cli", instanceField(t, kubeconfig, "{.spec.owner}"),
			"a failed gate must not flip ownership")
	})

	t.Run("already operator-owned refuses, stating no reverse mode", func(t *testing.T) {
		resetHandoffInstance(t, kubeconfig)
		applyCLIOwned(t, kubeconfig)
		_, _, err := runHandoffOPM(t, 10*time.Minute, "instance", "handoff", handoffInstance)
		require.NoError(t, err)

		_, stderr, err := runHandoffOPM(t, 5*time.Minute, "instance", "handoff", handoffInstance)
		require.Error(t, err)
		assert.Contains(t, stderr, "forward-only")
	})

	t.Run("missing instance reports not found", func(t *testing.T) {
		resetHandoffInstance(t, kubeconfig)

		_, stderr, err := runHandoffOPM(t, 5*time.Minute, "instance", "handoff", "no-such-instance")
		require.Error(t, err)
		assert.Contains(t, stderr, "no ModuleInstance")
	})
}
