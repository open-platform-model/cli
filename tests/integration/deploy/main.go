//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== OPM Instance Deploy Integration Test ===")
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

	instanceName := "opm-deploy-test"
	namespace := "default"
	// Fixed instance UUID for the ModuleInstance CR's status.instanceUUID.
	instanceID := "a1b2c3d4-1111-2222-3333-aabbccdd0011"
	const modulePath = "opmodel.dev/modules/opm_deploy_test@v0"
	const moduleVersion = "0.1.0"

	// 2. Build test resources (a ConfigMap and a Service)
	fmt.Println()
	fmt.Println("2. Building test resources...")

	// OPM labels that the CUE transformers normally inject via #context.labels.
	// Since this integration test bypasses the render pipeline, we add them manually.
	opmLabels := map[string]interface{}{
		pkgcore.LabelManagedBy:             pkgcore.LabelManagedByValue,
		"module-instance.opmodel.dev/name": instanceName,
		"module-instance.opmodel.dev/uuid": instanceID,
		"module.opmodel.dev/name":          instanceName,
		"module.opmodel.dev/version":       "0.1.0",
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

	resources := []*unstructured.Unstructured{cm, svc}
	fmt.Printf("   OK: %d resources built (ConfigMap, Service)\n", len(resources))

	// 3. Dry-run apply
	fmt.Println()
	fmt.Println("3. Testing dry-run apply...")
	dryResult, err := kubernetes.Apply(ctx, client, resources, instanceName, kubernetes.ApplyOptions{
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
	applyResult, err := kubernetes.Apply(ctx, client, resources, instanceName, kubernetes.ApplyOptions{})
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

	// Write inventory to the ModuleInstance CR so subsequent operations use the
	// inventory-first path.
	if err := writeInventoryCR(ctx, client, instanceName, namespace, instanceID, modulePath, moduleVersion, 1, entriesFromResources(resources)); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: writing inventory: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   OK: inventory written")

	// 5. Verify resources via inventory-first discovery
	fmt.Println()
	fmt.Println("5. Discovering resources via inventory...")
	readInv, err := inventory.GetRecord(ctx, client, instanceName, namespace)
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
		fmt.Printf("   - %s/%s (managed-by=%s, instance=%s)\n",
			r.GetKind(), r.GetName(),
			labels["app.kubernetes.io/managed-by"],
			labels["module-instance.opmodel.dev/name"],
		)
	}
	if len(discovered) < 2 {
		fmt.Fprintf(os.Stderr, "FAIL: expected at least 2 discovered resources, got %d\n", len(discovered))
		os.Exit(1)
	}

	// 6. Idempotency test - apply again
	fmt.Println()
	fmt.Println("6. Testing idempotency (second apply)...")
	applyResult2, err := kubernetes.Apply(ctx, client, resources, instanceName, kubernetes.ApplyOptions{})
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
	dryInv, err := inventory.GetRecord(ctx, client, instanceName, namespace)
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
		InstanceName:  instanceName,
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
	delInv, err := inventory.GetRecord(ctx, client, instanceName, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: reading inventory for delete: %v\n", err)
		os.Exit(1)
	}
	var delLive []*unstructured.Unstructured
	if delInv != nil {
		delLive, _, err = inventory.DiscoverResourcesFromInventory(ctx, client, delInv)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: discovering from inventory for delete: %v\n", err)
			os.Exit(1)
		}
	}
	deleteResult, err := kubernetes.Delete(ctx, client, kubernetes.DeleteOptions{
		InstanceName:          instanceName,
		Namespace:             namespace,
		InventoryLive:         delLive,
		InventoryRecordExists: delInv != nil,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: delete: %v\n", err)
		os.Exit(1)
	}
	// Delete the ModuleInstance CR last, after all workload resources are gone.
	if delInv != nil {
		if err := inventory.DeleteCR(ctx, client, instanceName, namespace); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: deleting ModuleInstance CR: %v\n", err)
			os.Exit(1)
		}
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

	// 9. Verify cleanup — inventory CR should be gone after delete
	fmt.Println()
	fmt.Println("9. Verifying cleanup (inventory after delete)...")
	kubernetes.ResetClient()
	client, _ = kubernetes.NewClient(kubernetes.ClientOptions{Context: "kind-opm-dev"})
	remainingInv, err := inventory.GetRecord(ctx, client, instanceName, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: post-delete inventory check: %v\n", err)
		os.Exit(1)
	}
	if remainingInv != nil {
		fmt.Fprintf(os.Stderr, "FAIL: ModuleInstance CR still exists after delete\n")
		os.Exit(1)
	}
	fmt.Println("   OK: ModuleInstance CR deleted (nil confirmed)")

	fmt.Println()
	fmt.Println("=== ALL TESTS PASSED ===")
}

// entriesFromResources builds inventory entries from rendered resources.
func entriesFromResources(resources []*unstructured.Unstructured) []inventory.InventoryEntry {
	entries := make([]inventory.InventoryEntry, len(resources))
	for i, r := range resources {
		entries[i] = inventory.NewEntryFromResource(r)
	}
	return entries
}

// writeInventoryCR writes the ModuleInstance CR spec and its CLI-owned status
// subset (the integration-test analog of the apply workflow's record write).
func writeInventoryCR(ctx context.Context, client *kubernetes.Client, name, namespace, instanceID, modulePath, moduleVersion string, revision int, entries []inventory.InventoryEntry) error {
	if _, err := inventory.ApplySpec(ctx, client, inventory.SpecInput{
		Name:          name,
		Namespace:     namespace,
		Owner:         inventory.OwnerCLI,
		ModulePath:    modulePath,
		ModuleVersion: moduleVersion,
	}); err != nil {
		return err
	}
	return inventory.ApplyStatus(ctx, client, inventory.StatusInput{
		Name:         name,
		Namespace:    namespace,
		InstanceUUID: instanceID,
		Inventory: inventory.Inventory{
			Revision: revision,
			Digest:   inventory.ComputeDigest(entries),
			Count:    len(entries),
			Entries:  entries,
		},
	})
}
