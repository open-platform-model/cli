//go:build ignore

// Integration test for the delete side of the ownership rule: prune and
// instance delete ask the library's delete verdict on a live read and send
// the DELETE with a precondition on the UID they read.
//
// Scenarios:
//   - prune leaves a stale ConfigMap whose name a user's unlabelled object took
//   - prune deletes a stale ConfigMap the instance still owns
//   - instance delete leaves a ConfigMap annotated for another instance
//   - the API server refuses a DELETE whose UID precondition is stale
//
// Requires a running kind cluster at context "kind-opm-dev".
// Run with: go run tests/integration/delete-ownership/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
)

const (
	clusterContext = "kind-opm-dev"
	instanceName   = "opm-delete-ownership-test"
	namespace      = "default"
	// The identity the instance's record holds, stamped on what it applied.
	instanceID = "c3d4e5f6-3333-4444-5555-ccddeeff0011"
	otherID    = "d4e5f6a7-4444-5555-6666-ddeeff001122"
)

var (
	configMapGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	names        = []string{"own-stale", "taken-stale", "adopted", "kept", "replaced"}
)

func main() {
	ctx := context.Background()
	fmt.Println("=== OPM Delete Ownership Integration Test ===")

	client, err := kubernetes.NewClient(kubernetes.ClientOptions{Context: clusterContext})
	check("creating Kubernetes client", err)
	cleanup(ctx, client)
	defer cleanup(ctx, client)

	// ----------------------------------------------------------------
	step(1, "prune leaves a stale name a user's object took, and deletes its own")
	create(ctx, client, configMap("own-stale", true, ""))
	create(ctx, client, configMap("taken-stale", false, "")) // no OPM label: a user's object

	stale := []k8sinventory.Entry{
		k8sinventory.NewEntry(configMap("own-stale", true, "")),
		k8sinventory.NewEntry(configMap("taken-stale", true, "")),
	}
	leftBehind, err := inventory.PruneStaleResources(ctx, client, stale, instanceID)
	check("pruning", err)
	if len(leftBehind) != 1 || leftBehind[0].Entry.Name != "taken-stale" {
		failf("want exactly taken-stale left behind, got %+v", leftBehind)
	}
	fmt.Printf("   OK: left behind: %s\n", leftBehind[0].Reason)
	waitFor(ctx, client, "own-stale", false)
	waitFor(ctx, client, "taken-stale", true)
	fmt.Println("   OK: own-stale is deleted, taken-stale is still there")

	// ----------------------------------------------------------------
	step(2, "instance delete leaves an object annotated for another instance")
	adopted := create(ctx, client, configMap("adopted", true, otherID))
	kept := create(ctx, client, configMap("kept", true, ""))

	result, err := kubernetes.Delete(ctx, client, kubernetes.DeleteOptions{
		InstanceName:          instanceName,
		Namespace:             namespace,
		InstanceUUID:          instanceID,
		InventoryLive:         []*unstructured.Unstructured{adopted, kept},
		InventoryRecordExists: true,
	})
	check("deleting", err)
	if len(result.Errors) != 0 {
		failf("delete had errors: %v", result.Errors[0])
	}
	if result.Deleted != 1 || len(result.LeftBehind) != 1 || result.LeftBehind[0].Name != "adopted" {
		failf("want 1 deleted and adopted left behind, got deleted=%d leftBehind=%+v", result.Deleted, result.LeftBehind)
	}
	fmt.Printf("   OK: left behind: %s\n", result.LeftBehind[0].Reason)
	waitFor(ctx, client, "kept", false)
	waitFor(ctx, client, "adopted", true)
	fmt.Println("   OK: kept is deleted, adopted is still there")

	// ----------------------------------------------------------------
	step(3, "the API server refuses a DELETE whose UID precondition is stale")
	create(ctx, client, configMap("replaced", true, ""))
	staleUID := types.UID("00000000-0000-0000-0000-000000000000")
	err = client.ResourceClient(configMapGVR, namespace).Delete(ctx, "replaced", metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &staleUID},
	})
	if !apierrors.IsConflict(err) {
		failf("want a Conflict for a stale UID precondition, got: %v", err)
	}
	waitFor(ctx, client, "replaced", true)
	fmt.Println("   OK: Conflict, and the object is still there")

	fmt.Println()
	fmt.Println("=== ALL SCENARIOS PASSED ===")
}

// configMap builds a test ConfigMap. managed stamps the labels an apply of
// the instance stamps; adoptedBy, when set, is the adopt annotation's value.
func configMap(name string, managed bool, adoptedBy string) *unstructured.Unstructured {
	meta := map[string]interface{}{"name": name, "namespace": namespace}
	if managed {
		meta["labels"] = map[string]interface{}{
			opmlabels.ManagedBy:          opmlabels.ManagedByCLI,
			opmlabels.ModuleInstanceUUID: instanceID,
			opmlabels.ModuleInstanceName: instanceName,
		}
	}
	if adoptedBy != "" {
		meta["annotations"] = map[string]interface{}{opmlabels.AnnotationAdopt: adoptedBy}
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   meta,
		"data":       map[string]interface{}{"key": "value-" + name},
	}}
}

func create(ctx context.Context, client *kubernetes.Client, obj *unstructured.Unstructured) *unstructured.Unstructured {
	created, err := client.ResourceClient(configMapGVR, namespace).Create(ctx, obj, metav1.CreateOptions{})
	check("creating ConfigMap/"+obj.GetName(), err)
	return created
}

// cleanup deletes every ConfigMap of the test.
func cleanup(ctx context.Context, client *kubernetes.Client) {
	for _, name := range names {
		_ = client.ResourceClient(configMapGVR, namespace).Delete(ctx, name, metav1.DeleteOptions{})
		waitFor(ctx, client, name, false)
	}
}

func waitFor(ctx context.Context, client *kubernetes.Client, name string, wantPresent bool) {
	deadline := time.Now().Add(15 * time.Second)
	for {
		_, err := client.ResourceClient(configMapGVR, namespace).Get(ctx, name, metav1.GetOptions{})
		if wantPresent && err == nil {
			return
		}
		if !wantPresent && apierrors.IsNotFound(err) {
			return
		}
		if err != nil && !apierrors.IsNotFound(err) {
			failf("waiting for ConfigMap/%s: %v", name, err)
		}
		if time.Now().After(deadline) {
			failf("timed out waiting for ConfigMap/%s (want present: %v)", name, wantPresent)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func step(n int, desc string) { fmt.Printf("\n--- Step %d: %s\n", n, desc) }

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
