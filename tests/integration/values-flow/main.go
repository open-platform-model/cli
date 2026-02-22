//go:build ignore

// Values-flow integration test.
//
// Validates the end-to-end loader → builder pipeline for a module that has
// extra values*.cue files sitting in the module directory alongside values.cue.
//
// What is tested:
//
//  1. loader.Load succeeds despite values_prod.cue being present — the file is
//     filtered silently and mod.Values reflects values.cue defaults only.
//
//  2. builder.Build with no --values uses values.cue defaults (image=nginx:default,
//     replicas=1). The extra values_prod.cue has no effect on the release.
//
//  3. builder.Build with --values=values_prod.cue uses the prod values
//     (image=nginx:prod, replicas=3), proving clean separation between the
//     loader's filter path and the builder's explicit override path.
//
// Requires OPM_REGISTRY to be set for opmodel.dev/core@v0 resolution.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/opmodel/cli/internal/builder"
	"github.com/opmodel/cli/internal/loader"
)

func main() {
	fmt.Println("=== OPM Values Flow Integration Test ===")
	fmt.Println()

	registry := os.Getenv("OPM_REGISTRY")
	if registry == "" {
		fmt.Fprintln(os.Stderr, "FAIL: OPM_REGISTRY is not set")
		os.Exit(1)
	}

	ctx := cuecontext.New()
	modPath := fixturePath()

	// Step 1: Load module with extra values*.cue files present.
	fmt.Println("1. Loading multi-values-module (values_prod.cue present in module dir)...")
	mod, err := loader.LoadModule(ctx, modPath, registry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: loader.LoadModule: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   OK: load succeeded — values_prod.cue filtered silently")

	if !mod.Values.Exists() {
		fmt.Fprintln(os.Stderr, "FAIL: mod.Values is not set after load")
		os.Exit(1)
	}
	if err := assertValues(mod.Values, "nginx:default", 1, "mod.Values (from values.cue)"); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   OK: mod.Values from values.cue: image=nginx:default, replicas=1")

	// Step 2: Build release with default values (no --values).
	fmt.Println()
	fmt.Println("2. Building release with default values (no --values)...")
	opts := builder.Options{Name: "values-flow-test", Namespace: "default"}
	rel, err := builder.Build(ctx, mod, opts, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: builder.Build (defaults): %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   OK: release built")

	if err := assertValues(rel.Values, "nginx:default", 1, "release.Values (default build)"); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   OK: release.Values: image=nginx:default, replicas=1 (from values.cue)")

	// Step 3: Build release with --values=values_prod.cue.
	fmt.Println()
	fmt.Println("3. Building release with --values=values_prod.cue...")
	prodValuesPath := filepath.Join(modPath, "values_prod.cue")
	relProd, err := builder.Build(ctx, mod, opts, []string{prodValuesPath})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: builder.Build (values_prod.cue): %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   OK: release built")

	if err := assertValues(relProd.Values, "nginx:prod", 3, "release.Values (prod build)"); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   OK: release.Values: image=nginx:prod, replicas=3 (from values_prod.cue)")

	fmt.Println()
	fmt.Println("=== ALL TESTS PASSED ===")
}

// assertValues checks that the given CUE value has the expected image string
// and replicas integer at the top level. label is used in error messages.
func assertValues(v cue.Value, wantImage string, wantReplicas int64, label string) error {
	image, err := v.LookupPath(cue.ParsePath("image")).String()
	if err != nil {
		return fmt.Errorf("%s: reading image: %w", label, err)
	}
	if image != wantImage {
		return fmt.Errorf("%s: image = %q, want %q", label, image, wantImage)
	}

	replicas, err := v.LookupPath(cue.ParsePath("replicas")).Int64()
	if err != nil {
		return fmt.Errorf("%s: reading replicas: %w", label, err)
	}
	if replicas != wantReplicas {
		return fmt.Errorf("%s: replicas = %d, want %d", label, replicas, wantReplicas)
	}

	return nil
}

// fixturePath returns the absolute path to tests/fixtures/valid/multi-values-module.
func fixturePath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "FAIL: could not determine source file path")
		os.Exit(1)
	}
	// tests/integration/values-flow/ → tests/ → repo root
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, "fixtures", "valid", "multi-values-module")
}
