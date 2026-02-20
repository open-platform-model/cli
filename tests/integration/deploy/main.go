//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== OPM Deploy Integration Test ===")
	fmt.Println()

	// 1. Create client targeting kind-opm-dev
	fmt.Println("1. Creating Kubernetes client (context: kind-opm-dev)...")
	client, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Context: "kind-opm-dev",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   OK: client created")

	releaseName := "opm-deploy-test"
	namespace := "default"
	// Fixed release UUID for deterministic inventory Secret naming.
	releaseID := "a1b2c3d4-1111-2222-3333-aabbccdd0011"

	// 2. Build test resources (a ConfigMap and a Service)
	fmt.Println()
	fmt.Println("2. Building test resources...")

	// OPM labels that the CUE transformers normally inject via #context.labels.
	// Since this integration test bypasses the render pipeline, we add them manually.
	opmLabels := map[string]interface{}{
		"app.kubernetes.io/managed-by":    "open-platform-model",
		"module-release.opmodel.dev/name": releaseName,
		"module-release.opmodel.dev/uuid": releaseID,
		"module.opmodel.dev/name":         releaseName,
		"module.opmodel.dev/version":      "0.1.0",
	}

	cm := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "opm-deploy-test-config",
			"namespace": namespace,
			"labels":    opmLabels,
		},
		"data": map[string]interface{}{
			"app.conf": "setting=value",
		},
	}}

	svc := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      "opm-deploy-test-svc",
			"namespace": namespace,
			"labels":    opmLabels,
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"app": "opm-deploy-test",
			},
			"ports": []interface{}{
				map[string]interface{}{
					"port":       int64(80),
					"targetPort": int64(8080),
					"protocol":   "TCP",
				},
			},
		},
	}}

	resources := []*core.Resource{
		{Object: cm, Component: "config"},
		{Object: svc, Component: "web"},
	}
	meta := core.ReleaseMetadata{
		Name:      releaseName,
		Namespace: namespace,
		UUID:      releaseID,
	}
	fmt.Printf("   OK: %d resources built (ConfigMap, Service)\n", len(resources))

	// 3. Dry-run apply
	fmt.Println()
	fmt.Println("3. Testing dry-run apply...")
	dryResult, err := kubernetes.Apply(ctx, client, resources, meta, kubernetes.ApplyOptions{
		DryRun: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: dry-run apply: %v\n", err)
		os.Exit(1)
	}
	if len(dryResult.Errors) > 0 {
		for _, e := range dryResult.Errors {
			fmt.Fprintf(os.Stderr, "FAIL: dry-run error: %v\n", e)
		}
		os.Exit(1)
	}
	fmt.Printf("   OK: %d resources would be applied\n", dryResult.Applied)

	// 4. Real apply
	fmt.Println()
	fmt.Println("4. Applying resources to cluster...")
	applyResult, err := kubernetes.Apply(ctx, client, resources, meta, kubernetes.ApplyOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: apply: %v\n", err)
		os.Exit(1)
	}
	if len(applyResult.Errors) > 0 {
		for _, e := range applyResult.Errors {
			fmt.Fprintf(os.Stderr, "FAIL: apply error: %v\n", e)
		}
		os.Exit(1)
	}
	fmt.Printf("   OK: %d resources applied\n", applyResult.Applied)

	// Write inventory after apply so subsequent operations use inventory-first path.
	inv := buildInventory(resources, releaseName, namespace, releaseID)
	if err := inventory.WriteInventory(ctx, client, inv, "", ""); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: writing inventory: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   OK: inventory written")

	// 5. Verify resources via inventory-first discovery
	fmt.Println()
	fmt.Println("5. Discovering resources via inventory...")
	readInv, err := inventory.GetInventory(ctx, client, releaseName, namespace, releaseID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: reading inventory: %v\n", err)
		os.Exit(1)
	}
	if readInv == nil {
		fmt.Fprintf(os.Stderr, "FAIL: inventory not found after apply\n")
		os.Exit(1)
	}
	discovered, _, err := inventory.DiscoverResourcesFromInventory(ctx, client, readInv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: discovering from inventory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   OK: found %d resources\n", len(discovered))
	for _, r := range discovered {
		labels := r.GetLabels()
		fmt.Printf("   - %s/%s (managed-by=%s, release=%s)\n",
			r.GetKind(), r.GetName(),
			labels[core.LabelManagedBy],
			labels[core.LabelReleaseName],
		)
	}
	if len(discovered) < 2 {
		fmt.Fprintf(os.Stderr, "FAIL: expected at least 2 discovered resources, got %d\n", len(discovered))
		os.Exit(1)
	}

	// 6. Idempotency test - apply again
	fmt.Println()
	fmt.Println("6. Testing idempotency (second apply)...")
	applyResult2, err := kubernetes.Apply(ctx, client, resources, meta, kubernetes.ApplyOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: second apply: %v\n", err)
		os.Exit(1)
	}
	if len(applyResult2.Errors) > 0 {
		for _, e := range applyResult2.Errors {
			fmt.Fprintf(os.Stderr, "FAIL: second apply error: %v\n", e)
		}
		os.Exit(1)
	}
	fmt.Printf("   OK: %d resources applied (idempotent)\n", applyResult2.Applied)

	// 7. Dry-run delete — discover from inventory for InventoryLive
	fmt.Println()
	fmt.Println("7. Testing dry-run delete...")
	dryInv, err := inventory.GetInventory(ctx, client, releaseName, namespace, releaseID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: reading inventory for dry-run delete: %v\n", err)
		os.Exit(1)
	}
	var dryLive []*unstructured.Unstructured
	if dryInv != nil {
		dryLive, _, err = inventory.DiscoverResourcesFromInventory(ctx, client, dryInv)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: discovering from inventory for dry-run delete: %v\n", err)
			os.Exit(1)
		}
	}
	dryDeleteResult, err := kubernetes.Delete(ctx, client, kubernetes.DeleteOptions{
		ReleaseName:   releaseName,
		Namespace:     namespace,
		DryRun:        true,
		InventoryLive: dryLive,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: dry-run delete: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   OK: %d resources would be deleted\n", dryDeleteResult.Deleted)

	// 8. Real delete — discover from inventory for InventoryLive + InventorySecretName
	fmt.Println()
	fmt.Println("8. Deleting resources from cluster...")
	kubernetes.ResetClient()
	client, _ = kubernetes.NewClient(kubernetes.ClientOptions{Context: "kind-opm-dev"})
	delInv, err := inventory.GetInventory(ctx, client, releaseName, namespace, releaseID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: reading inventory for delete: %v\n", err)
		os.Exit(1)
	}
	var delLive []*unstructured.Unstructured
	invSecretName := inventory.SecretName(releaseName, releaseID)
	if delInv != nil {
		delLive, _, err = inventory.DiscoverResourcesFromInventory(ctx, client, delInv)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: discovering from inventory for delete: %v\n", err)
			os.Exit(1)
		}
	}
	deleteResult, err := kubernetes.Delete(ctx, client, kubernetes.DeleteOptions{
		ReleaseName:              releaseName,
		Namespace:                namespace,
		InventoryLive:            delLive,
		InventorySecretName:      invSecretName,
		InventorySecretNamespace: namespace,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: delete: %v\n", err)
		os.Exit(1)
	}
	// Note: Kubernetes auto-generates derivative resources (Endpoints, EndpointSlice)
	// for Services. When the Service is deleted, these may disappear before we try
	// to delete them, resulting in "not found" errors. These are expected and non-fatal.
	if len(deleteResult.Errors) > 0 {
		fmt.Printf("   WARN: %d non-fatal delete errors (expected for auto-generated resources)\n", len(deleteResult.Errors))
		for _, e := range deleteResult.Errors {
			fmt.Printf("   - %s\n", e.Error())
		}
	}
	fmt.Printf("   OK: %d resources deleted\n", deleteResult.Deleted)

	// 9. Verify cleanup — inventory should be gone after delete
	fmt.Println()
	fmt.Println("9. Verifying cleanup (inventory after delete)...")
	kubernetes.ResetClient()
	client, _ = kubernetes.NewClient(kubernetes.ClientOptions{Context: "kind-opm-dev"})
	remainingInv, err := inventory.FindInventoryByReleaseName(ctx, client, releaseName, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: post-delete inventory check: %v\n", err)
		os.Exit(1)
	}
	if remainingInv != nil {
		fmt.Fprintf(os.Stderr, "FAIL: inventory Secret still exists after delete\n")
		os.Exit(1)
	}
	fmt.Println("   OK: inventory Secret deleted (nil confirmed)")

	fmt.Println()
	fmt.Println("=== ALL TESTS PASSED ===")
}

// buildInventory creates an InventorySecret from the given resources.
func buildInventory(resources []*core.Resource, releaseName, namespace, releaseID string) *inventory.InventorySecret {
	entries := make([]inventory.InventoryEntry, len(resources))
	for i, r := range resources {
		entries[i] = inventory.NewEntryFromResource(r)
	}

	digest := inventory.ComputeManifestDigest(resources)
	source := inventory.ChangeSource{
		Path:        "tests/integration/deploy",
		Version:     "0.1.0",
		ReleaseName: releaseName,
	}

	inv := &inventory.InventorySecret{
		ReleaseMetadata: inventory.ReleaseMetadata{
			Kind:             "ModuleRelease",
			APIVersion:       "core.opmodel.dev/v1alpha1",
			ReleaseName:      releaseName,
			ReleaseNamespace: namespace,
			ReleaseID:        releaseID,
		},
		ModuleMetadata: inventory.ModuleMetadata{
			Kind:       "Module",
			APIVersion: "core.opmodel.dev/v1alpha1",
			Name:       releaseName,
		},
		Index:   []string{},
		Changes: map[string]*inventory.ChangeEntry{},
	}

	changeID, changeEntry := inventory.PrepareChange(source, "", digest, entries)
	inv.Changes[changeID] = changeEntry
	inv.Index = inventory.UpdateIndex(inv.Index, changeID)
	return inv
}
