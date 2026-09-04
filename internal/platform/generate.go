package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/open-platform-model/library/opm/helper/platformmodule"
)

// ClusterPlatformModulePath is the generated cluster platform module's own
// identity: the reserved, never-published platforms namespace (0019 D6), the
// same path the operator generates under, so a CR that renders in-cluster
// renders from a byte-identical module here.
const ClusterPlatformModulePath = "opmodel.dev/platforms/cluster@v0"

// ErrEntryMissingVersion is the sentinel for a subscription with no version:
// a stored CR predating the scalar-version shape. Generation refuses it
// before any registry I/O.
var ErrEntryMissingVersion = errors.New("registry entry has no version")

// GenerateOptions configures GenerateClusterModule.
type GenerateOptions struct {
	// CacheDir is the directory generated modules live under
	// (config.PlatformCacheDir): one content-hash-named subdirectory each.
	CacheDir string

	// Registry is the CUE registry mapping (CUE_REGISTRY syntax) closure
	// derivation resolves published module files through. Empty falls back
	// to CUE_REGISTRY in the process environment.
	Registry string

	// ModFiles serves published module files for the closure derivation.
	// Nil constructs one from Registry; a test injects a fixture graph.
	ModFiles platformmodule.ModFileSource
}

// GenerateClusterModule turns a cluster Platform CR's spec into a platform
// module on disk exactly as the operator's PlatformReconciler does: the
// entries feed the library's generator, the dependency closure is derived
// from the pinned modules' published module files, core is pinned at the
// library's verified release and the module path is
// ClusterPlatformModulePath. It returns the module directory.
//
// The module lives under opts.CacheDir at <sha256 of the generated files>/,
// so an unchanged CR maps to the same directory across invocations and the
// write is idempotent: an existing directory holding identical content is
// reused untouched; otherwise the files are written to a staging sibling
// and renamed into place, so a reader never sees a partial module and two
// concurrent invocations converge on identical content. Nothing here is
// published, written to the cluster or user-editable; the cache is derived
// state and may be deleted at any time.
//
// A subscription without a version fails with ErrEntryMissingVersion and
// the legacy-CR hint before any registry access. An unpublished pin fails
// from the closure derivation naming the module path and version.
func GenerateClusterModule(ctx context.Context, spec Spec, opts GenerateOptions) (string, error) {
	entries, err := generatorEntries(spec)
	if err != nil {
		return "", err
	}
	if opts.CacheDir == "" {
		return "", errors.New("platform cache directory is not set")
	}

	src := opts.ModFiles
	if src == nil {
		src, err = platformmodule.NewRegistry(platformmodule.RegistryConfig{
			Registry:   opts.Registry,
			ClientType: "opm-cli",
			Env:        os.Environ(),
		})
		if err != nil {
			return "", fmt.Errorf("configuring module registry: %w", err)
		}
	}
	deps, err := platformmodule.Closure(ctx, src, platformmodule.Roots(entries))
	if err != nil {
		// The pinned build does not exist, or the registry is unreachable:
		// the error names the module path and version.
		return "", fmt.Errorf("resolving cluster Platform dependencies: %w", err)
	}

	files, err := platformmodule.Generate(platformmodule.Input{
		Name:       spec.Name,
		Type:       spec.Type,
		ModulePath: ClusterPlatformModulePath,
		Entries:    entries,
		Deps:       deps,
	})
	if err != nil {
		return "", fmt.Errorf("generating cluster platform module: %w", err)
	}
	return writeCached(opts.CacheDir, files)
}

// generatorEntries maps the spec's entries to the generator's, refusing a
// subscription without a version: a stored object predating the CRD-required
// field. The hint is permanent, not transitional — stored CRs keep their old
// shape in etcd until their next spec write.
func generatorEntries(spec Spec) ([]platformmodule.Entry, error) {
	entries := make([]platformmodule.Entry, 0, len(spec.Entries))
	for _, e := range spec.Entries {
		if e.Version == "" {
			return nil, fmt.Errorf("registry entry %q: %w — the cluster Platform may predate the scalar-version subscription shape (legacy filter-shaped CR); re-apply its spec with version set on every subscription", e.Path, ErrEntryMissingVersion)
		}
		entries = append(entries, platformmodule.Entry{Path: e.Path, Version: e.Version, Enable: e.Enable})
	}
	return entries, nil
}

// hashFiles returns the hex SHA-256 over the generated files, name and
// content, in sorted name order — the cache directory's name.
func hashFiles(files platformmodule.Files) string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write(files[name])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// writeCached places files under cacheDir/<hash>/ and returns that
// directory, reusing it untouched when it already holds identical content.
func writeCached(cacheDir string, files platformmodule.Files) (string, error) {
	hash := hashFiles(files)
	dir := filepath.Join(cacheDir, hash)
	if dirHolds(dir, files) {
		return dir, nil
	}
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return "", fmt.Errorf("creating platform cache directory %s: %w", cacheDir, err)
	}

	// Staging beside the target so the rename is atomic on one filesystem;
	// the hidden prefix keeps a crashed write from ever being mistaken for a
	// complete module.
	staging, err := os.MkdirTemp(cacheDir, ".staging-"+hash[:12]+"-")
	if err != nil {
		return "", fmt.Errorf("creating staging directory: %w", err)
	}
	if err := files.WriteTo(staging); err != nil {
		_ = os.RemoveAll(staging)
		return "", fmt.Errorf("writing cluster platform module: %w", err)
	}
	if err := os.Rename(staging, dir); err != nil {
		// The name is taken: a concurrent invocation converged on the same
		// content (success), or a stale directory holds it (replaced).
		if dirHolds(dir, files) {
			_ = os.RemoveAll(staging)
			return dir, nil
		}
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			_ = os.RemoveAll(staging)
			return "", fmt.Errorf("replacing stale platform module %s: %w", dir, rmErr)
		}
		if err := os.Rename(staging, dir); err != nil {
			_ = os.RemoveAll(staging)
			return "", fmt.Errorf("moving cluster platform module into place at %s: %w", dir, err)
		}
	}
	return dir, nil
}

// dirHolds reports whether dir exists and holds exactly the bytes of every
// file in files. Reads are root-scoped to dir.
func dirHolds(dir string, files platformmodule.Files) bool {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return false
	}
	defer root.Close()
	for name, want := range files {
		got, err := root.ReadFile(filepath.FromSlash(name))
		if err != nil || !bytes.Equal(got, want) {
			return false
		}
	}
	return true
}
