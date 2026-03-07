// Command factory-smoke exercises the full OPM factory pipeline:
//
//  1. Load the kubernetes provider from the factory v1alpha1 CUE module.
//  2. Load a release (ModuleRelease or BundleRelease) from a release.cue + values.cue pair.
//  3. Detect the release kind and dispatch to the appropriate renderer.
//  4. Print each rendered resource as YAML to stdout.
//
// This is a development tool / smoke test, not a production binary.
// Run from the experiments/factory directory:
//
//	go run ./cmd
//
// Flags:
//
//	-release   Path to release.cue file, or directory containing it
//	           (default: <cue-module>/examples/releases/minecraft/)
//	-values    Path to the values CUE file
//	           (default: values.cue in the same directory as -release)
//	-provider  Provider name from #Registry (default: kubernetes)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/opmodel/cli/experiments/factory/internal/core/provider"
	"github.com/opmodel/cli/experiments/factory/pkg/engine"
	"github.com/opmodel/cli/experiments/factory/pkg/loader"
	"github.com/opmodel/cli/experiments/factory/pkg/output"
)

func main() {
	releaseArg := flag.String("release", "", "Path to release.cue file or directory containing it.\n"+
		"Supports both ModuleRelease and BundleRelease.\n"+
		"Defaults to <cue-module>/examples/releases/minecraft/")
	valuesArg := flag.String("values", "", "Path to the values CUE file.\n"+
		"Defaults to values.cue in the same directory as -release.")
	providerName := flag.String("provider", "kubernetes", "Provider name from #Registry")
	flag.Parse()

	if err := run(*providerName, *releaseArg, *valuesArg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(providerName, releaseArg, valuesArg string) error {
	// Locate the factory v1alpha1 CUE module directory relative to this source file.
	cueModuleDir, err := factoryCUEModuleDir()
	if err != nil {
		return fmt.Errorf("locating CUE module: %w", err)
	}
	fmt.Printf("CUE module: %s\n", cueModuleDir)

	// Resolve release path.
	releasePath := releaseArg
	if releasePath == "" {
		releasePath = filepath.Join(cueModuleDir, "examples", "releases", "minecraft")
	}
	// Resolve values path (empty means: let loader default to values.cue beside the release).
	valuesPath := valuesArg

	cueCtx := cuecontext.New()

	// Load the provider.
	fmt.Printf("Loading provider %q...\n", providerName)
	prov, err := loader.LoadProvider(cueCtx, cueModuleDir, providerName)
	if err != nil {
		return fmt.Errorf("loading provider: %w", err)
	}
	fmt.Printf("Provider: %s v%s\n", prov.Metadata.Name, prov.Metadata.Version)

	// Load the release package (shared for both ModuleRelease and BundleRelease).
	fmt.Printf("Loading release from %s...\n", releasePath)
	if valuesPath != "" {
		fmt.Printf("  values: %s\n", valuesPath)
	}
	pkg, releaseDir, err := loader.LoadReleasePackage(cueCtx, releasePath, valuesPath)
	if err != nil {
		return fmt.Errorf("loading release package: %w", err)
	}

	// Detect the release kind and dispatch.
	kind, err := loader.DetectReleaseKind(pkg)
	if err != nil {
		return fmt.Errorf("detecting release kind: %w", err)
	}

	switch kind {
	case "ModuleRelease":
		return runModuleRelease(cueCtx, pkg, releaseDir, prov, cueModuleDir)
	case "BundleRelease":
		return runBundleRelease(cueCtx, pkg, prov, cueModuleDir)
	default:
		return fmt.Errorf("unsupported release kind: %q", kind)
	}
}

// runModuleRelease handles the ModuleRelease rendering path.
func runModuleRelease(cueCtx *cue.Context, pkg cue.Value, releaseDir string, prov *provider.Provider, cueModuleDir string) error {
	rel, err := loader.LoadModuleReleaseFromValue(cueCtx, pkg, filepath.Base(releaseDir))
	if err != nil {
		return fmt.Errorf("loading module release: %w", err)
	}
	fmt.Printf("Release: %s (namespace: %s, uuid: %s)\n",
		rel.Metadata.Name, rel.Metadata.Namespace, rel.Metadata.UUID)
	fmt.Printf("Module:  %s v%s\n", rel.Module.Metadata.FQN, rel.Module.Metadata.Version)

	// Build the engine and render.
	r := engine.NewModuleRenderer(prov, cueModuleDir, cueCtx)

	fmt.Printf("\nRendering...\n\n")
	result, err := r.Render(context.Background(), rel)
	if err != nil {
		return fmt.Errorf("rendering: %w", err)
	}

	// Print warnings.
	output.PrintWarnings(os.Stderr, result.Warnings)

	// Print each resource as YAML.
	fmt.Printf("--- # %d resources rendered\n", len(result.Resources))
	if err := output.PrintResources(os.Stdout, result.Resources); err != nil {
		return fmt.Errorf("printing resources: %w", err)
	}

	return nil
}

// runBundleRelease handles the BundleRelease rendering path.
func runBundleRelease(cueCtx *cue.Context, pkg cue.Value, prov *provider.Provider, cueModuleDir string) error {
	rel, err := loader.LoadBundleReleaseFromValue(cueCtx, pkg)
	if err != nil {
		return fmt.Errorf("loading bundle release: %w", err)
	}
	fmt.Printf("BundleRelease: %s (uuid: %s)\n",
		rel.Metadata.Name, rel.Metadata.UUID)
	fmt.Printf("Bundle:        %s %s\n",
		rel.Bundle.Metadata.FQN, rel.Bundle.Metadata.Version)
	fmt.Printf("Releases:      %d module release(s)\n", len(rel.Releases))
	for _, key := range sortedKeys(rel.Releases) {
		modRel := rel.Releases[key]
		fmt.Printf("  %s: %s (namespace: %s)\n",
			key, modRel.Module.Metadata.FQN, modRel.Metadata.Namespace)
	}

	// Build the bundle renderer and render all releases.
	br := engine.NewBundleRenderer(prov, cueModuleDir, cueCtx)

	fmt.Printf("\nRendering...\n\n")
	result, err := br.Render(context.Background(), rel)
	if err != nil {
		return fmt.Errorf("rendering bundle: %w", err)
	}

	// Print warnings.
	output.PrintWarnings(os.Stderr, result.Warnings)

	// Print each resource as YAML.
	fmt.Printf("--- # %d resources rendered from %d release(s)\n",
		len(result.Resources), len(result.ReleaseOrder))
	if err := output.PrintResources(os.Stdout, result.Resources); err != nil {
		return fmt.Errorf("printing resources: %w", err)
	}

	return nil
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// factoryCUEModuleDir returns the absolute path to the factory v1alpha1 CUE module root.
// It uses runtime.Caller to find the source file's location — this is intentional:
// this binary is a development smoke test, always run from source, never deployed.
func factoryCUEModuleDir() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot determine source file path via runtime.Caller")
	}
	// thisFile: .../experiments/factory/cmd/main.go
	// target:   .../experiments/factory/v1alpha1
	factoryRoot := filepath.Dir(filepath.Dir(thisFile))
	dir := filepath.Join(factoryRoot, "v1alpha1")
	if _, err := os.Stat(filepath.Join(dir, "cue.mod")); err != nil {
		return "", fmt.Errorf("expected CUE module at %s: %w", dir, err)
	}
	return dir, nil
}
