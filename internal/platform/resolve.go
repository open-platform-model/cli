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
	// SourceArgumentDir is a platform module directory named as a command
	// argument (`opm platform check <dir>`), above every other source.
	SourceArgumentDir Source = "argument"
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
// render-bearing command surfaces (0006:D21: the fallback warns, it never
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
	// RegistryOrigin names which half of the cluster CR the module was
	// generated from: RegistryOriginEffective (the registry the operator
	// recorded on status) or RegistryOriginSpec (the authored
	// subscriptions, because no operator has recorded one). Set only for
	// SourceClusterCR.
	RegistryOrigin string
	// PackageIdentity is status.packageIdentity as the operator recorded
	// it, reported and never verified (the CLI cannot recompute it: the
	// identity covers every active claim coordinate, and a claim that
	// overlaps a subscription leaves no trace on the recorded registry).
	// Set only for SourceClusterCR, empty when none is recorded.
	PackageIdentity string
	// Warning is non-empty when resolution fell back from the cluster CR
	// to the local default.
	Warning string
}

// Registry origins for Resolution.RegistryOrigin.
const (
	// RegistryOriginEffective is the resolved registry the operator
	// recorded on the Platform's status: its own union of the authored
	// subscriptions and the catalogs active claims contributed.
	RegistryOriginEffective = "effective"
	// RegistryOriginSpec is the CR's authored subscriptions, used when the
	// status records no registry.
	RegistryOriginSpec = "spec"
)

// Describe returns the one-line provenance description for command output,
// naming the directory the render acquires.
func (r Resolution) Describe() string {
	switch r.Source {
	case SourceArgumentDir:
		return "platform: " + r.Dir + " (argument)"
	case SourceFlagDir:
		return "platform: " + r.Dir + " (--platform)"
	case SourceClusterCR:
		return "platform: cluster Platform CR " + r.Location + " (" + r.describeRegistry() + "generated module " + r.Dir + ")"
	case SourceLocalDefault:
		return "platform: " + r.Dir + " (local default)"
	default:
		return "platform: unknown source"
	}
}

// describeRegistry is the cluster CR's registry provenance, as a prefix
// ending in ", ": which half of the CR the module was generated from and,
// for the effective registry, the package identity the operator recorded.
// Empty for a Resolution that carries no origin.
func (r Resolution) describeRegistry() string {
	switch r.RegistryOrigin {
	case RegistryOriginEffective:
		if r.PackageIdentity == "" {
			return "effective registry, no package identity recorded, "
		}
		return "effective registry, package " + r.PackageIdentity + ", "
	case RegistryOriginSpec:
		return "spec registry, no operator generation recorded, "
	default:
		return ""
	}
}

// ClusterPlatform is the singleton cluster Platform document as the getter
// read it: name, generation, spec and status, all undecoded. The status
// half carries the effective registry the operator generated the running
// package from (DecodeCR); it is nil on a cluster no operator has
// reconciled.
type ClusterPlatform struct {
	Name       string
	Generation int64
	Spec       map[string]any
	Status     map[string]any
}

// ClusterPlatformGetter fetches the cluster Platform CR. It returns
// (doc, "", nil) on success and (nil, unavailable-reason, nil) when the CR
// is absent or unreadable in a way that permits warn-fallback (NotFound,
// Forbidden — 0006:D21). Any other error is fatal to resolution.
type ClusterPlatformGetter func(ctx context.Context) (doc *ClusterPlatform, unavailable string, err error)

// ErrClusterRead marks a fatal failure to read the cluster Platform: the
// cluster is unreachable or the API rejected the read. NotFound and
// Forbidden never reach it — they are warn-fallback conditions.
var ErrClusterRead = errors.New("reading cluster Platform")

// ErrNoClusterPlatform is returned instead of the local fallback when the
// cluster Platform is absent or unreadable and ResolveOptions.NoLocalFallback
// is set: a command whose subject is the cluster's own platform has nothing
// to resolve, and the local default is not a stand-in for it.
var ErrNoClusterPlatform = errors.New("no readable cluster Platform")

// ResolveOptions selects the platform sources for one command invocation.
type ResolveOptions struct {
	// Argument is a platform module directory named as a positional command
	// argument. Highest precedence, and set only by commands that take one
	// (`opm platform check <dir>`); empty everywhere else.
	Argument string
	// PlatformFlag is the --platform flag value: a platform module
	// directory, above the cluster CR and the configured default.
	PlatformFlag string
	// ConfigPath is the resolved config file path; the local default
	// platform module and the generated-module cache are its siblings, so
	// --config overrides move them together.
	ConfigPath string
	// Cluster is the cluster CR getter. nil means the command is offline
	// (build/render) and MUST NOT read the cluster (0006:D17/D21).
	Cluster ClusterPlatformGetter
	// NoLocalFallback refuses the local-default step when the cluster
	// Platform is unavailable, returning ErrNoClusterPlatform instead. Set
	// by commands whose subject is the cluster's platform (`opm platform
	// pull`), for which the local default would be a different platform
	// wearing the cluster's name.
	NoLocalFallback bool
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
	// 0. A directory named as a command argument outranks every configured
	// source; it gets the same module-shape check as the flag, so a
	// non-module directory fails before anything is built.
	if opts.Argument != "" {
		if err := checkPlatformModuleDir(opts.Argument, "platform directory "+opts.Argument); err != nil {
			return "", Resolution{}, err
		}
		return opts.Argument, Resolution{Source: SourceArgumentDir, Location: opts.Argument, Dir: opts.Argument}, nil
	}

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
		doc, unavailable, err := opts.Cluster(ctx)
		if err != nil {
			return "", Resolution{}, fmt.Errorf("%w: %w", ErrClusterRead, err)
		}
		if unavailable == "" {
			return resolveClusterCR(ctx, doc, opts)
		}
		if opts.NoLocalFallback {
			return "", Resolution{}, fmt.Errorf("%w (%s)", ErrNoClusterPlatform, unavailable)
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

// resolveClusterCR generates the platform module the cluster renders against.
// The registry the operator recorded on status wins over the authored spec
// whenever it holds entries: it is the union the operator resolved from the
// subscriptions and the active TransformerRegistration claims, and it is what
// the running package was generated from (0015:D13/D17). The spec is the
// fallback for a cluster no operator has generated for.
//
// Two divergences between spec and effective package are warned, never
// silently substituted: a status behind the spec's generation, and a Platform
// the operator refused. Neither changes which registry is used — the effective
// package is what the cluster renders against in both cases, and the warning
// says why the laptop is deliberately behind.
func resolveClusterCR(ctx context.Context, doc *ClusterPlatform, opts ResolveOptions) (string, Resolution, error) {
	s, eff, err := DecodeCR(doc)
	if err != nil {
		return "", Resolution{}, err
	}

	entries, origin, identity := s.Entries, RegistryOriginSpec, ""
	if eff != nil && len(eff.Entries) > 0 {
		entries, origin, identity = eff.Entries, RegistryOriginEffective, eff.PackageIdentity
		if eff.ObservedGeneration < doc.Generation {
			output.Warn(fmt.Sprintf(
				"cluster Platform generation %d is not yet generated by the operator (status describes generation %d); rendering against the effective package",
				doc.Generation, eff.ObservedGeneration))
		}
		if eff.Ready != nil && eff.Ready.Status == conditionFalse {
			output.Warn(fmt.Sprintf(
				"cluster Platform is Ready=False (%s); rendering against the last good package the operator recorded",
				eff.Ready.Reason))
		}
	}

	dir, err := GenerateClusterModule(ctx, Spec{Name: s.Name, Type: s.Type, Entries: entries}, GenerateOptions{
		CacheDir: config.PlatformCacheDir(opts.ConfigPath),
		Registry: opts.Registry,
		ModFiles: opts.ModFiles,
	})
	if err != nil {
		return "", Resolution{}, fmt.Errorf("cluster Platform %q: %w", s.Name, err)
	}
	return dir, Resolution{
		Source:          SourceClusterCR,
		Location:        s.Name,
		Dir:             dir,
		SkewPolicy:      s.SkewPolicy,
		RegistryOrigin:  origin,
		PackageIdentity: identity,
	}, nil
}

// conditionFalse is metav1.ConditionFalse's value, the one status the two
// registry warnings key off.
const conditionFalse = "False"

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
