//go:build ignore

// Integration test for inventory-first diff, delete, and status operations (RFC-0001).
//
// Tests covered:
//   - 6.6: Diff without inventory — falls back to label-scan, orphans detected correctly
//   - 6.5: Diff with inventory — orphan computed via set-difference
//   - 6.7: Delete with inventory — no Endpoints deleted, inventory Secret deleted last
//   - 6.8: Status with inventory — missing resource shown with "Missing" status
//
// Requires a running kind cluster at context "kind-opm-dev".
// Run with: go run tests/integration/inventory-ops/main.go
package main

import (
	"context"
	"fmt"
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
)

const (
	clusterContext = "kind-opm-dev"
	releaseName    = "opm-inv-ops-test"
	namespace      = "default"
	// Fixed release UUID for deterministic inventory Secret naming.
	releaseID = "b2c3d4e5-2222-3333-4444-bbccddeeff00"

	modulePath    = "tests/integration/inventory-ops"
	moduleVersion = "0.1.0"
)

var (
	configMapGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	serviceGVR   = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
)

func main() {
	ctx := context.Background()

	fmt.Println("=== OPM Inventory Ops Integration Test ===")
	fmt.Println()

	client, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Context: clusterContext,
	})
	check("creating Kubernetes client", err)
	fmt.Println("OK: client created")
	fmt.Println()

	// ----------------------------------------------------------------
	// Scenario 6.6: Diff without inventory — label-scan fallback
	// ----------------------------------------------------------------
	step(1, "6.6: Diff without inventory — label-scan fallback")

	cleanup(ctx, client)

	// Apply resources without writing an inventory Secret.
	res66 := buildResources([]string{"cm-a"})
	_, err = kubernetes.Apply(ctx, client, res66, moduleMeta(), kubernetes.ApplyOptions{})
	check("applying resources for 6.6", err)
	fmt.Println("   OK: resources applied (no inventory)")

	// Diff with same render set — no DiffOptions (nil InventoryLive = label-scan path).
	comparer := kubernetes.NewComparer()
	diffResult66, err := kubernetes.Diff(ctx, client, res66, moduleMeta(), comparer)
	check("diffing without inventory", err)
	if diffResult66.Orphaned != 0 {
		failf("6.6: expected 0 orphans on label-scan fallback, got %d", diffResult66.Orphaned)
	}
	if diffResult66.Unchanged != 1 {
		failf("6.6: expected 1 unchanged resource, got %d", diffResult66.Unchanged)
	}
	fmt.Printf("   OK: diff result: %d unchanged, %d orphaned (label-scan fallback)\n",
		diffResult66.Unchanged, diffResult66.Orphaned)

	cleanup(ctx, client)

	// ----------------------------------------------------------------
	// Scenario 6.5: Diff with inventory — orphan from set-difference
	// ----------------------------------------------------------------
	step(2, "6.5: Diff with inventory — orphan from set-difference")

	// Apply both cm-a and svc-a; write inventory tracking both.
	res65cm := buildResources([]string{"cm-a"})
	res65svc := buildServiceResources([]string{"svc-a"})
	res65all := append(res65cm, res65svc...)

	_, err = kubernetes.Apply(ctx, client, res65all, moduleMeta(), kubernetes.ApplyOptions{})
	check("applying resources for 6.5", err)
	fmt.Println("   OK: cm-a and svc-a applied")

	inv65 := buildInventory(res65all)
	err = inventory.WriteInventory(ctx, client, inv65)
	check("writing inventory for 6.5", err)
	fmt.Println("   OK: inventory written tracking [cm-a, svc-a]")

	// Read back and discover live resources from inventory.
	readInv65, err := inventory.GetInventory(ctx, client, releaseName, namespace, releaseID)
	check("reading inventory for 6.5", err)
	liveResources65, missing65, err := inventory.DiscoverResourcesFromInventory(ctx, client, readInv65)
	check("discovering resources from inventory for 6.5", err)
	if len(missing65) != 0 {
		failf("6.5: expected 0 missing resources, got %d", len(missing65))
	}
	if len(liveResources65) != 2 {
		failf("6.5: expected 2 live resources, got %d", len(liveResources65))
	}
	fmt.Println("   OK: discovered 2 live resources from inventory")

	// Diff: local render has only cm-a; svc-a is an orphan.
	diffResult65, err := kubernetes.Diff(ctx, client, res65cm, moduleMeta(), comparer,
		kubernetes.DiffOptions{InventoryLive: liveResources65})
	check("diffing with inventory for 6.5", err)
	if diffResult65.Orphaned != 1 {
		failf("6.5: expected 1 orphan (svc-a), got %d", diffResult65.Orphaned)
	}
	if diffResult65.Unchanged != 1 {
		failf("6.5: expected 1 unchanged resource (cm-a), got %d", diffResult65.Unchanged)
	}
	fmt.Printf("   OK: diff result: %d unchanged, %d orphaned (inventory-first)\n",
		diffResult65.Unchanged, diffResult65.Orphaned)

	cleanup(ctx, client)

	// ----------------------------------------------------------------
	// Scenario 6.7: Delete with inventory — no Endpoints, inventory Secret deleted last
	// ----------------------------------------------------------------
	step(3, "6.7: Delete with inventory — no Endpoints deleted, inventory Secret deleted last")

	// Apply cm-a and svc-a; write inventory.
	res67cm := buildResources([]string{"cm-a"})
	res67svc := buildServiceResources([]string{"svc-a"})
	res67all := append(res67cm, res67svc...)

	_, err = kubernetes.Apply(ctx, client, res67all, moduleMeta(), kubernetes.ApplyOptions{})
	check("applying resources for 6.7", err)
	fmt.Println("   OK: cm-a and svc-a applied")

	inv67 := buildInventory(res67all)
	err = inventory.WriteInventory(ctx, client, inv67)
	check("writing inventory for 6.7", err)
	fmt.Println("   OK: inventory written")

	// Discover live resources.
	readInv67, err := inventory.GetInventory(ctx, client, releaseName, namespace, releaseID)
	check("reading inventory for 6.7", err)
	liveResources67, _, err := inventory.DiscoverResourcesFromInventory(ctx, client, readInv67)
	check("discovering resources from inventory for 6.7", err)

	invSecretName67 := inventory.SecretName(releaseName, releaseID)

	// Delete with inventory-first path.
	deleteResult67, err := kubernetes.Delete(ctx, client, kubernetes.DeleteOptions{
		ReleaseName:              releaseName,
		Namespace:                namespace,
		InventoryLive:            liveResources67,
		InventorySecretName:      invSecretName67,
		InventorySecretNamespace: namespace,
	})
	check("deleting with inventory for 6.7", err)

	// Only cm-a and svc-a should be deleted (not auto-generated Endpoints).
	if deleteResult67.Deleted != 2 {
		failf("6.7: expected 2 deleted resources (cm-a + svc-a), got %d", deleteResult67.Deleted)
	}
	fmt.Printf("   OK: %d resources deleted (no Endpoints)\n", deleteResult67.Deleted)

	// Verify the inventory Secret is gone (deleted last).
	_, errInvGet := client.Clientset.CoreV1().Secrets(namespace).Get(ctx, invSecretName67, metav1.GetOptions{})
	if !apierrors.IsNotFound(errInvGet) {
		failf("6.7: expected inventory Secret to be deleted, but got: %v", errInvGet)
	}
	fmt.Println("   OK: inventory Secret deleted last (404 confirmed)")

	// cleanup is no-op here since delete already removed everything
	cleanup(ctx, client)

	// ----------------------------------------------------------------
	// Scenario 6.8: Status with inventory — missing resource shown
	// ----------------------------------------------------------------
	step(4, "6.8: Status with inventory — missing resource shown with Missing status")

	// Apply cm-a and svc-a; write inventory.
	res68cm := buildResources([]string{"cm-a"})
	res68svc := buildServiceResources([]string{"svc-a"})
	res68all := append(res68cm, res68svc...)

	_, err = kubernetes.Apply(ctx, client, res68all, moduleMeta(), kubernetes.ApplyOptions{})
	check("applying resources for 6.8", err)
	fmt.Println("   OK: cm-a and svc-a applied")

	inv68 := buildInventory(res68all)
	err = inventory.WriteInventory(ctx, client, inv68)
	check("writing inventory for 6.8", err)
	fmt.Println("   OK: inventory written")

	// Manually delete svc-a to simulate a missing resource.
	err = client.ResourceClient(serviceGVR, namespace).Delete(ctx, "svc-a", metav1.DeleteOptions{})
	check("manually deleting svc-a for 6.8", err)
	fmt.Println("   OK: svc-a manually deleted (simulating missing resource)")

	// Discover resources from inventory — svc-a should be in missing list.
	readInv68, err := inventory.GetInventory(ctx, client, releaseName, namespace, releaseID)
	check("reading inventory for 6.8", err)
	liveResources68, missingResources68, err := inventory.DiscoverResourcesFromInventory(ctx, client, readInv68)
	check("discovering resources from inventory for 6.8", err)

	if len(liveResources68) != 1 {
		failf("6.8: expected 1 live resource (cm-a), got %d", len(liveResources68))
	}
	if len(missingResources68) != 1 {
		failf("6.8: expected 1 missing resource (svc-a), got %d", len(missingResources68))
	}
	if missingResources68[0].Name != "svc-a" {
		failf("6.8: expected missing resource to be svc-a, got %q", missingResources68[0].Name)
	}
	fmt.Println("   OK: DiscoverResourcesFromInventory: 1 live, 1 missing (svc-a)")

	// Build MissingResources list for GetModuleStatus.
	missing68 := make([]kubernetes.MissingResource, len(missingResources68))
	for i, e := range missingResources68 {
		missing68[i] = kubernetes.MissingResource{
			Kind:      e.Kind,
			Namespace: e.Namespace,
			Name:      e.Name,
		}
	}

	// Call GetModuleStatus with inventory-first path.
	statusResult, err := kubernetes.GetModuleStatus(ctx, client, kubernetes.StatusOptions{
		ReleaseName:      releaseName,
		Namespace:        namespace,
		InventoryLive:    liveResources68,
		MissingResources: missing68,
	})
	check("getting module status for 6.8", err)

	if len(statusResult.Resources) != 2 {
		failf("6.8: expected 2 resources in status (cm-a + svc-a missing), got %d", len(statusResult.Resources))
	}
	fmt.Printf("   OK: status has %d resources\n", len(statusResult.Resources))

	// Find svc-a in the results and check it has "Missing" status.
	foundMissing := false
	for _, r := range statusResult.Resources {
		if r.Name == "svc-a" {
			if string(r.Status) != "Missing" {
				failf("6.8: expected svc-a status to be 'Missing', got %q", r.Status)
			}
			foundMissing = true
		}
	}
	if !foundMissing {
		failf("6.8: svc-a not found in status result")
	}
	fmt.Println("   OK: svc-a shows status 'Missing'")

	// Aggregate status should not be Ready (because of missing resource).
	if string(statusResult.AggregateStatus) == "Ready" {
		failf("6.8: expected AggregateStatus to be non-Ready due to missing resource, got Ready")
	}
	fmt.Printf("   OK: AggregateStatus = %q (not Ready)\n", statusResult.AggregateStatus)

	// Cleanup.
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

// opmLabels returns the standard OPM labels for test resources.
func opmLabels() map[string]interface{} {
	return map[string]interface{}{
		"app.kubernetes.io/managed-by":       "open-platform-model",
		"module-release.opmodel.dev/name":    releaseName,
		"module-release.opmodel.dev/version": moduleVersion,
		"module-release.opmodel.dev/uuid":    releaseID,
		"module.opmodel.dev/name":            releaseName,
		"module.opmodel.dev/version":         moduleVersion,
	}
}

// buildCM builds a single ConfigMap build.Resource.
func buildCM(name string) *build.Resource {
	labels := opmLabels()
	labels["component.opmodel.dev/name"] = "config"

	return &build.Resource{
		Object: &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels":    labels,
			},
			"data": map[string]interface{}{
				"key": fmt.Sprintf("value-%s", name),
			},
		}},
		Component: "config",
	}
}

// buildSvc builds a single Service build.Resource.
func buildSvc(name string) *build.Resource {
	labels := opmLabels()
	labels["component.opmodel.dev/name"] = "web"

	return &build.Resource{
		Object: &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels":    labels,
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"app": releaseName,
				},
				"ports": []interface{}{
					map[string]interface{}{
						"port":       int64(80),
						"targetPort": int64(8080),
						"protocol":   "TCP",
					},
				},
			},
		}},
		Component: "web",
	}
}

// buildResources builds ConfigMap resources.
func buildResources(names []string) []*build.Resource {
	res := make([]*build.Resource, len(names))
	for i, name := range names {
		res[i] = buildCM(name)
	}
	return res
}

// buildServiceResources builds Service resources.
func buildServiceResources(names []string) []*build.Resource {
	res := make([]*build.Resource, len(names))
	for i, name := range names {
		res[i] = buildSvc(name)
	}
	return res
}

// moduleMeta returns the ModuleReleaseMetadata for the test release.
func moduleMeta() build.ModuleReleaseMetadata {
	return build.ModuleReleaseMetadata{
		Name:            releaseName,
		Namespace:       namespace,
		Version:         moduleVersion,
		ReleaseIdentity: releaseID,
	}
}

// buildInventory creates a new InventorySecret from the given resources.
func buildInventory(resources []*build.Resource) *inventory.InventorySecret {
	entries := make([]inventory.InventoryEntry, len(resources))
	for i, r := range resources {
		entries[i] = inventory.NewEntryFromResource(r)
	}

	digest := inventory.ComputeManifestDigest(resources)
	module := inventory.ModuleRef{Path: modulePath, Version: moduleVersion, Name: releaseName}

	inv := &inventory.InventorySecret{
		Metadata: inventory.InventoryMetadata{
			Kind:        "ModuleRelease",
			APIVersion:  "core.opmodel.dev/v1alpha1",
			Name:        releaseName, // module name (same as release name in this test)
			ReleaseName: releaseName,
			Namespace:   namespace,
			ReleaseID:   releaseID,
		},
		Index:   []string{},
		Changes: map[string]*inventory.ChangeEntry{},
	}

	changeID, changeEntry := inventory.PrepareChange(module, "", digest, entries)
	inv.Changes[changeID] = changeEntry
	inv.Index = inventory.UpdateIndex(inv.Index, changeID)
	return inv
}

// cleanup deletes all test resources and the inventory Secret.
func cleanup(ctx context.Context, client *kubernetes.Client) {
	for _, name := range []string{"cm-a", "cm-b"} {
		_ = client.ResourceClient(configMapGVR, namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}
	for _, name := range []string{"svc-a", "svc-b"} {
		_ = client.ResourceClient(serviceGVR, namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}
	secretName := inventory.SecretName(releaseName, releaseID)
	_ = client.Clientset.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
}

// step prints a numbered step header.
func step(n int, desc string) {
	fmt.Printf("\n--- Step %d: %s\n", n, desc)
}

// check prints an error and exits if err is non-nil.
func check(label string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %s: %v\n", label, err)
		os.Exit(1)
	}
}

// failf prints a failure message and exits.
func failf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}
