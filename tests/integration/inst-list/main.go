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

	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/workflow/query"
	pkgcore "github.com/opmodel/cli/pkg/core"
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

	moduleName    = "test-module"
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

	listA, err := inventory.ListInventories(ctx, client, nsA)
	check("listing inventories in ns-a", err)
	if len(listA) != 2 {
		failf("expected 2 instances in %s, got %d", nsA, len(listA))
	}
	// Should be sorted: app-one, app-two
	if listA[0].InstanceMetadata.InstanceName != instanceOne {
		failf("expected first instance to be %q, got %q", instanceOne, listA[0].InstanceMetadata.InstanceName)
	}
	if listA[1].InstanceMetadata.InstanceName != instanceTwo {
		failf("expected second instance to be %q, got %q", instanceTwo, listA[1].InstanceMetadata.InstanceName)
	}
	fmt.Printf("   OK: %d instances in %s, correctly sorted\n", len(listA), nsA)

	listB, err := inventory.ListInventories(ctx, client, nsB)
	check("listing inventories in ns-b", err)
	if len(listB) != 1 {
		failf("expected 1 instance in %s, got %d", nsB, len(listB))
	}
	if listB[0].InstanceMetadata.InstanceName != instanceThree {
		failf("expected instance %q in ns-b, got %q", instanceThree, listB[0].InstanceMetadata.InstanceName)
	}
	fmt.Printf("   OK: %d instance in %s\n", len(listB), nsB)

	// ----------------------------------------------------------------
	// Scenario 2: List all namespaces
	// ----------------------------------------------------------------
	step(2, "List all namespaces returns instances from all test namespaces")

	listAll, err := inventory.ListInventories(ctx, client, "")
	check("listing inventories across all namespaces", err)

	// Find our test instances in the full list (there may be others from other tests)
	foundInstances := map[string]bool{}
	for _, inv := range listAll {
		switch inv.InstanceMetadata.InstanceName {
		case instanceOne, instanceTwo, instanceThree:
			foundInstances[inv.InstanceMetadata.InstanceName] = true
		}
	}
	if len(foundInstances) != 3 {
		failf("expected to find all 3 test instances across all namespaces, found %d: %v", len(foundInstances), foundInstances)
	}
	fmt.Printf("   OK: all 3 test instances found in all-namespaces list (%d total instances)\n", len(listAll))

	// Verify namespace is populated on each
	for _, inv := range listAll {
		if inv.InstanceMetadata.InstanceName == instanceOne {
			if inv.InstanceMetadata.InstanceNamespace != nsA {
				failf("expected namespace %q for %s, got %q", nsA, instanceOne, inv.InstanceMetadata.InstanceNamespace)
			}
		}
	}
	fmt.Println("   OK: namespace correctly populated on each instance")

	// ----------------------------------------------------------------
	// Scenario 2b: Ownership visibility in summaries
	// ----------------------------------------------------------------
	step(3, "Ownership visibility - controller and legacy inventories resolve correctly")

	controllerInv, err := inventory.GetInventory(ctx, client, instanceThree, nsB, instanceIDThree)
	check("reading inventory for controller scenario", err)
	controllerInv.CreatedBy = inventory.CreatedByController
	err = inventory.WriteInventory(ctx, client, controllerInv, moduleName, "", controllerInv.ModuleMetadata.Version, inventory.CreatedByCLI)
	check("rewriting controller-owned inventory", err)

	legacyInv, err := inventory.GetInventory(ctx, client, instanceTwo, nsA, instanceIDTwo)
	check("reading inventory for legacy scenario", err)
	legacyInv.CreatedBy = ""
	legacySecret, err := inventory.MarshalToSecret(legacyInv)
	check("marshaling legacy inventory", err)
	_, err = client.Clientset.CoreV1().Secrets(nsA).Update(ctx, legacySecret, metav1.UpdateOptions{})
	check("writing legacy inventory", err)

	controllerRead, err := inventory.GetInventory(ctx, client, instanceThree, nsB, instanceIDThree)
	check("reading controller-owned inventory", err)
	legacyRead, err := inventory.GetInventory(ctx, client, instanceTwo, nsA, instanceIDTwo)
	check("reading legacy inventory", err)

	controllerSummary := query.BuildInstanceSummary(controllerRead)
	legacySummary := query.BuildInstanceSummary(legacyRead)
	if controllerSummary.Owner != "controller" {
		failf("expected controller-owned summary owner, got %q", controllerSummary.Owner)
	}
	if legacySummary.Owner != "cli" {
		failf("expected legacy summary owner cli, got %q", legacySummary.Owner)
	}
	fmt.Println("   OK: controller-owned instances show owner=controller and legacy inventories resolve to owner=cli")

	// ----------------------------------------------------------------
	// Scenario 3: Empty namespace
	// ----------------------------------------------------------------
	step(4, "List in empty namespace returns zero results")

	listEmpty, err := inventory.ListInventories(ctx, client, nsEmpty)
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
	invOneRefresh, err := inventory.GetInventory(ctx, client, instanceOne, nsA, instanceIDOne)
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
		// Module name
		if inv.ModuleMetadata.Name != moduleName {
			failf("expected module name %q for %s, got %q", moduleName, inv.InstanceMetadata.InstanceName, inv.ModuleMetadata.Name)
		}

		// Instance ID
		switch inv.InstanceMetadata.InstanceName {
		case instanceOne:
			if inv.InstanceMetadata.InstanceID != instanceIDOne {
				failf("expected instance ID %q, got %q", instanceIDOne, inv.InstanceMetadata.InstanceID)
			}
		case instanceTwo:
			if inv.InstanceMetadata.InstanceID != instanceIDTwo {
				failf("expected instance ID %q, got %q", instanceIDTwo, inv.InstanceMetadata.InstanceID)
			}
		}

		if inv.ModuleMetadata.Version != moduleVersion {
			failf("expected version %q for %s, got %q", moduleVersion, inv.InstanceMetadata.InstanceName, inv.ModuleMetadata.Version)
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

// deployInstance creates ConfigMap resources and an inventory Secret for an instance.
func deployInstance(ctx context.Context, client *kubernetes.Client, name, ns, instanceID string, cmNames []string) {
	inv := buildInventory(name, ns, instanceID, cmNames, inventory.CreatedByCLI)
	writeInstance(ctx, client, name, ns, inv, cmNames)
}

func writeInstance(ctx context.Context, client *kubernetes.Client, name, ns string, inv *inventory.InstanceInventoryRecord, cmNames []string) {
	resources := buildResources(name, ns, inv.InstanceMetadata.InstanceID, cmNames)

	// Apply resources
	result, err := kubernetes.Apply(ctx, client, resources, name, kubernetes.ApplyOptions{})
	check(fmt.Sprintf("applying resources for %s", name), err)
	if len(result.Errors) > 0 {
		failf("apply errors for %s: %v", name, result.Errors[0])
	}

	inv.InstanceMetadata.LastTransitionTime = time.Now().UTC().Format(time.RFC3339)

	err = inventory.WriteInventory(ctx, client, inv, moduleName, "", inv.ModuleMetadata.Version, inventory.CreatedByCLI)
	check(fmt.Sprintf("writing inventory for %s", name), err)
}

func buildInventory(name, ns, instanceID string, cmNames []string, createdBy inventory.CreatedBy) *inventory.InstanceInventoryRecord {
	resources := buildResources(name, ns, instanceID, cmNames)
	entries := make([]inventory.InventoryEntry, len(resources))
	for i, r := range resources {
		entries[i] = inventory.NewEntryFromResource(r)
	}

	inv := &inventory.InstanceInventoryRecord{
		CreatedBy: createdBy,
		InstanceMetadata: inventory.InstanceMetadata{
			Kind:              "ModuleInstance",
			APIVersion:        inventory.APIVersionV1Alpha1,
			InstanceName:      name,
			InstanceNamespace: ns,
			InstanceID:        instanceID,
		},
		ModuleMetadata: inventory.ModuleMetadata{
			Kind:       "Module",
			APIVersion: inventory.APIVersionV1Alpha1,
			Name:       moduleName,
			Version:    moduleVersion,
		},
		Inventory: inventory.Inventory{
			Revision: 1,
			Digest:   inventory.ComputeDigest(entries),
			Count:    len(entries),
			Entries:  entries,
		},
	}
	return inv
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
	// Delete inventory Secrets
	for _, entry := range []struct {
		name, ns, id string
	}{
		{instanceOne, nsA, instanceIDOne},
		{instanceTwo, nsA, instanceIDTwo},
		{instanceThree, nsB, instanceIDThree},
	} {
		secretName := inventory.SecretName(entry.name, entry.id)
		_ = client.Clientset.CoreV1().Secrets(entry.ns).Delete(ctx, secretName, metav1.DeleteOptions{})
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
