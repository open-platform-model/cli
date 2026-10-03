//go:build ignore

// Integration program for staged apply: a CustomResourceDefinition, a
// Namespace, a custom resource of that definition and a ConfigMap in that
// Namespace, handed to kubernetes.Apply in reverse weight order, apply in one
// call on a cluster that has none of them. A dry run of the same set skips the
// custom resource and reports the ConfigMap's namespace as missing (the dry
// run of the Namespace persists nothing).
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

const (
	namespace = "opm-staging-test"
	crdName   = "stagingtests.opmodel.dev"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== OPM Apply Staging Integration Test ===")
	fmt.Println()

	fmt.Println("1. Creating Kubernetes client (context: kind-opm-dev)...")
	client, err := kubernetes.NewClient(kubernetes.ClientOptions{Context: "kind-opm-dev"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   OK: client created")

	objs := resources()
	fail := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
		cleanup(ctx, client, objs)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("2. Making sure the cluster has none of the test objects...")
	cleanup(ctx, client, objs)
	fmt.Println("   OK: clean")

	fmt.Println()
	fmt.Println("3. Dry run in reverse weight order...")
	dry, err := kubernetes.Apply(ctx, client, objs, "opm-staging-test", kubernetes.ApplyOptions{DryRun: true})
	if err != nil {
		fail("dry run returned an error: %v", err)
	}
	if dry.Skipped != 1 {
		fail("dry run skipped %d resources, want 1 (the custom resource)", dry.Skipped)
	}
	if len(dry.Errors) != 1 || dry.Errors[0].Kind != "ConfigMap" || !apierrors.IsNotFound(dry.Errors[0].Err) {
		fail("dry run errors = %v, want exactly the ConfigMap's namespace NotFound", dry.Errors)
	}
	if dry.Applied != 2 {
		fail("dry run applied %d resources, want 2 (the CRD and the Namespace)", dry.Applied)
	}
	fmt.Println("   OK: custom resource skipped, ConfigMap refused for its missing namespace (documented limit)")

	fmt.Println()
	fmt.Println("4. Real apply in reverse weight order...")
	res, err := kubernetes.Apply(ctx, client, objs, "opm-staging-test", kubernetes.ApplyOptions{
		EstablishDeadline: time.Now().Add(2 * time.Minute),
	})
	if err != nil {
		fail("apply returned an error: %v", err)
	}
	if len(res.Errors) > 0 {
		fail("apply had %d resource error(s): %v", len(res.Errors), res.Errors)
	}
	if res.Applied != 4 || res.Skipped != 0 {
		fail("applied %d skipped %d, want 4 and 0", res.Applied, res.Skipped)
	}
	cr := objs[0]
	if _, err := client.ResourceClient(kubernetes.GVRFromUnstructured(cr), namespace).Get(ctx, cr.GetName(), metav1.GetOptions{}); err != nil {
		fail("custom resource not readable after apply: %v", err)
	}
	fmt.Println("   OK: all four applied on the first call")

	fmt.Println()
	fmt.Println("5. Cleaning up...")
	cleanup(ctx, client, objs)
	fmt.Println("   OK: cleaned up")

	fmt.Println()
	fmt.Println("=== All apply-staging scenarios passed ===")
}

// resources returns the test objects in reverse weight order: the custom
// resource, the ConfigMap, the Namespace, the CustomResourceDefinition.
func resources() []*unstructured.Unstructured {
	cr := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "opmodel.dev/v1alpha1",
		"kind":       "StagingTest",
		"metadata":   map[string]any{"name": "sample", "namespace": namespace},
		"spec":       map[string]any{"message": "staged"},
	}}
	cm := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": "staging-config", "namespace": namespace},
		"data":       map[string]any{"k": "v"},
	}}
	ns := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata":   map[string]any{"name": namespace},
	}}
	crd := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata":   map[string]any{"name": crdName},
		"spec": map[string]any{
			"group": "opmodel.dev",
			"names": map[string]any{
				"kind":     "StagingTest",
				"listKind": "StagingTestList",
				"plural":   "stagingtests",
				"singular": "stagingtest",
			},
			"scope": "Namespaced",
			"versions": []any{map[string]any{
				"name":    "v1alpha1",
				"served":  true,
				"storage": true,
				"schema": map[string]any{"openAPIV3Schema": map[string]any{
					"type":                                 "object",
					"x-kubernetes-preserve-unknown-fields": true,
				}},
			}},
		},
	}}
	return []*unstructured.Unstructured{cr, cm, ns, crd}
}

// cleanup deletes every test object through the dynamic client (instance
// delete and prune never remove a Namespace or a CRD) and waits until they are
// gone, so a second run starts from a clean cluster.
func cleanup(ctx context.Context, client *kubernetes.Client, objs []*unstructured.Unstructured) {
	for _, obj := range objs {
		err := client.ResourceClient(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace()).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			fmt.Printf("   note: deleting %s/%s: %v\n", obj.GetKind(), obj.GetName(), err)
		}
	}
	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := kubernetes.WaitAbsent(waitCtx, client, objs, time.Now()); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: cleanup: %v\n", err)
		os.Exit(1)
	}
}
