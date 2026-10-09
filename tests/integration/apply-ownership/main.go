//go:build ignore

// Integration test for the apply side of the ownership rule: the ownership
// guard runs on every apply, refuses before any write, honours the adopt
// annotation, and lets go of an object another instance adopted.
//
// Scenarios:
//   - a first apply writes the record
//   - a later apply that newly renders a foreign ConfigMap exits 1 and
//     changes nothing
//   - the adopt annotation with the instance's UUID lets the instance take it
//   - an object annotated for another instance is neither applied nor deleted
//     and leaves the inventory
//
// Requires a running kind cluster at context "kind-opm-dev" with the
// ModuleInstance CRDs installed.
// Run with: go run tests/integration/apply-ownership/main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/module"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	workflowapply "github.com/open-platform-model/cli/internal/workflow/apply"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

const (
	clusterContext = "kind-opm-dev"
	instanceName   = "opm-apply-ownership-test"
	namespace      = "default"
	instanceID     = "e5f6a7b8-5555-6666-7777-eeff00112233"
	otherID        = "f6a7b8c9-6666-7777-8888-ff0011223344"
	foreignValue   = "set-by-someone-else"
)

var (
	configMapGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	names        = []string{"own", "taken"}
)

func main() {
	ctx := context.Background()
	fmt.Println("=== OPM Apply Ownership Integration Test ===")

	client, err := kubernetes.NewClient(kubernetes.ClientOptions{Context: clusterContext})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	cleanup(ctx, client)
	// The scenarios panic with a failure on the first thing that is wrong, so
	// that the test objects are removed on a failed run too.
	code := 0
	func() {
		defer func() {
			if f, ok := recover().(failure); ok {
				fmt.Fprintln(os.Stderr, "FAIL: "+string(f))
				code = 1
			}
		}()
		scenarios(ctx, client)
	}()
	cleanup(ctx, client)
	if code != 0 {
		os.Exit(code)
	}
	fmt.Println()
	fmt.Println("=== ALL SCENARIOS PASSED ===")
}

// failure is the panic value of a failed check.
type failure string

func scenarios(ctx context.Context, client *kubernetes.Client) {
	// ----------------------------------------------------------------
	step(1, "a first apply writes the record")
	check("first apply", apply(ctx, client, "own"))
	wantInventory(ctx, client, "own")

	// ----------------------------------------------------------------
	step(2, "a later apply refuses a foreign ConfigMap with exit 1 and changes nothing")
	create(ctx, client, foreignConfigMap("taken", ""))
	err := apply(ctx, client, "own", "taken")
	var exitErr *opmexit.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != opmexit.ExitGeneralError {
		failf("want exit 1 from the ownership guard, got: %v", err)
	}
	for _, want := range []string{"ConfigMap/" + namespace + "/taken", "not managed by OPM", opmlabels.AnnotationAdopt + "=" + instanceID, "apply stopped before any change"} {
		if !strings.Contains(err.Error(), want) {
			failf("the refusal does not contain %q: %v", want, err)
		}
	}
	fmt.Printf("   OK: refused: %v\n", err)
	wantData(ctx, client, "taken", foreignValue)
	wantInventory(ctx, client, "own")
	fmt.Println("   OK: the foreign object and the record are unchanged")

	// ----------------------------------------------------------------
	step(3, "the adopt annotation lets the instance take the object")
	annotate(ctx, client, "taken", instanceID)
	check("apply after the adopt annotation", apply(ctx, client, "own", "taken"))
	wantData(ctx, client, "taken", "value-taken")
	wantInventory(ctx, client, "own", "taken")
	fmt.Println("   OK: taken is applied and recorded")

	// ----------------------------------------------------------------
	step(4, "an object annotated for another instance is neither applied nor deleted")
	annotate(ctx, client, "taken", otherID)
	setData(ctx, client, "taken", foreignValue)
	check("apply after the hand-over", apply(ctx, client, "own", "taken"))
	wantData(ctx, client, "taken", foreignValue)
	wantInventory(ctx, client, "own")
	fmt.Println("   OK: taken is still there, unchanged, and out of the inventory")

	// A later apply that no longer renders it does not prune it either.
	check("apply without the handed-over object", apply(ctx, client, "own"))
	wantData(ctx, client, "taken", foreignValue)
	fmt.Println("   OK: a later apply does not delete it")
}

// apply runs the apply workflow for the instance, rendering the named
// ConfigMaps as the kernel would label them.
func apply(ctx context.Context, client *kubernetes.Client, configMaps ...string) error {
	resources := make([]*unstructured.Unstructured, 0, len(configMaps))
	for _, name := range configMaps {
		resources = append(resources, renderedConfigMap(name))
	}
	return workflowapply.Execute(ctx, workflowapply.Request{
		Result: &workflowrender.Result{
			Resources: resources,
			Instance:  module.InstanceMetadata{Name: instanceName, Namespace: namespace, UUID: instanceID},
			Module: module.ModuleMetadata{
				Name: "apply-ownership", ModulePath: "testing.opmodel.dev/modules/cli/apply-ownership@v0", Version: "0.1.0",
			},
			RenderDigest: "sha256:apply-ownership",
		},
		K8sClient: client,
		Log:       output.InstanceLogger(instanceName),
		Options: workflowapply.Options{
			SuccessAppliedMessage:  "Instance applied",
			SuccessUpToDateMessage: "Instance up to date",
		},
	})
}

func renderedConfigMap(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": name, "namespace": namespace,
			"labels": map[string]interface{}{
				opmlabels.ManagedBy:          opmlabels.ManagedByCLI,
				opmlabels.ModuleInstanceUUID: instanceID,
				opmlabels.ModuleInstanceName: instanceName,
			},
		},
		"data": map[string]interface{}{"key": "value-" + name},
	}}
}

// foreignConfigMap is a ConfigMap no OPM apply wrote; adoptedBy, when set, is
// the adopt annotation's value.
func foreignConfigMap(name, adoptedBy string) *unstructured.Unstructured {
	meta := map[string]interface{}{"name": name, "namespace": namespace}
	if adoptedBy != "" {
		meta["annotations"] = map[string]interface{}{opmlabels.AnnotationAdopt: adoptedBy}
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   meta,
		"data":       map[string]interface{}{"key": foreignValue},
	}}
}

func create(ctx context.Context, client *kubernetes.Client, obj *unstructured.Unstructured) {
	_, err := client.ResourceClient(configMapGVR, namespace).Create(ctx, obj, metav1.CreateOptions{})
	check("creating ConfigMap/"+obj.GetName(), err)
}

// annotate sets the adopt annotation of a live ConfigMap, as a user does with
// kubectl annotate.
func annotate(ctx context.Context, client *kubernetes.Client, name, uuid string) {
	update(ctx, client, name, func(cm *unstructured.Unstructured) {
		annotations := cm.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[opmlabels.AnnotationAdopt] = uuid
		cm.SetAnnotations(annotations)
	})
}

func setData(ctx context.Context, client *kubernetes.Client, name, value string) {
	update(ctx, client, name, func(cm *unstructured.Unstructured) {
		check("setting data", unstructured.SetNestedField(cm.Object, value, "data", "key"))
	})
}

func update(ctx context.Context, client *kubernetes.Client, name string, change func(*unstructured.Unstructured)) {
	res := client.ResourceClient(configMapGVR, namespace)
	cm, err := res.Get(ctx, name, metav1.GetOptions{})
	check("reading ConfigMap/"+name, err)
	change(cm)
	_, err = res.Update(ctx, cm, metav1.UpdateOptions{})
	check("updating ConfigMap/"+name, err)
}

func wantData(ctx context.Context, client *kubernetes.Client, name, want string) {
	cm, err := client.ResourceClient(configMapGVR, namespace).Get(ctx, name, metav1.GetOptions{})
	check("reading ConfigMap/"+name, err)
	got, _, _ := unstructured.NestedString(cm.Object, "data", "key")
	if got != want {
		failf("ConfigMap/%s holds %q, want %q", name, got, want)
	}
}

// wantInventory fails unless the instance's record lists exactly the named
// ConfigMaps.
func wantInventory(ctx context.Context, client *kubernetes.Client, want ...string) {
	rec, err := inventory.GetRecord(ctx, client, instanceName, namespace)
	check("reading the record", err)
	if rec == nil {
		failf("the instance has no record")
	}
	got := map[string]bool{}
	for _, e := range rec.Inventory.Entries {
		got[e.Name] = true
	}
	if len(got) != len(want) {
		failf("the inventory holds %v, want %v", rec.Inventory.Entries, want)
	}
	for _, name := range want {
		if !got[name] {
			failf("the inventory holds %v, want %v", rec.Inventory.Entries, want)
		}
	}
}

// cleanup deletes the record and every ConfigMap of the test and gives each
// ConfigMap a moment to go. It never fails the run.
func cleanup(ctx context.Context, client *kubernetes.Client) {
	_ = client.ResourceClient(inventory.ModuleInstanceGVR, namespace).Delete(ctx, instanceName, metav1.DeleteOptions{})
	for _, name := range names {
		_ = client.ResourceClient(configMapGVR, namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}
	deadline := time.Now().Add(15 * time.Second)
	for _, name := range names {
		for time.Now().Before(deadline) {
			if _, err := client.ResourceClient(configMapGVR, namespace).Get(ctx, name, metav1.GetOptions{}); apierrors.IsNotFound(err) {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
}

func step(n int, desc string) { fmt.Printf("\n--- Step %d: %s\n", n, desc) }

func check(label string, err error) {
	if err != nil {
		failf("%s: %v", label, err)
	}
}

func failf(format string, args ...interface{}) {
	panic(failure(fmt.Sprintf(format, args...)))
}
