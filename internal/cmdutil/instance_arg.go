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

	// Namespace is the instance namespace extracted from the file's metadata.
	// Only populated when the arg was a file or directory path.
	Namespace string
}

// ToSelectorFlags builds a ReleaseSelectorFlags from the resolved arg.
// namespaceFlag (the --namespace CLI flag) overrides the file-derived Namespace
// when non-empty, matching the precedence: flag > file > config default.
//
// Note: the returned flag bundle is still named ReleaseSelectorFlags — that type
// (and its --release-name/--release-id flags) is renamed in the X4 slice; this
// cross-type reference compiles unchanged in the meantime.
func (r InstanceArg) ToSelectorFlags(namespaceFlag string) *ReleaseSelectorFlags {
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
//     instance name and namespace are extracted by loading the CUE file.
//  2. An instance UUID — matched by the lowercase UUID v4/v5 pattern.
//  3. An instance name — any other string.
//
// For form 1, the cfg registry and CUE context are used for CUE module
// resolution. The caller's --namespace flag takes precedence over any namespace
// found in the file.
func ResolveInstanceArg(arg string, cfg *config.GlobalConfig) (InstanceArg, error) {
	if isInstancePath(arg) {
		return resolveInstanceArgFromFile(arg, cfg)
	}
	name, uuid := ResolveReleaseIdentifier(arg)
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

// resolveInstanceArgFromFile loads an instance.cue file (or a directory
// containing one) and extracts the instance name and namespace from its
// metadata. Was: resolveReleaseArgFromFile.
func resolveInstanceArgFromFile(arg string, cfg *config.GlobalConfig) (InstanceArg, error) {
	if err := ValidateReleaseInputPath(arg); err != nil {
		return InstanceArg{}, err
	}

	cueCtx := cfg.CueContext
	if cueCtx == nil {
		cueCtx = cuecontext.New()
	}

	pkg, _, err := loader.LoadInstanceFile(cueCtx, arg, loader.LoadOptions{Registry: cfg.Registry})
	if err != nil {
		return InstanceArg{}, fmt.Errorf("loading instance file %q: %w", arg, err)
	}

	name, namespace, err := extractInstanceFileIdentity(pkg)
	if err != nil {
		return InstanceArg{}, fmt.Errorf("reading metadata from %q: %w", arg, err)
	}

	return InstanceArg{Name: name, Namespace: namespace}, nil
}

// extractInstanceFileIdentity reads the instance name and namespace from the
// metadata struct of an already-loaded CUE instance value. Was:
// extractReleaseFileIdentity.
//
// name must be a concrete string; namespace is extracted best-effort (an
// incomplete or constrained namespace value is silently ignored so the caller
// can fall back to the configured default or --namespace flag).
func extractInstanceFileIdentity(pkg cue.Value) (name, namespace string, err error) {
	metaVal := pkg.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return "", "", fmt.Errorf("no metadata field in instance file")
	}

	nameVal := metaVal.LookupPath(cue.ParsePath("name"))
	if !nameVal.Exists() {
		return "", "", fmt.Errorf("metadata.name not found in instance file")
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
