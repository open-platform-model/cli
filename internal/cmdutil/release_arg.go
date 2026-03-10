package cmdutil

import (
	"fmt"
	"os"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/pkg/loader"
)

// ReleaseArg holds the resolved release identifier from a positional CLI arg.
// The arg may be a file path (release.cue or directory), a release name, or a
// release UUID.
//
// Exactly one of Name or UUID will be non-empty. Namespace is only set when the
// identifier was resolved from a file path.
type ReleaseArg struct {
	// Name is the release name (set when arg was a name or a file path).
	Name string

	// UUID is the release UUID (set when arg matched the UUID v4/v5 pattern).
	UUID string

	// Namespace is the release namespace extracted from the file's metadata.
	// Only populated when the arg was a file or directory path.
	Namespace string
}

// ToSelectorFlags builds a ReleaseSelectorFlags from the resolved arg.
// namespaceFlag (the --namespace CLI flag) overrides the file-derived Namespace
// when non-empty, matching the precedence: flag > file > config default.
func (r ReleaseArg) ToSelectorFlags(namespaceFlag string) *ReleaseSelectorFlags {
	ns := namespaceFlag
	if ns == "" {
		ns = r.Namespace
	}
	return &ReleaseSelectorFlags{
		ReleaseName: r.Name,
		ReleaseID:   r.UUID,
		Namespace:   ns,
	}
}

// EffectiveNamespace returns the namespace to use for Kubernetes resolution.
// The --namespace flag takes precedence; the file-derived namespace is the
// fallback. An empty string means "use the configured default".
func (r ReleaseArg) EffectiveNamespace(namespaceFlag string) string {
	if namespaceFlag != "" {
		return namespaceFlag
	}
	return r.Namespace
}

// ResolveReleaseArg resolves a positional CLI argument into a ReleaseArg.
// It accepts three forms:
//
//  1. A path to a release.cue file or a directory containing one — the release
//     name and namespace are extracted by loading the CUE file.
//  2. A release UUID — matched by the lowercase UUID v4/v5 pattern.
//  3. A release name — any other string.
//
// For form 1, the cfg registry and CUE context are used for CUE module
// resolution. The caller's --namespace flag takes precedence over any namespace
// found in the file.
func ResolveReleaseArg(arg string, cfg *config.GlobalConfig) (ReleaseArg, error) {
	if isReleasePath(arg) {
		return resolveReleaseArgFromFile(arg, cfg)
	}
	name, uuid := ResolveReleaseIdentifier(arg)
	return ReleaseArg{Name: name, UUID: uuid}, nil
}

// isReleasePath reports whether arg should be treated as a filesystem path
// rather than a release name or UUID.
//
// Detection order:
//  1. os.Stat succeeds — path exists on disk (file or directory).
//  2. arg ends with ".cue" — explicit file extension.
//  3. arg contains a path separator, or starts with "." or "~" — path-like.
func isReleasePath(arg string) bool {
	if _, err := os.Stat(arg); err == nil {
		return true
	}
	return strings.HasSuffix(arg, ".cue") ||
		strings.ContainsRune(arg, os.PathSeparator) ||
		strings.HasPrefix(arg, ".") ||
		strings.HasPrefix(arg, "~")
}

// resolveReleaseArgFromFile loads a release.cue file (or a directory containing
// one) and extracts the release name and namespace from its metadata.
func resolveReleaseArgFromFile(arg string, cfg *config.GlobalConfig) (ReleaseArg, error) {
	cueCtx := cfg.CueContext
	if cueCtx == nil {
		cueCtx = cuecontext.New()
	}

	pkg, _, err := loader.LoadReleaseFile(cueCtx, arg, cfg.Registry)
	if err != nil {
		return ReleaseArg{}, fmt.Errorf("loading release file %q: %w", arg, err)
	}

	name, namespace, err := extractReleaseFileIdentity(pkg)
	if err != nil {
		return ReleaseArg{}, fmt.Errorf("reading metadata from %q: %w", arg, err)
	}

	return ReleaseArg{Name: name, Namespace: namespace}, nil
}

// extractReleaseFileIdentity reads the release name and namespace from the
// metadata struct of an already-loaded CUE release value.
//
// name must be a concrete string; namespace is extracted best-effort (an
// incomplete or constrained namespace value is silently ignored so the caller
// can fall back to the configured default or --namespace flag).
func extractReleaseFileIdentity(pkg cue.Value) (name, namespace string, err error) {
	metaVal := pkg.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return "", "", fmt.Errorf("no metadata field in release file")
	}

	nameVal := metaVal.LookupPath(cue.ParsePath("name"))
	if !nameVal.Exists() {
		return "", "", fmt.Errorf("metadata.name not found in release file")
	}
	name, err = nameVal.String()
	if err != nil {
		return "", "", fmt.Errorf("metadata.name is not a concrete string: %w", err)
	}
	if name == "" {
		return "", "", fmt.Errorf("metadata.name is empty")
	}

	// Namespace is best-effort: if not concrete (e.g. still a CUE constraint),
	// leave it empty so the caller falls back to the config default.
	if nsVal := metaVal.LookupPath(cue.ParsePath("namespace")); nsVal.Exists() {
		if ns, nsErr := nsVal.String(); nsErr == nil {
			namespace = ns
		}
	}

	return name, namespace, nil
}
