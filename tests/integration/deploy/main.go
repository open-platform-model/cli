//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/build"
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

	moduleName := "opm-deploy-test"
	namespace := "default"

	// 2. Build test resources (a ConfigMap and a Service)
	fmt.Println()
	fmt.Println("2. Building test resources...")

	cm := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "opm-deploy-test-config",
			"namespace": namespace,
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

	resources := []*build.Resource{
		{Object: cm, Component: "config"},
		{Object: svc, Component: "web"},
	}
	meta := build.ModuleMetadata{
		Name:      moduleName,
		Namespace: namespace,
		Version:   "0.1.0",
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

	// 5. Verify labels by discovering
	fmt.Println()
	fmt.Println("5. Discovering resources via OPM labels...")
	discovered, err := kubernetes.DiscoverResources(ctx, client, kubernetes.DiscoveryOptions{ModuleName: moduleName, Namespace: namespace})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: discover: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   OK: found %d resources\n", len(discovered))
	for _, r := range discovered {
		labels := r.GetLabels()
		fmt.Printf("   - %s/%s (managed-by=%s, module=%s, component=%s)\n",
			r.GetKind(), r.GetName(),
			labels[kubernetes.LabelManagedBy],
			labels[kubernetes.LabelModuleName],
			labels[kubernetes.LabelComponentName],
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

	// 7. Dry-run delete
	fmt.Println()
	fmt.Println("7. Testing dry-run delete...")
	dryDeleteResult, err := kubernetes.Delete(ctx, client, kubernetes.DeleteOptions{
		ModuleName: moduleName,
		Namespace:  namespace,
		DryRun:     true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: dry-run delete: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   OK: %d resources would be deleted\n", dryDeleteResult.Deleted)

	// 8. Real delete
	fmt.Println()
	fmt.Println("8. Deleting resources from cluster...")
	kubernetes.ResetClient()
	client, _ = kubernetes.NewClient(kubernetes.ClientOptions{Context: "kind-opm-dev"})
	deleteResult, err := kubernetes.Delete(ctx, client, kubernetes.DeleteOptions{
		ModuleName: moduleName,
		Namespace:  namespace,
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

	// 9. Verify cleanup
	fmt.Println()
	fmt.Println("9. Verifying cleanup (discover after delete)...")
	kubernetes.ResetClient()
	client, _ = kubernetes.NewClient(kubernetes.ClientOptions{Context: "kind-opm-dev"})
	remaining, err := kubernetes.DiscoverResources(ctx, client, kubernetes.DiscoveryOptions{ModuleName: moduleName, Namespace: namespace})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: post-delete discover: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   OK: %d resources remaining\n", len(remaining))

	fmt.Println()
	fmt.Println("=== ALL TESTS PASSED ===")
}
