package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-platform-model/library/opm/helper/platformmodule"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
)

// Source identifies which precedence step produced the resolved platform.
type Source string

const (
	// SourceFlagDir is the explicit --platform <dir> override.
	SourceFlagDir Source = "flag"
	// SourceClusterCR is the cluster Platform CR spec, generated into a
	// module under the OPM home cache.
	SourceClusterCR Source = "cluster"
	// SourceLocalDefault is the local default platform module beside the
	// config file (~/.opm/platform/).
	SourceLocalDefault Source = "local"
)

// Resolution reports where the platform came from — the provenance every
// render-bearing command surfaces (D21: the fallback warns, it never
// silently swaps platforms).
type Resolution struct {
	// Source is the precedence step that produced the platform.
	Source Source
	// Location names the concrete origin: the --platform argument, the CR
	// name, or the local default directory.
	Location string
	// Dir is the platform module directory the kernel acquires. For the
	// flag and local sources it equals Location; for the cluster CR it is
	// the generated module under the cache.
	Dir string
	// SkewPolicy is the cluster CR's spec.skewPolicy verbatim ("Warn",
	// "Refuse" or empty when unset). Set only for SourceClusterCR; the
	// render layer maps it, and the config key, onto the kernel's policy.
	SkewPolicy string
	// Warning is non-empty when resolution fell back from the cluster CR
	// to the local default.
	Warning string
}

// Describe returns the one-line provenance description for command output,
// naming the directory the render acquires.
func (r Resolution) Describe() string {
	switch r.Source {
	case SourceFlagDir:
		return "platform: " + r.Dir + " (--platform)"
	case SourceClusterCR:
		return "platform: cluster Platform CR " + r.Location + " (generated module " + r.Dir + ")"
	case SourceLocalDefault:
		return "platform: " + r.Dir + " (local default)"
	default:
		return "platform: unknown source"
	}
}

// ClusterSpecGetter fetches the cluster Platform CR's spec. It returns
// (spec, name, "", nil) on success and ("", unavailable-reason, nil) when the
// CR is absent or unreadable in a way that permits warn-fallback (NotFound,
// Forbidden — D21). Any other error is fatal to resolution.
type ClusterSpecGetter func(ctx context.Context) (spec map[string]any, name string, unavailable string, err error)

// ResolveOptions selects the platform sources for one command invocation.
type ResolveOptions struct {
	// PlatformFlag is the --platform flag value (highest precedence): a
	// platform module directory.
	PlatformFlag string
	// ConfigPath is the resolved config file path; the local default
	// platform module and the generated-module cache are its siblings, so
	// --config overrides move them together.
	ConfigPath string
	// Cluster is the cluster CR getter. nil means the command is offline
	// (build/render) and MUST NOT read the cluster (D17/D21).
	Cluster ClusterSpecGetter
	// Registry is the CUE registry mapping the cluster CR's dependency
	// closure resolves through (the CLI's configured registry).
	Registry string
	// ModFiles serves published module files for the closure derivation.
	// Nil constructs one from Registry; a test injects a fixture graph.
	ModFiles platformmodule.ModFileSource
}

// Resolve resolves the platform by precedence and returns the platform
// module directory the kernel acquires plus its provenance. Only the cluster
// CR source performs I/O beyond a stat: it is generated into a module under
// the cache (GenerateClusterModule), which derives the dependency closure
// through the registry. Nothing is built here; acquisition is the caller's
// one call after resolution, so every source fails the same way.
func Resolve(ctx context.Context, opts ResolveOptions) (string, Resolution, error) {
	// 1. Explicit local override.
	if opts.PlatformFlag != "" {
		if err := checkPlatformModuleDir(opts.PlatformFlag, "--platform "+opts.PlatformFlag); err != nil {
			return "", Resolution{}, err
		}
		return opts.PlatformFlag, Resolution{Source: SourceFlagDir, Location: opts.PlatformFlag, Dir: opts.PlatformFlag}, nil
	}

	// 2. Cluster Platform CR (cluster-facing commands only).
	fallbackWarning := ""
	if opts.Cluster != nil {
		spec, name, unavailable, err := opts.Cluster(ctx)
		if err != nil {
			return "", Resolution{}, fmt.Errorf("reading cluster Platform: %w", err)
		}
		if unavailable == "" {
			s, err := DecodeCRSpec(spec, name)
			if err != nil {
				return "", Resolution{}, err
			}
			dir, err := GenerateClusterModule(ctx, s, GenerateOptions{
				CacheDir: config.PlatformCacheDir(opts.ConfigPath),
				Registry: opts.Registry,
				ModFiles: opts.ModFiles,
			})
			if err != nil {
				return "", Resolution{}, fmt.Errorf("cluster Platform %q: %w", name, err)
			}
			return dir, Resolution{Source: SourceClusterCR, Location: name, Dir: dir, SkewPolicy: s.SkewPolicy}, nil
		}
		fallbackWarning = "cluster Platform not used (" + unavailable + ") — falling back to the local default platform"
		output.Warn(fallbackWarning)
	}

	// 3. Local default: the module `opm config init` writes.
	localDir := config.PlatformDir(opts.ConfigPath)
	if _, err := os.Stat(localDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", Resolution{}, fmt.Errorf("checking the local default platform at %s: %w", localDir, err)
		}
		return "", Resolution{}, fmt.Errorf(
			"no platform source available: no --platform flag%s and no local default platform module at %s — run 'opm config init' to seed one, or pass --platform <dir>",
			clusterCloseParen(opts.Cluster != nil), localDir)
	}
	if err := checkPlatformModuleDir(localDir, "local default platform "+localDir); err != nil {
		return "", Resolution{}, err
	}
	return localDir, Resolution{Source: SourceLocalDefault, Location: localDir, Dir: localDir, Warning: fallbackWarning}, nil
}

// checkPlatformModuleDir refuses anything but a directory holding
// cue.mod/module.cue, naming the expected shape and the migration. The
// package itself is not evaluated here: the kernel's acquisition is the
// build, and it fails identically for every source.
func checkPlatformModuleDir(dir, what string) error {
	const shape = "expected a platform module directory: cue.mod/module.cue plus a platform.cue package embedding core.#Platform — 'opm config init' seeds one at ~/.opm/platform/"
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: not found; %s", what, shape)
		}
		return fmt.Errorf("%s: %w", what, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is a file, not a platform module; a data-only platform file is no longer accepted — %s", what, shape)
	}
	modFile := filepath.Join(dir, filepath.FromSlash(config.PlatformModuleFileName))
	if _, err := os.Stat(modFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s is not a platform module: %s not found; %s", what, config.PlatformModuleFileName, shape)
		}
		return fmt.Errorf("%s: %w", what, err)
	}
	return nil
}

// clusterCloseParen phrases the no-source error for cluster-facing vs
// offline commands.
func clusterCloseParen(clusterTried bool) string {
	if clusterTried {
		return ", no readable cluster Platform,"
	}
	return ""
}
