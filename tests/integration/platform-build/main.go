//go:build ignore

// Integration test for platform resolution + kernel acquisition
// (enhancement 0006 C2 Phase B; 0019 D5 platform modules).
//
// Verifies the seeded local default platform module (~/.opm/platform/, what
// `opm config init` writes) resolves offline (precedence source 3, D21),
// builds through the kernel's shape-gated directory acquisition
// (AcquirePlatformFromDir — the operator's own ingestion path) against the
// registry in OPM_REGISTRY, and that every #registry entry's derived version
// is the build the module's cue.mod pins.
//
// The default platform subscribes to the first-party catalogs at pinned
// versions. When the registry does not serve them, the test SKIPS unless
// OPM_ITEST_PLATFORM_BUILD=1 forces a hard failure — CI registries that only
// publish example modules stay green while GHCR exercises the real path.
//
// Run with: go run tests/integration/platform-build/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
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

	// Seed a temp ~/.opm with the default templates (what config init writes).
	dir, err := os.MkdirTemp("", "opm-platform-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	configPath := filepath.Join(dir, "config.cue")
	if err := os.WriteFile(configPath, []byte(config.DefaultConfigTemplate), 0o600); err != nil {
		return err
	}
	if err := config.WritePlatformModule(config.PlatformDir(configPath)); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Offline resolution (nil Cluster getter): must land on the local default.
	platformDir, res, err := platform.Resolve(ctx, platform.ResolveOptions{ConfigPath: configPath, Registry: registry})
	if err != nil {
		return fmt.Errorf("resolving platform: %w", err)
	}
	if res.Source != platform.SourceLocalDefault {
		return fmt.Errorf("expected local-default source, got %q", res.Source)
	}
	if platformDir != config.PlatformDir(configPath) {
		return fmt.Errorf("expected the local default module directory, got %q", platformDir)
	}
	fmt.Println("resolved:", res.Describe())

	k := kernel.New(kernel.WithRegistry(registry))
	p, err := k.AcquirePlatformFromDir(ctx, platformDir, loaderfile.LoadOptions{Registry: registry})
	if err != nil {
		if os.Getenv("OPM_ITEST_PLATFORM_BUILD") == "1" {
			return fmt.Errorf("building default platform: %w", err)
		}
		fmt.Printf("SKIP: default platform did not build against %s (catalogs not served?): %v\n", registry, err)
		return nil
	}
	if p.Source == nil {
		return fmt.Errorf("acquired platform carries no source")
	}

	spec, err := platform.SpecFromPlatform(p)
	if err != nil {
		return fmt.Errorf("decoding seed spec from the built platform: %w", err)
	}
	if len(spec.Entries) != len(config.DefaultCatalogPaths) {
		return fmt.Errorf("expected %d registry entries, got %d", len(config.DefaultCatalogPaths), len(spec.Entries))
	}
	for i, path := range config.DefaultCatalogPaths {
		want := strings.TrimPrefix(config.DefaultCatalogPins[i], "v")
		found := false
		for _, e := range spec.Entries {
			if e.Path != path {
				continue
			}
			found = true
			if e.Version != want {
				return fmt.Errorf("entry %s: derived version %q, want the pinned %q", path, e.Version, want)
			}
			if !e.Enable {
				return fmt.Errorf("entry %s: expected enabled", path)
			}
			fmt.Printf("built entry %s -> %s (enable=%t)\n", e.Path, e.Version, e.Enable)
		}
		if !found {
			return fmt.Errorf("entry %s missing from the built platform", path)
		}
	}
	fmt.Println("PASS: platform-build")
	return nil
}
