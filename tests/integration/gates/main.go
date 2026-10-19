//go:build ignore

// Integration test for the pre-apply gate battery (enhancement 0006)
// against a live cluster with the ModuleInstance CRD installed.
//
// This covers the happy paths that require a real API server: CRD presence,
// the CRD field floor, the controller-version ceiling being inert (skip) against
// the current controller release (no Platform.status.controllerVersion), and the
// status-RBAC pre-flight passing for a cluster-admin. The refusal paths
// (missing CRD hint, SSAR denial for a namespace-scoped user, refuse-when-newer
// ceiling) are covered by internal/inventory unit tests and, for the ceiling,
// by task 7.1 once an A6-carrying controller release exists.
//
// Requires a running kind cluster at context "kind-opm-dev" with the
// ModuleInstance CRD installed (opm controller install --crds-only).
// Run with: go run tests/integration/gates/main.go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
)

const clusterContext = "kind-opm-dev"

func main() {
	ctx := context.Background()
	fmt.Println("=== OPM Pre-Apply Gate Battery Integration Test ===")

	client, err := kubernetes.NewClient(kubernetes.ClientOptions{Context: clusterContext})
	check("creating Kubernetes client", err)

	step(1, "CRD presence gate passes when the ModuleInstance CRD is installed")
	check("GateCRDPresent", inventory.GateCRDPresent(ctx, client))
	fmt.Println("   OK: ModuleInstance CRD present")

	step(2, "CRD field-presence floor passes (spec.owner + status.inventory)")
	check("GateCRDFieldFloor", inventory.GateCRDFieldFloor(ctx, client))
	fmt.Println("   OK: CRD schema carries spec.owner and status.inventory")

	step(3, "Controller-version ceiling is inert against the current controller (skip)")
	// A released-semver CLI version against a cluster with no
	// Platform.status.controllerVersion must be skipped (solo-cluster semantics),
	// i.e. apply is permitted. Passing a real semver exercises the non-dev path.
	if err := inventory.GateControllerVersionCeiling(ctx, client, "1.0.0"); err != nil {
		failf("expected the ceiling to be inert (skip) against the current controller, got: %v", err)
	}
	fmt.Println("   OK: ceiling skipped (no Platform.status.controllerVersion)")

	step(4, "Status-RBAC pre-flight passes for a permitted user")
	check("GateStatusRBAC", inventory.GateStatusRBAC(ctx, client, "default"))
	fmt.Println("   OK: patch moduleinstances/status is allowed")

	fmt.Println("\n=== ALL SCENARIOS PASSED ===")
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
