//go:build ignore

// Integration test for platform resolution + kernel acquisition
// (0019:D5 platform modules).
//
// Verifies the repo's maintained platform module (hack/platform/, the module
// the kind dev flow and the offline tests pass with --platform) resolves as a
// --platform directory, builds through the kernel's shape-gated directory
// acquisition (AcquirePlatformFromDir, the operator's own ingestion path)
// against the registry in OPM_REGISTRY, and that every #registry entry's
// derived version is the catalog build the module's cue.mod pins.
//
// The module subscribes to the first-party catalogs at pinned versions. When
// the registry does not serve them, the test SKIPS unless
// OPM_ITEST_PLATFORM_BUILD=1 forces a hard failure: CI registries that only
// publish example modules stay green while GHCR exercises the real path.
//
// Run with: go run tests/integration/platform-build/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/mod/modfile"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/platform"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
}

// hackPlatformDir is hack/platform/ in this checkout, located from this
// source file so the program runs from any working directory.
func hackPlatformDir() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot locate this source file")
	}
	return filepath.Abs(filepath.Join(filepath.Dir(file), "..", "..", "..", "hack", "platform"))
}

// catalogPins reads the opmodel.dev/catalogs/* pins of a platform module's
// cue.mod/module.cue, as bare versions keyed by major-qualified path.
func catalogPins(dir string) (map[string]string, error) {
	name := filepath.Join(dir, "cue.mod", "module.cue")
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	f, err := modfile.Parse(data, name)
	if err != nil {
		return nil, err
	}
	pins := map[string]string{}
	for path, dep := range f.Deps {
		if strings.HasPrefix(path, platform.CatalogPathPrefix) {
			pins[path] = strings.TrimPrefix(dep.Version, "v")
		}
	}
	return pins, nil
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

	platformDir, err := hackPlatformDir()
	if err != nil {
		return err
	}
	pins, err := catalogPins(platformDir)
	if err != nil {
		return fmt.Errorf("reading %s pins: %w", platformDir, err)
	}
	if len(pins) == 0 {
		return fmt.Errorf("%s pins no catalog", platformDir)
	}

	// A throwaway OPM home: resolution of a --platform directory generates
	// nothing, but the cache location must never be the developer's.
	home, err := os.MkdirTemp("", "opm-platform-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(home)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dir, res, err := platform.Resolve(ctx, platform.ResolveOptions{
		PlatformFlag: platformDir,
		ConfigPath:   filepath.Join(home, "config.cue"),
		Registry:     registry,
	})
	if err != nil {
		return fmt.Errorf("resolving platform: %w", err)
	}
	if res.Source != platform.SourceFlagDir || dir != platformDir {
		return fmt.Errorf("expected the --platform directory %s, got %q from source %q", platformDir, dir, res.Source)
	}
	fmt.Println("resolved:", res.Describe())

	k := kernel.New(kernel.WithRegistry(registry))
	p, err := k.AcquirePlatformFromDir(ctx, dir)
	if err != nil {
		if os.Getenv("OPM_ITEST_PLATFORM_BUILD") == "1" {
			return fmt.Errorf("building %s: %w", dir, err)
		}
		fmt.Printf("SKIP: %s did not build against %s (catalogs not served?): %v\n", dir, registry, err)
		return nil
	}
	if p.Source == nil {
		return fmt.Errorf("acquired platform carries no source")
	}

	reg := p.Package.LookupPath(cue.MakePath(cue.Def("registry")))
	if !reg.Exists() {
		return fmt.Errorf("built platform has no #registry")
	}
	it, err := reg.Fields()
	if err != nil {
		return fmt.Errorf("reading #registry: %w", err)
	}
	got := map[string]string{}
	for it.Next() {
		path := it.Selector().Unquoted()
		var e struct {
			Enable  bool   `json:"enable"`
			Version string `json:"version"`
		}
		if err := it.Value().Decode(&e); err != nil {
			return fmt.Errorf("reading #registry entry %q: %w", path, err)
		}
		if !e.Enable {
			return fmt.Errorf("entry %s: expected enabled", path)
		}
		got[path] = e.Version
	}

	paths := make([]string, 0, len(pins))
	for path := range pins {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		v, ok := got[path]
		if !ok {
			return fmt.Errorf("entry %s missing from the built platform", path)
		}
		if v != pins[path] {
			return fmt.Errorf("entry %s: derived version %q, want the pinned %q", path, v, pins[path])
		}
		fmt.Printf("built entry %s -> %s\n", path, v)
	}
	if len(got) != len(pins) {
		return fmt.Errorf("expected %d registry entries, got %d", len(pins), len(got))
	}
	fmt.Println("PASS: platform-build")
	return nil
}
