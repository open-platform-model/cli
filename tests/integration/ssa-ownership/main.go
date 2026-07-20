//go:build ignore

// Server-side-apply field-ownership integration test (enhancement 0006 slice
// C3, design LD4/LD6 as corrected).
//
// This exists because a fake dynamic client cannot catch the bug it guards
// against. Under server-side apply, a field manager's document is its COMPLETE
// declared intent: a field the manager previously owned but now omits is
// released and — when no other manager claims it — pruned from the live object.
// The CLI's opm-cli manager owns spec.owner, spec.module and spec.values, so a
// "targeted" single-field apply silently deletes the other two.
//
// An earlier implementation shipped exactly that, with unit tests that asserted
// the minimal payload and passed, because client-go's fake dynamic client does
// not implement apply-patch merge semantics at all. Only a real API server
// shows the pruning. Hence this program.
//
// What it proves:
//  1. A full-spec apply that changes only spec.owner preserves module + values
//     (the handoff flip).
//  2. Restating unchanged fields does not bump metadata.generation, so the
//     post-flip reconcile wait cannot be fooled by a no-op re-apply.
//  3. A thin-editor apply that restates the operator's owner preserves it.
//
// Requires a running kind cluster at context "kind-opm-dev" with the
// ModuleInstance CRD installed (opm operator install --crds-only).
// Run with: go run tests/integration/ssa-ownership/main.go
package main

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
)

const (
	clusterContext = "kind-opm-dev"
	instanceName   = "ssa-ownership-itest"
	namespace      = "default"
	modulePath     = "opmodel.dev/modules/test/podinfo@v0"
	moduleVersion  = "v0.1.3"
)

func main() {
	ctx := context.Background()
	fmt.Println("=== OPM Server-Side-Apply Field-Ownership Integration Test ===")

	client, err := kubernetes.NewClient(kubernetes.ClientOptions{Context: clusterContext})
	check("creating Kubernetes client", err)

	cleanup(ctx, client)
	defer cleanup(ctx, client)

	values := map[string]any{"replicas": float64(3), "greeting": "hello"}

	step(1, "seed a CLI-owned instance carrying owner + module + values")
	gen1, err := inventory.ApplySpec(ctx, client, inventory.SpecInput{
		Name:          instanceName,
		Namespace:     namespace,
		Owner:         inventory.OwnerCLI,
		ModulePath:    modulePath,
		ModuleVersion: moduleVersion,
		Values:        values,
	})
	check("seeding the instance", err)
	rec := mustGet(ctx, client)
	requireEqual("spec.owner", inventory.OwnerCLI, rec.Owner)
	requireEqual("spec.module.path", modulePath, rec.ModulePath)
	requireTrue("spec.values present", len(rec.SpecValues) == 2)
	fmt.Printf("   OK: seeded at generation %d\n", gen1)

	step(2, "the ownership flip changes owner and preserves module + values")
	gen2, err := inventory.ApplySpec(ctx, client, inventory.SpecInput{
		Name:          instanceName,
		Namespace:     namespace,
		Owner:         inventory.OwnerOperator,
		ModulePath:    rec.ModulePath,
		ModuleVersion: rec.ModuleVersion,
		Values:        rec.SpecValues,
	})
	check("flipping ownership", err)

	rec = mustGet(ctx, client)
	requireEqual("spec.owner after flip", inventory.OwnerOperator, rec.Owner)
	requireEqual("spec.module.path survived the flip", modulePath, rec.ModulePath)
	requireEqual("spec.module.version survived the flip", moduleVersion, rec.ModuleVersion)
	requireTrue("spec.values survived the flip", len(rec.SpecValues) == 2)
	requireTrue("the flip bumped the generation", gen2 > gen1)
	fmt.Printf("   OK: owner=%s, module and values intact, generation %d -> %d\n", rec.Owner, gen1, gen2)

	step(3, "re-applying the identical spec is a no-op and does not bump the generation")
	gen3, err := inventory.ApplySpec(ctx, client, inventory.SpecInput{
		Name:          instanceName,
		Namespace:     namespace,
		Owner:         inventory.OwnerOperator,
		ModulePath:    rec.ModulePath,
		ModuleVersion: rec.ModuleVersion,
		Values:        rec.SpecValues,
	})
	check("re-applying the identical spec", err)
	requireEqual("generation is unchanged by a no-op apply", fmt.Sprint(gen2), fmt.Sprint(gen3))
	fmt.Printf("   OK: generation still %d — the reconcile wait cannot be fooled by a no-op\n", gen3)

	step(4, "a thin-editor apply restates the operator owner and preserves it")
	newValues := map[string]any{"replicas": float64(5), "greeting": "hello"}
	_, err = inventory.ApplySpec(ctx, client, inventory.SpecInput{
		Name:          instanceName,
		Namespace:     namespace,
		Owner:         rec.Owner, // the value read from the live CR
		ModulePath:    modulePath,
		ModuleVersion: moduleVersion,
		Values:        newValues,
	})
	check("thin-editor edit", err)

	rec = mustGet(ctx, client)
	requireEqual("spec.owner survived the edit", inventory.OwnerOperator, rec.Owner)
	requireEqual("spec.module.path survived the edit", modulePath, rec.ModulePath)
	requireTrue("spec.values was updated", fmt.Sprint(rec.SpecValues["replicas"]) == "5")
	fmt.Println("   OK: owner preserved, module intact, values updated")

	fmt.Println("\nPASS: server-side-apply field ownership behaves as the handoff and thin-editor paths require")
}

func mustGet(ctx context.Context, client *kubernetes.Client) *inventory.Record {
	rec, err := inventory.GetRecord(ctx, client, instanceName, namespace)
	check("reading the ModuleInstance", err)
	if rec == nil {
		fail("ModuleInstance %q not found in namespace %q", instanceName, namespace)
	}
	return rec
}

func cleanup(ctx context.Context, client *kubernetes.Client) {
	_ = client.ResourceClient(inventory.ModuleInstanceGVR, namespace).
		Delete(ctx, instanceName, metav1.DeleteOptions{})
}

func step(n int, msg string) {
	fmt.Printf("\n[%d] %s\n", n, msg)
}

func check(what string, err error) {
	if err != nil {
		fail("%s: %v", what, err)
	}
}

func requireEqual(what, want, got string) {
	if want != got {
		fail("%s: want %q, got %q", what, want, got)
	}
}

func requireTrue(what string, ok bool) {
	if !ok {
		fail("%s: condition not met", what)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}
