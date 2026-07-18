//go:build ignore

// Integration test for opm instance list — inventory listing and health evaluation.
//
// Tests covered:
//   - Scenario 1: List in specific namespace returns correct instances
//   - Scenario 2: List all namespaces returns instances from all test namespaces
//   - Scenario 3: List in empty namespace returns zero results
//   - Scenario 4: Health status accuracy — missing resource shows NotReady
//   - Scenario 5: Metadata correctness — module name, version, instance ID
//
// Requires a running kind cluster at context "kind-opm-dev".
// Run with: go run tests/integration/inst-list/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/workflow/query"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

const (
	clusterContext = "kind-opm-dev"

	nsA     = "opm-list-test-a"
	nsB     = "opm-list-test-b"
	nsEmpty = "opm-list-test-empty"

	instanceOne   = "app-one"
	instanceTwo   = "app-two"
	instanceThree = "app-three"

	instanceIDOne   = "11111111-aaaa-bbbb-cccc-000000000001"
	instanceIDTwo   = "22222222-aaaa-bbbb-cccc-000000000002"
	instanceIDThree = "33333333-aaaa-bbbb-cccc-000000000003"

	modulePath    = "opmodel.dev/modules/test_module@v0"
	moduleVersion = "1.0.0"
)

var configMapGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
var serviceGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}

func main() {
	ctx := context.Background()

	fmt.Println("=== OPM Instance List Integration Test ===")
	fmt.Println()

	client, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Context: clusterContext,
	})
	check("creating Kubernetes client", err)
	fmt.Println("OK: client created")
	fmt.Println()

	// Clean state
	cleanup(ctx, client)
	ensureNamespaces(ctx, client)

	// Deploy test instances
	deployInstance(ctx, client, instanceOne, nsA, instanceIDOne, []string{"cm-one-a", "cm-one-b"})
	deployInstance(ctx, client, instanceTwo, nsA, instanceIDTwo, []string{"cm-two-a"})
	deployInstance(ctx, client, instanceThree, nsB, instanceIDThree, []string{"cm-three-a", "cm-three-b"})
	fmt.Println("OK: test instances deployed")
	fmt.Println()

	// ----------------------------------------------------------------
	// Scenario 1: List in specific namespace
	// ----------------------------------------------------------------
	step(1, "List in specific namespace returns correct instances")

	listA, err := inventory.ListRecords(ctx, client, nsA)
	check("listing inventories in ns-a", err)
	if len(listA) != 2 {
		failf("expected 2 instances in %s, got %d", nsA, len(listA))
	}
	// Should be sorted: app-one, app-two
	if listA[0].Name != instanceOne {
		failf("expected first instance to be %q, got %q", instanceOne, listA[0].Name)
	}
	if listA[1].Name != instanceTwo {
		failf("expected second instance to be %q, got %q", instanceTwo, listA[1].Name)
	}
	fmt.Printf("   OK: %d instances in %s, correctly sorted\n", len(listA), nsA)

	listB, err := inventory.ListRecords(ctx, client, nsB)
	check("listing inventories in ns-b", err)
	if len(listB) != 1 {
		failf("expected 1 instance in %s, got %d", nsB, len(listB))
	}
	if listB[0].Name != instanceThree {
		failf("expected instance %q in ns-b, got %q", instanceThree, listB[0].Name)
	}
	fmt.Printf("   OK: %d instance in %s\n", len(listB), nsB)

	// ----------------------------------------------------------------
	// Scenario 2: List all namespaces
	// ----------------------------------------------------------------
	step(2, "List all namespaces returns instances from all test namespaces")

	listAll, err := inventory.ListRecords(ctx, client, "")
	check("listing inventories across all namespaces", err)

	// Find our test instances in the full list (there may be others from other tests)
	foundInstances := map[string]bool{}
	for _, inv := range listAll {
		switch inv.Name {
		case instanceOne, instanceTwo, instanceThree:
			foundInstances[inv.Name] = true
		}
	}
	if len(foundInstances) != 3 {
		failf("expected to find all 3 test instances across all namespaces, found %d: %v", len(foundInstances), foundInstances)
	}
	fmt.Printf("   OK: all 3 test instances found in all-namespaces list (%d total instances)\n", len(listAll))

	// Verify namespace is populated on each
	for _, inv := range listAll {
		if inv.Name == instanceOne {
			if inv.Namespace != nsA {
				failf("expected namespace %q for %s, got %q", nsA, instanceOne, inv.Namespace)
			}
		}
	}
	fmt.Println("   OK: namespace correctly populated on each instance")

	// ----------------------------------------------------------------
	// Scenario 2b: Ownership visibility in summaries
	// ----------------------------------------------------------------
	step(3, "Ownership visibility - operator-owned and cli-owned instances resolve correctly")

	// Re-mark app-three as operator-owned by rewriting its spec.owner. Ownership
	// now derives from the CR's spec.owner, not a record's createdBy.
	err = inventory.ApplySpec(ctx, client, inventory.SpecInput{
		Name:          instanceThree,
		Namespace:     nsB,
		Owner:         inventory.OwnerOperator,
		ModulePath:    modulePath,
		ModuleVersion: moduleVersion,
	})
	check("marking app-three operator-owned", err)

	operatorRead, err := inventory.GetRecord(ctx, client, instanceThree, nsB)
	check("reading operator-owned inventory", err)
	cliRead, err := inventory.GetRecord(ctx, client, instanceTwo, nsA)
	check("reading cli-owned inventory", err)

	operatorSummary := query.BuildInstanceSummary(operatorRead)
	cliSummary := query.BuildInstanceSummary(cliRead)
	if operatorSummary.Owner != "operator" {
		failf("expected operator-owned summary owner, got %q", operatorSummary.Owner)
	}
	if cliSummary.Owner != "cli" {
		failf("expected cli summary owner, got %q", cliSummary.Owner)
	}
	fmt.Println("   OK: operator-owned instances show owner=operator and cli-owned show owner=cli")

	// ----------------------------------------------------------------
	// Scenario 3: Empty namespace
	// ----------------------------------------------------------------
	step(4, "List in empty namespace returns zero results")

	listEmpty, err := inventory.ListRecords(ctx, client, nsEmpty)
	check("listing inventories in empty namespace", err)
	if len(listEmpty) != 0 {
		failf("expected 0 instances in %s, got %d", nsEmpty, len(listEmpty))
	}
	fmt.Printf("   OK: 0 instances in %s\n", nsEmpty)

	// ----------------------------------------------------------------
	// Scenario 4: Health status accuracy
	// ----------------------------------------------------------------
	step(5, "Health status accuracy — delete a resource, verify NotReady")

	// All resources are ConfigMaps (passive) so they should all be Ready initially
	live, missing, err := inventory.DiscoverResourcesFromInventory(ctx, client, listA[0])
	check("discovering resources for app-one", err)
	if len(missing) != 0 {
		failf("expected 0 missing resources initially, got %d", len(missing))
	}

	status, ready, total := kubernetes.QuickInstanceHealth(live, len(missing))
	if status != kubernetes.HealthReady {
		failf("expected HealthReady initially, got %s", status)
	}
	if ready != 2 || total != 2 {
		failf("expected 2/2, got %d/%d", ready, total)
	}
	fmt.Println("   OK: app-one is Ready (2/2) initially")

	// Delete one ConfigMap to simulate a missing resource
	err = client.ResourceClient(configMapGVR, nsA).Delete(ctx, "cm-one-b", metav1.DeleteOptions{})
	check("deleting cm-one-b to simulate missing resource", err)

	// Re-read inventory and check health
	invOneRefresh, err := inventory.GetRecord(ctx, client, instanceOne, nsA)
	check("refreshing app-one inventory", err)
	live2, missing2, err := inventory.DiscoverResourcesFromInventory(ctx, client, invOneRefresh)
	check("re-discovering resources for app-one", err)

	if len(missing2) != 1 {
		failf("expected 1 missing resource after delete, got %d", len(missing2))
	}

	status2, ready2, total2 := kubernetes.QuickInstanceHealth(live2, len(missing2))
	if status2 != kubernetes.HealthNotReady {
		failf("expected HealthNotReady after delete, got %s", status2)
	}
	if ready2 != 1 || total2 != 2 {
		failf("expected 1/2 after delete, got %d/%d", ready2, total2)
	}
	fmt.Println("   OK: app-one is NotReady (1/2) after deleting cm-one-b")

	// ----------------------------------------------------------------
	// Scenario 5: Metadata correctness
	// ----------------------------------------------------------------
	step(6, "Metadata correctness — module name, version, instance ID")

	for _, inv := range listA {
		// Module path
		if inv.ModulePath != modulePath {
			failf("expected module path %q for %s, got %q", modulePath, inv.Name, inv.ModulePath)
		}

		// Instance ID
		switch inv.Name {
		case instanceOne:
			if inv.InstanceUUID != instanceIDOne {
				failf("expected instance ID %q, got %q", instanceIDOne, inv.InstanceUUID)
			}
		case instanceTwo:
			if inv.InstanceUUID != instanceIDTwo {
				failf("expected instance ID %q, got %q", instanceIDTwo, inv.InstanceUUID)
			}
		}

		if inv.ModuleVersion != moduleVersion {
			failf("expected version %q for %s, got %q", moduleVersion, inv.Name, inv.ModuleVersion)
		}
	}
	fmt.Println("   OK: module name, instance ID, and version all correct")

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

func ensureNamespaces(ctx context.Context, client *kubernetes.Client) {
	for _, ns := range []string{nsA, nsB, nsEmpty} {
		_, err := client.EnsureNamespace(ctx, ns, false)
		check(fmt.Sprintf("ensuring namespace %s", ns), err)
	}
}

// deployInstance applies ConfigMap resources and writes the ModuleInstance CR
// (spec + CLI-owned status) for an instance.
func deployInstance(ctx context.Context, client *kubernetes.Client, name, ns, instanceID string, cmNames []string) {
	resources := buildResources(name, ns, instanceID, cmNames)

	result, err := kubernetes.Apply(ctx, client, resources, name, kubernetes.ApplyOptions{})
	check(fmt.Sprintf("applying resources for %s", name), err)
	if len(result.Errors) > 0 {
		failf("apply errors for %s: %v", name, result.Errors[0])
	}

	entries := make([]inventory.InventoryEntry, len(resources))
	for i, r := range resources {
		entries[i] = inventory.NewEntryFromResource(r)
	}
	err = writeInventoryCR(ctx, client, name, ns, instanceID, inventory.OwnerCLI, 1, entries)
	check(fmt.Sprintf("writing inventory for %s", name), err)
}

// writeInventoryCR writes the ModuleInstance CR spec (with the given owner) and
// its CLI-owned status subset.
func writeInventoryCR(ctx context.Context, client *kubernetes.Client, name, ns, instanceID, owner string, revision int, entries []inventory.InventoryEntry) error {
	if err := inventory.ApplySpec(ctx, client, inventory.SpecInput{
		Name:          name,
		Namespace:     ns,
		Owner:         owner,
		ModulePath:    modulePath,
		ModuleVersion: moduleVersion,
	}); err != nil {
		return err
	}
	return inventory.ApplyStatus(ctx, client, inventory.StatusInput{
		Name:         name,
		Namespace:    ns,
		InstanceUUID: instanceID,
		Inventory: inventory.Inventory{
			Revision: revision,
			Digest:   inventory.ComputeDigest(entries),
			Count:    len(entries),
			Entries:  entries,
		},
		LastAppliedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func buildResources(relName, ns, instanceID string, cmNames []string) []*unstructured.Unstructured {
	resources := make([]*unstructured.Unstructured, len(cmNames))
	for i, cmName := range cmNames {
		labels := map[string]interface{}{
			pkgcore.LabelManagedBy:             pkgcore.LabelManagedByValue,
			"module-instance.opmodel.dev/name": relName,
			"module-instance.opmodel.dev/uuid": instanceID,
		}
		resources[i] = &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      cmName,
				"namespace": ns,
				"labels":    labels,
			},
			"data": map[string]interface{}{
				"key": fmt.Sprintf("value-%s", cmName),
			},
		}}
	}
	return resources
}

func cleanup(ctx context.Context, client *kubernetes.Client) {
	// Delete ModuleInstance CRs
	for _, entry := range []struct {
		name, ns, id string
	}{
		{instanceOne, nsA, instanceIDOne},
		{instanceTwo, nsA, instanceIDTwo},
		{instanceThree, nsB, instanceIDThree},
	} {
		_ = inventory.DeleteCR(ctx, client, entry.name, entry.ns)
	}

	// Delete ConfigMaps
	for _, cm := range []struct {
		name, ns string
	}{
		{"cm-one-a", nsA}, {"cm-one-b", nsA},
		{"cm-two-a", nsA},
		{"cm-three-a", nsB}, {"cm-three-b", nsB},
	} {
		_ = client.ResourceClient(configMapGVR, cm.ns).Delete(ctx, cm.name, metav1.DeleteOptions{})
	}

	// Delete test namespaces
	for _, ns := range []string{nsA, nsB, nsEmpty} {
		_ = client.Clientset.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
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

func failf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}
