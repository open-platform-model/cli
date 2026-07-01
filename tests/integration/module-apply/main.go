//go:build ignore

// Integration test for `opm module apply`.
//
// Exercises the full synthesis-to-apply pipeline against the local
// kind-opm-dev cluster using the testdata module fixture.
//
// Scenarios:
//   - First apply: resources are created and an inventory Secret is written
//     keyed on the synthetic instance UUID.
//   - Idempotent re-apply: no new resources, inventory revision increments.
//   - Dry-run: no inventory Secret, no resource creation.
//   - Prune: re-applying with values_api_off.cue removes the api resources
//     and bumps the inventory revision.
//
// Requires:
//   - kind cluster at context "kind-opm-dev"
//   - OPM_REGISTRY (catalog must be reachable so the testdata module
//     dependencies resolve)
//
// Run with: go run tests/integration/module-apply/main.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
)

const (
	clusterContext = "kind-opm-dev"
	testNamespace  = "opm-module-apply-itest"

	// Synthetic instance name: testdata module is "module-apply-itest" → "<module>-debug" by default.
	// We pass --name explicitly to keep the name deterministic in the assertions.
	instanceName = "module-apply-itest"

	moduleFixture       = "tests/integration/module-apply/testdata"
	valuesApiOffFixture = "tests/integration/module-apply/testdata/values_api_off.cue"

	binaryPath = "bin/opm-module-apply-itest"
)

var (
	deploymentGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	serviceGVR    = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
)

func main() {
	ctx := context.Background()

	fmt.Println("=== OPM Module Apply Integration Test ===")
	fmt.Println()

	buildBinary()

	client, err := kubernetes.NewClient(kubernetes.ClientOptions{Context: clusterContext})
	check("creating Kubernetes client", err)
	fmt.Println("OK: client created")
	fmt.Println()

	// Clean state from any prior run, then ensure namespace exists.
	cleanup(ctx, client)
	_, err = client.EnsureNamespace(ctx, testNamespace, false)
	check("ensuring test namespace", err)

	// ----------------------------------------------------------------
	// Scenario 1: First apply — resources and inventory created
	// ----------------------------------------------------------------
	step(1, "First apply — resources + inventory created")

	stdout, stderr, exitCode := runModuleApply([]string{
		moduleFixture,
		"--context", clusterContext,
		"--name", instanceName,
		"-n", testNamespace,
	})
	if exitCode != 0 {
		failf("first apply exited %d:\n%s\n%s", exitCode, stdout, stderr)
	}
	fmt.Printf("   OK: apply succeeded (exit 0)\n")

	instanceID := waitForInstanceUUID(ctx, client, instanceName, testNamespace)
	fmt.Printf("   OK: instance UUID resolved from inventory: %s\n", instanceID)

	inv1, err := inventory.GetInventory(ctx, client, instanceName, testNamespace, instanceID)
	check("reading inventory after first apply", err)
	if inv1 == nil {
		failf("expected non-nil inventory after first apply")
	}
	if inv1.Inventory.Revision != 1 {
		failf("expected revision 1 after first apply, got %d", inv1.Inventory.Revision)
	}
	if len(inv1.Inventory.Entries) == 0 {
		failf("expected non-empty inventory entries after first apply")
	}

	// debugValues renders both web and api components.
	hasWeb := containsComponent(inv1.Inventory.Entries, "web")
	hasAPI := containsComponent(inv1.Inventory.Entries, "api")
	if !hasWeb || !hasAPI {
		failf("expected both web and api components in inventory; web=%v api=%v", hasWeb, hasAPI)
	}
	firstDigest := inv1.Inventory.Digest
	firstEntries := inv1.Inventory.Entries
	fmt.Printf("   OK: inventory revision=1, %d entries, both web and api present\n", len(firstEntries))

	// ChangeDescriptor must record the absolute module-directory path.
	wd, _ := os.Getwd()
	wantPath, _ := filepath.Abs(filepath.Join(wd, moduleFixture))
	if !filepath.IsAbs(wantPath) {
		failf("test bug: could not compute absolute fixture path")
	}
	// Inventory Secret stores ChangeDescriptor; not directly exposed via the
	// top-level inventory record fields. The path round-trip is covered by
	// the unit test for runModuleApply; here we just assert the inventory
	// secret exists by name, which is a strong proxy.
	secretName := inventory.SecretName(instanceName, instanceID)
	if _, err := client.Clientset.CoreV1().Secrets(testNamespace).Get(ctx, secretName, metav1.GetOptions{}); err != nil {
		failf("inventory Secret %q not found: %v", secretName, err)
	}
	fmt.Printf("   OK: inventory Secret %q exists\n", secretName)

	// ----------------------------------------------------------------
	// Scenario 2: Idempotent re-apply
	// ----------------------------------------------------------------
	step(2, "Idempotent re-apply — revision++ and digest unchanged")

	stdout, stderr, exitCode = runModuleApply([]string{
		moduleFixture,
		"--context", clusterContext,
		"--name", instanceName,
		"-n", testNamespace,
	})
	if exitCode != 0 {
		failf("re-apply exited %d:\n%s\n%s", exitCode, stdout, stderr)
	}

	inv2, err := inventory.GetInventory(ctx, client, instanceName, testNamespace, instanceID)
	check("reading inventory after re-apply", err)
	if inv2.Inventory.Revision <= inv1.Inventory.Revision {
		failf("expected revision to increment after re-apply, got %d (was %d)", inv2.Inventory.Revision, inv1.Inventory.Revision)
	}
	if inv2.Inventory.Digest != firstDigest {
		failf("expected digest to be stable on idempotent re-apply: was %q now %q", firstDigest, inv2.Inventory.Digest)
	}
	fmt.Printf("   OK: revision %d→%d, digest stable\n", inv1.Inventory.Revision, inv2.Inventory.Revision)

	// ----------------------------------------------------------------
	// Scenario 3: Dry-run leaves no inventory mutation
	// ----------------------------------------------------------------
	step(3, "Dry-run — no resource changes, no inventory mutation")

	revBeforeDryRun := inv2.Inventory.Revision
	stdout, stderr, exitCode = runModuleApply([]string{
		moduleFixture,
		"--context", clusterContext,
		"--name", instanceName,
		"-n", testNamespace,
		"--dry-run",
	})
	if exitCode != 0 {
		failf("dry-run apply exited %d:\n%s\n%s", exitCode, stdout, stderr)
	}

	inv3, err := inventory.GetInventory(ctx, client, instanceName, testNamespace, instanceID)
	check("reading inventory after dry-run", err)
	if inv3.Inventory.Revision != revBeforeDryRun {
		failf("dry-run mutated inventory: revision %d → %d", revBeforeDryRun, inv3.Inventory.Revision)
	}
	fmt.Printf("   OK: inventory revision unchanged (%d)\n", inv3.Inventory.Revision)

	// ----------------------------------------------------------------
	// Scenario 4: Prune — re-apply with api disabled
	// ----------------------------------------------------------------
	step(4, "Prune — re-apply with values_api_off drops api resources")

	stdout, stderr, exitCode = runModuleApply([]string{
		moduleFixture,
		"--context", clusterContext,
		"--name", instanceName,
		"-n", testNamespace,
		"-f", valuesApiOffFixture,
	})
	if exitCode != 0 {
		failf("prune-mode apply exited %d:\n%s\n%s", exitCode, stdout, stderr)
	}

	inv4, err := inventory.GetInventory(ctx, client, instanceName, testNamespace, instanceID)
	check("reading inventory after prune-mode apply", err)
	if inv4.Inventory.Revision <= inv3.Inventory.Revision {
		failf("expected revision to increment after prune-mode re-apply, got %d (was %d)", inv4.Inventory.Revision, inv3.Inventory.Revision)
	}
	if containsComponent(inv4.Inventory.Entries, "api") {
		failf("expected api component to be absent after prune-mode re-apply, but inventory still tracks it")
	}
	if !containsComponent(inv4.Inventory.Entries, "web") {
		failf("expected web component to remain after prune-mode re-apply")
	}
	fmt.Printf("   OK: inventory now tracks %d entries, api component pruned\n", len(inv4.Inventory.Entries))

	// Verify the api Deployment is actually gone from the cluster.
	apiDeploymentName := instanceName + "-api"
	waitForResourceAbsent(ctx, client, deploymentGVR, testNamespace, apiDeploymentName)
	fmt.Printf("   OK: Deployment/%s is deleted (404)\n", apiDeploymentName)

	// ----------------------------------------------------------------
	// Cleanup
	// ----------------------------------------------------------------
	fmt.Println()
	fmt.Println("Cleaning up...")
	cleanup(ctx, client)
	fmt.Println("OK: cleanup complete")

	fmt.Println()
	fmt.Println("=== ALL SCENARIOS PASSED ===")
}

// ----------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------

func buildBinary() {
	fmt.Println("Building opm binary for integration test...")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/opm")
	out, err := cmd.CombinedOutput()
	if err != nil {
		failf("building opm: %v\n%s", err, out)
	}
	fmt.Printf("OK: binary built at %s\n", binaryPath)
	fmt.Println()
}

// runModuleApply invokes `./bin/<binary> module apply <args...>` and returns stdout,
// stderr, and the exit code.
func runModuleApply(args []string) (stdout, stderr string, exitCode int) {
	full := append([]string{"module", "apply"}, args...)
	cmd := exec.Command("./"+binaryPath, full...)
	cmd.Env = append(os.Environ(),
		"OPM_REGISTRY="+os.Getenv("OPM_REGISTRY"),
	)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		failf("running opm: %v", err)
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// waitForInstanceUUID polls inventory.FindInventoryByInstanceName until the
// inventory Secret exists, then returns the instance UUID. The synthetic UUID
// is computed in CUE; we read it back from the cluster rather than predicting it.
func waitForInstanceUUID(ctx context.Context, client *kubernetes.Client, name, ns string) string {
	deadline := time.Now().Add(15 * time.Second)
	for {
		inv, err := inventory.FindInventoryByInstanceName(ctx, client, name, ns)
		if err == nil && inv != nil {
			return inv.InstanceMetadata.InstanceID
		}
		if time.Now().After(deadline) {
			failf("timed out waiting for inventory Secret for instance %q in %q", name, ns)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func containsComponent(entries []inventory.InventoryEntry, component string) bool {
	for _, e := range entries {
		// Inventory entry names are "<instance>-<component>" by convention.
		if strings.Contains(e.Name, "-"+component) {
			return true
		}
	}
	return false
}

func waitForResourceAbsent(ctx context.Context, client *kubernetes.Client, gvr schema.GroupVersionResource, ns, name string) {
	deadline := time.Now().Add(15 * time.Second)
	for {
		_, err := client.ResourceClient(gvr, ns).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return
		}
		if err != nil && !apierrors.IsNotFound(err) {
			failf("waiting for %s/%s to be absent: %v", gvr.Resource, name, err)
		}
		if time.Now().After(deadline) {
			failf("timed out waiting for %s/%s to be deleted", gvr.Resource, name)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func cleanup(ctx context.Context, client *kubernetes.Client) {
	// Delete inventory Secrets in the test namespace.
	secrets, err := client.Clientset.CoreV1().Secrets(testNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "opmodel.dev/component=inventory",
	})
	if err == nil {
		for _, s := range secrets.Items {
			_ = client.Clientset.CoreV1().Secrets(testNamespace).Delete(ctx, s.Name, metav1.DeleteOptions{})
		}
	}
	// Delete the test namespace; this cascades to all owned resources.
	_ = client.Clientset.CoreV1().Namespaces().Delete(ctx, testNamespace, metav1.DeleteOptions{})
	// Best-effort wait for namespace removal so the next run starts clean.
	deadline := time.Now().Add(30 * time.Second)
	for {
		_, err := client.Clientset.CoreV1().Namespaces().Get(ctx, testNamespace, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func step(n int, desc string) {
	fmt.Printf("\n--- Step %d: %s\n", n, desc)
}

func check(label string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %s: %v\n", label, err)
		os.Exit(1)
	}
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}
