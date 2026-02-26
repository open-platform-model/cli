//go:build ignore

// Integration test for opm mod list — inventory listing and health evaluation.
//
// Tests covered:
//   - Scenario 1: List in specific namespace returns correct releases
//   - Scenario 2: List all namespaces returns releases from all test namespaces
//   - Scenario 3: List in empty namespace returns zero results
//   - Scenario 4: Health status accuracy — missing resource shows NotReady
//   - Scenario 5: Metadata correctness — module name, version, release ID
//
// Requires a running kind cluster at context "kind-opm-dev".
// Run with: go run tests/integration/mod-list/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/core/modulerelease"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
)

const (
	clusterContext = "kind-opm-dev"

	nsA     = "opm-list-test-a"
	nsB     = "opm-list-test-b"
	nsEmpty = "opm-list-test-empty"

	releaseOne   = "app-one"
	releaseTwo   = "app-two"
	releaseThree = "app-three"

	releaseIDOne   = "11111111-aaaa-bbbb-cccc-000000000001"
	releaseIDTwo   = "22222222-aaaa-bbbb-cccc-000000000002"
	releaseIDThree = "33333333-aaaa-bbbb-cccc-000000000003"

	moduleName    = "test-module"
	moduleVersion = "1.0.0"
)

var configMapGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
var serviceGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}

func main() {
	ctx := context.Background()

	fmt.Println("=== OPM Mod List Integration Test ===")
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

	// Deploy test releases
	deployRelease(ctx, client, releaseOne, nsA, releaseIDOne, []string{"cm-one-a", "cm-one-b"})
	deployRelease(ctx, client, releaseTwo, nsA, releaseIDTwo, []string{"cm-two-a"})
	deployRelease(ctx, client, releaseThree, nsB, releaseIDThree, []string{"cm-three-a", "cm-three-b"})
	fmt.Println("OK: test releases deployed")
	fmt.Println()

	// ----------------------------------------------------------------
	// Scenario 1: List in specific namespace
	// ----------------------------------------------------------------
	step(1, "List in specific namespace returns correct releases")

	listA, err := inventory.ListInventories(ctx, client, nsA)
	check("listing inventories in ns-a", err)
	if len(listA) != 2 {
		failf("expected 2 releases in %s, got %d", nsA, len(listA))
	}
	// Should be sorted: app-one, app-two
	if listA[0].ReleaseMetadata.ReleaseName != releaseOne {
		failf("expected first release to be %q, got %q", releaseOne, listA[0].ReleaseMetadata.ReleaseName)
	}
	if listA[1].ReleaseMetadata.ReleaseName != releaseTwo {
		failf("expected second release to be %q, got %q", releaseTwo, listA[1].ReleaseMetadata.ReleaseName)
	}
	fmt.Printf("   OK: %d releases in %s, correctly sorted\n", len(listA), nsA)

	listB, err := inventory.ListInventories(ctx, client, nsB)
	check("listing inventories in ns-b", err)
	if len(listB) != 1 {
		failf("expected 1 release in %s, got %d", nsB, len(listB))
	}
	if listB[0].ReleaseMetadata.ReleaseName != releaseThree {
		failf("expected release %q in ns-b, got %q", releaseThree, listB[0].ReleaseMetadata.ReleaseName)
	}
	fmt.Printf("   OK: %d release in %s\n", len(listB), nsB)

	// ----------------------------------------------------------------
	// Scenario 2: List all namespaces
	// ----------------------------------------------------------------
	step(2, "List all namespaces returns releases from all test namespaces")

	listAll, err := inventory.ListInventories(ctx, client, "")
	check("listing inventories across all namespaces", err)

	// Find our test releases in the full list (there may be others from other tests)
	foundReleases := map[string]bool{}
	for _, inv := range listAll {
		switch inv.ReleaseMetadata.ReleaseName {
		case releaseOne, releaseTwo, releaseThree:
			foundReleases[inv.ReleaseMetadata.ReleaseName] = true
		}
	}
	if len(foundReleases) != 3 {
		failf("expected to find all 3 test releases across all namespaces, found %d: %v", len(foundReleases), foundReleases)
	}
	fmt.Printf("   OK: all 3 test releases found in all-namespaces list (%d total releases)\n", len(listAll))

	// Verify namespace is populated on each
	for _, inv := range listAll {
		if inv.ReleaseMetadata.ReleaseName == releaseOne {
			if inv.ReleaseMetadata.ReleaseNamespace != nsA {
				failf("expected namespace %q for %s, got %q", nsA, releaseOne, inv.ReleaseMetadata.ReleaseNamespace)
			}
		}
	}
	fmt.Println("   OK: namespace correctly populated on each release")

	// ----------------------------------------------------------------
	// Scenario 3: Empty namespace
	// ----------------------------------------------------------------
	step(3, "List in empty namespace returns zero results")

	listEmpty, err := inventory.ListInventories(ctx, client, nsEmpty)
	check("listing inventories in empty namespace", err)
	if len(listEmpty) != 0 {
		failf("expected 0 releases in %s, got %d", nsEmpty, len(listEmpty))
	}
	fmt.Printf("   OK: 0 releases in %s\n", nsEmpty)

	// ----------------------------------------------------------------
	// Scenario 4: Health status accuracy
	// ----------------------------------------------------------------
	step(4, "Health status accuracy — delete a resource, verify NotReady")

	// All resources are ConfigMaps (passive) so they should all be Ready initially
	live, missing, err := inventory.DiscoverResourcesFromInventory(ctx, client, listA[0])
	check("discovering resources for app-one", err)
	if len(missing) != 0 {
		failf("expected 0 missing resources initially, got %d", len(missing))
	}

	status, ready, total := kubernetes.QuickReleaseHealth(live, len(missing))
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
	invOneRefresh, err := inventory.GetInventory(ctx, client, releaseOne, nsA, releaseIDOne)
	check("refreshing app-one inventory", err)
	live2, missing2, err := inventory.DiscoverResourcesFromInventory(ctx, client, invOneRefresh)
	check("re-discovering resources for app-one", err)

	if len(missing2) != 1 {
		failf("expected 1 missing resource after delete, got %d", len(missing2))
	}

	status2, ready2, total2 := kubernetes.QuickReleaseHealth(live2, len(missing2))
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
	step(5, "Metadata correctness — module name, version, release ID")

	for _, inv := range listA {
		// Module name
		if inv.ModuleMetadata.Name != moduleName {
			failf("expected module name %q for %s, got %q", moduleName, inv.ReleaseMetadata.ReleaseName, inv.ModuleMetadata.Name)
		}

		// Release ID
		switch inv.ReleaseMetadata.ReleaseName {
		case releaseOne:
			if inv.ReleaseMetadata.ReleaseID != releaseIDOne {
				failf("expected release ID %q, got %q", releaseIDOne, inv.ReleaseMetadata.ReleaseID)
			}
		case releaseTwo:
			if inv.ReleaseMetadata.ReleaseID != releaseIDTwo {
				failf("expected release ID %q, got %q", releaseIDTwo, inv.ReleaseMetadata.ReleaseID)
			}
		}

		// Version from latest change
		if len(inv.Index) > 0 {
			change := inv.Changes[inv.Index[0]]
			if change == nil {
				failf("latest change entry is nil for %s", inv.ReleaseMetadata.ReleaseName)
			}
			if change.Source.Version != moduleVersion {
				failf("expected version %q for %s, got %q", moduleVersion, inv.ReleaseMetadata.ReleaseName, change.Source.Version)
			}
		}
	}
	fmt.Println("   OK: module name, release ID, and version all correct")

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

// deployRelease creates ConfigMap resources and an inventory Secret for a release.
func deployRelease(ctx context.Context, client *kubernetes.Client, name, ns, releaseID string, cmNames []string) {
	resources := buildResources(name, ns, releaseID, cmNames)
	meta := modulerelease.ReleaseMetadata{
		Name:      name,
		Namespace: ns,
		UUID:      releaseID,
	}

	// Apply resources
	result, err := kubernetes.Apply(ctx, client, resources, meta, kubernetes.ApplyOptions{})
	check(fmt.Sprintf("applying resources for %s", name), err)
	if len(result.Errors) > 0 {
		failf("apply errors for %s: %v", name, result.Errors[0])
	}

	// Build and write inventory
	entries := make([]inventory.InventoryEntry, len(resources))
	for i, r := range resources {
		entries[i] = inventory.NewEntryFromResource(r)
	}

	digest := inventory.ComputeManifestDigest(resources)
	source := inventory.ChangeSource{
		Path:        fmt.Sprintf("tests/integration/mod-list/%s", name),
		Version:     moduleVersion,
		ReleaseName: name,
	}
	changeID, changeEntry := inventory.PrepareChange(source, "", digest, entries)

	inv := &inventory.InventorySecret{
		ReleaseMetadata: inventory.ReleaseMetadata{
			Kind:               "ModuleRelease",
			APIVersion:         "core.opmodel.dev/v1alpha1",
			ReleaseName:        name,
			ReleaseNamespace:   ns,
			ReleaseID:          releaseID,
			LastTransitionTime: time.Now().UTC().Format(time.RFC3339),
		},
		Index:   []string{changeID},
		Changes: map[string]*inventory.ChangeEntry{changeID: changeEntry},
	}

	err = inventory.WriteInventory(ctx, client, inv, moduleName, "")
	check(fmt.Sprintf("writing inventory for %s", name), err)
}

func buildResources(relName, ns, releaseID string, cmNames []string) []*core.Resource {
	resources := make([]*core.Resource, len(cmNames))
	for i, cmName := range cmNames {
		labels := map[string]interface{}{
			"app.kubernetes.io/managed-by":    "open-platform-model",
			"module-release.opmodel.dev/name": relName,
			"module-release.opmodel.dev/uuid": releaseID,
		}
		resources[i] = &core.Resource{
			Object: &unstructured.Unstructured{Object: map[string]interface{}{
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
			}},
			Component: "config",
		}
	}
	return resources
}

func cleanup(ctx context.Context, client *kubernetes.Client) {
	// Delete inventory Secrets
	for _, entry := range []struct {
		name, ns, id string
	}{
		{releaseOne, nsA, releaseIDOne},
		{releaseTwo, nsA, releaseIDTwo},
		{releaseThree, nsB, releaseIDThree},
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
