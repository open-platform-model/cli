//go:build ignore

// Integration test for platform resolution + kernel materialization
// (enhancement 0006 C2 Phase B, tasks 2.1/2.5).
//
// Verifies the seeded default ~/.opm/platform.cue resolves (offline
// precedence, D21) and materializes through the kernel's
// SynthesizePlatform → Materialize chain (the operator's own ingestion
// path) against the registry in OPM_REGISTRY.
//
// The default platform subscribes to the official catalogs
// (opmodel.dev/catalogs/opm, opmodel.dev/catalogs/kubernetes). When the
// registry does not serve them, the test SKIPS unless
// OPM_ITEST_PLATFORM_MATERIALIZE=1 forces a hard failure — CI registries
// that only publish example modules stay green while a fully-populated
// local registry exercises the real path.
//
// Run with: go run tests/integration/platform-materialize/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/platform"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	registry := os.Getenv("OPM_REGISTRY")
	if registry == "" {
		registry = os.Getenv("CUE_REGISTRY")
	}
	if registry == "" {
		fmt.Println("SKIP: neither OPM_REGISTRY nor CUE_REGISTRY is set")
		return nil
	}
	// The kernel's schema OCILoader resolves opmodel.dev/core@v1 against the
	// process environment, not the kernel registry option — mirror the
	// resolved registry into CUE_REGISTRY for the schema fetch.
	if os.Getenv("CUE_REGISTRY") == "" {
		os.Setenv("CUE_REGISTRY", registry)
	}

	// Seed a temp ~/.opm with the default templates (what config init writes).
	dir, err := os.MkdirTemp("", "opm-platform-materialize-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	configPath := filepath.Join(dir, "config.cue")
	if err := os.WriteFile(configPath, []byte(config.DefaultConfigTemplate), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "platform.cue"), []byte(config.DefaultPlatformTemplate), 0o600); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Offline resolution (nil Cluster getter): must land on the local default.
	in, res, err := platform.Resolve(ctx, platform.ResolveOptions{ConfigPath: configPath})
	if err != nil {
		return fmt.Errorf("resolving platform: %w", err)
	}
	if res.Source != platform.SourceLocalDefault {
		return fmt.Errorf("expected local-default source, got %q", res.Source)
	}
	fmt.Println("resolved:", res.Describe())

	k := kernel.New(kernel.WithRegistry(registry))
	mp, err := platform.Materialize(ctx, k, in)
	if err != nil {
		if os.Getenv("OPM_ITEST_PLATFORM_MATERIALIZE") == "1" {
			return fmt.Errorf("materializing default platform: %w", err)
		}
		fmt.Printf("SKIP: default platform did not materialize against %s (catalogs not served?): %v\n", registry, err)
		return nil
	}

	if len(mp.Resolved) == 0 {
		return fmt.Errorf("materialized platform resolved no subscriptions")
	}
	for path, version := range mp.Resolved {
		fmt.Printf("materialized subscription %s -> %s\n", path, version)
	}
	fmt.Println("PASS: platform-materialize")
	return nil
}
