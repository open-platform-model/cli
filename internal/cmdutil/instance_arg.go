package cmdutil

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/open-platform-model/cli/internal/config"
)

// InstanceArg holds the resolved instance identifier from a positional CLI arg.
// Was: ReleaseArg (enhancement 0002 D9/D10). The arg may be a file path
// (instance.cue or directory), an instance name, or an instance UUID.
//
// Exactly one of Name or UUID will be non-empty. Namespace is only set when the
// identifier was resolved from a file path.
type InstanceArg struct {
	// Name is the instance name (set when arg was a name or a file path).
	Name string

	// UUID is the instance UUID (set when arg matched the UUID v4/v5 pattern).
	UUID string

	// Namespace is the instance namespace read from the acquired instance's
	// metadata. Only populated when the arg was a file or directory path.
	Namespace string
}

// ToSelectorFlags builds an InstanceSelectorFlags from the resolved arg.
// namespaceFlag (the --namespace CLI flag) overrides the file-derived Namespace
// when non-empty, matching the precedence: flag > file > config default.
func (r InstanceArg) ToSelectorFlags(namespaceFlag string) *InstanceSelectorFlags {
	ns := namespaceFlag
	if ns == "" {
		ns = r.Namespace
	}
	return &InstanceSelectorFlags{
		InstanceName: r.Name,
		InstanceID:   r.UUID,
		Namespace:    ns,
	}
}

// EffectiveNamespace returns the namespace to use for Kubernetes resolution.
// The --namespace flag takes precedence; the file-derived namespace is the
// fallback. An empty string means "use the configured default".
func (r InstanceArg) EffectiveNamespace(namespaceFlag string) string {
	if namespaceFlag != "" {
		return namespaceFlag
	}
	return r.Namespace
}

// ResolveInstanceArg resolves a positional CLI argument into an InstanceArg.
// Was: ResolveReleaseArg. It accepts three forms:
//
//  1. A path to an instance.cue file or a directory containing one — the
//     package is acquired through the kernel with the cfg registry, and the
//     instance name and namespace are read from the acquired instance's
//     metadata.
//  2. An instance UUID — matched by the lowercase UUID v4/v5 pattern.
//  3. An instance name — any other string.
//
// The caller's --namespace flag takes precedence over any namespace found in
// the file (ToSelectorFlags, EffectiveNamespace).
func ResolveInstanceArg(ctx context.Context, arg string, cfg *config.GlobalConfig) (InstanceArg, error) {
	if isInstancePath(arg) {
		return resolveInstanceArgFromFile(ctx, arg, cfg)
	}
	name, uuid := ResolveInstanceIdentifier(arg)
	return InstanceArg{Name: name, UUID: uuid}, nil
}

// isInstancePath reports whether arg should be treated as a filesystem path
// rather than an instance name or UUID. Was: isReleasePath.
//
// Detection order:
//  1. os.Stat succeeds — path exists on disk (file or directory).
//  2. arg ends with ".cue" — explicit file extension.
//  3. arg contains a path separator, or starts with "." or "~" — path-like.
func isInstancePath(arg string) bool {
	if _, err := os.Stat(arg); err == nil {
		return true
	}
	return strings.HasSuffix(arg, ".cue") ||
		strings.ContainsRune(arg, os.PathSeparator) ||
		strings.HasPrefix(arg, ".") ||
		strings.HasPrefix(arg, "~")
}

// resolveInstanceArgFromFile acquires the instance package at arg (an
// instance.cue file, or a directory holding one) through the kernel and reads
// the instance name and namespace from its metadata. The acquire is the one
// the render path runs, so it refuses what a render would refuse: a package
// whose kind is not ModuleInstance, whose metadata.name or metadata.namespace
// is not concrete, or that fails to build. Was: resolveReleaseArgFromFile.
func resolveInstanceArgFromFile(ctx context.Context, arg string, cfg *config.GlobalConfig) (InstanceArg, error) {
	if err := ValidateInstanceInputPath(arg); err != nil {
		return InstanceArg{}, err
	}

	dir, err := InstanceDir(arg)
	if err != nil {
		return InstanceArg{}, err
	}

	inst, err := config.NewKernel(cfg.Registry).AcquireInstanceFromDir(ctx, dir)
	if err != nil {
		return InstanceArg{}, fmt.Errorf("loading instance %q: %w", arg, err)
	}

	return InstanceArg{Name: inst.Metadata.Name, Namespace: inst.Metadata.Namespace}, nil
}
