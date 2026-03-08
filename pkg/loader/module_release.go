package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/modulerelease"
)

// LoadReleasePackage loads a release CUE package (release.cue + values.cue)
// and returns the raw evaluated cue.Value and the release directory path.
//
// This is the shared entry point for both ModuleRelease and BundleRelease loading.
// The caller inspects the `kind` field to determine which decode path to take.
//
// releaseFile is the path to the release.cue file (or a directory, in which
// case release.cue is assumed to live inside it).
//
// valuesFile is the path to the values CUE file to load alongside release.cue.
// If empty, values.cue in the same directory as the release file is used.
func LoadReleasePackage(cueCtx *cue.Context, releaseFile string, valuesFile string) (cue.Value, string, error) {
	// Resolve release file path: allow directory as shorthand.
	releaseFile, err := resolveReleaseFile(releaseFile)
	if err != nil {
		return cue.Value{}, "", err
	}
	releaseDir := filepath.Dir(releaseFile)

	// Resolve values file path.
	if valuesFile == "" {
		valuesFile = filepath.Join(releaseDir, "values.cue")
	}

	// Load only the two explicit files as one CUE instance.
	// Using explicit filenames prevents other values_*.cue files in the same
	// directory from being included and causing conflicts.
	releaseBase := filepath.Base(releaseFile)
	valuesBase := filepath.Base(valuesFile)

	cfg := &load.Config{
		Dir: releaseDir,
		// LoadFromFiles restricts the package to the named files only.
		// This is the key mechanism that lets us select a specific values file.
	}
	instances := load.Instances([]string{releaseBase, valuesBase}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, "", fmt.Errorf("no CUE instances found for %s + %s", releaseBase, valuesBase)
	}
	if instances[0].Err != nil {
		return cue.Value{}, "", fmt.Errorf("loading release package: %w", instances[0].Err)
	}

	pkg := cueCtx.BuildInstance(instances[0])
	if err := pkg.Err(); err != nil {
		return cue.Value{}, "", fmt.Errorf("building release package: %w", err)
	}

	return pkg, releaseDir, nil
}

// DetectReleaseKind reads the `kind` field from an already-loaded release
// package value. Returns "ModuleRelease", "BundleRelease", or an error.
func DetectReleaseKind(pkg cue.Value) (string, error) {
	kindVal := pkg.LookupPath(cue.ParsePath("kind"))
	if !kindVal.Exists() {
		return "", fmt.Errorf("release package has no 'kind' field")
	}
	kind, err := kindVal.String()
	if err != nil {
		return "", fmt.Errorf("reading 'kind' field: %w", err)
	}
	switch kind {
	case "ModuleRelease", "BundleRelease":
		return kind, nil
	default:
		return "", fmt.Errorf("unknown release kind: %q", kind)
	}
}

// LoadModuleReleaseFromValue decodes a ModuleRelease from an already-loaded
// CUE package value. The fallbackName is used as metadata.name if the CUE
// value does not provide one (typically the release directory name).
//
// This is called directly when the package has already been loaded for kind
// detection (avoiding a double-load of the CUE package).
func LoadModuleReleaseFromValue(cueCtx *cue.Context, pkg cue.Value, fallbackName string) (*modulerelease.ModuleRelease, error) {
	releaseVal := pkg

	if err := releaseVal.Err(); err != nil {
		return nil, fmt.Errorf("evaluating release: %w", err)
	}

	// Module Gate: validate consumer values against #module.#config before any
	// further processing. Catches type mismatches and missing required fields at
	// the values/schema boundary — produces a clear, attributed error rather than
	// a CUE unification error buried in finalization.
	moduleConfigVal := releaseVal.LookupPath(cue.ParsePath("#module.#config"))
	moduleValuesVal := releaseVal.LookupPath(cue.ParsePath("values"))
	if cfgErr := validateConfig(moduleConfigVal, moduleValuesVal, "module", fallbackName); cfgErr != nil {
		return nil, cfgErr
	}

	// Concreteness check on the whole release value.
	// Module Gate already validated the values/schema boundary. This catches any
	// remaining open fields (e.g. uuid interpolations, label constraints) that
	// are not part of the consumer-facing #config.
	if err := releaseVal.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("release %q: not fully concrete: %w", fallbackName, err)
	}

	// Extract release-level metadata by decoding the metadata struct directly.
	relMeta, err := extractReleaseMetadata(releaseVal, fallbackName)
	if err != nil {
		return nil, fmt.Errorf("extracting release metadata: %w", err)
	}

	// Extract module-level metadata from the #module hidden field.
	modMeta, modRaw, err := extractModuleInfo(releaseVal)
	if err != nil {
		return nil, fmt.Errorf("extracting module info: %w", err)
	}

	// Finalize the release value to strip schema constraints (matchN validators,
	// close() enforcement, definition fields) and take defaults. Then extract
	// just the components — the only field the renderer needs for FillPath
	// injection into transformers.
	dataVal, err := finalizeValue(cueCtx, releaseVal)
	if err != nil {
		return nil, fmt.Errorf("finalizing release: %w", err)
	}
	dataComponents := dataVal.LookupPath(cue.ParsePath("components"))
	if !dataComponents.Exists() {
		return nil, fmt.Errorf("no components field in finalized release value")
	}

	return modulerelease.NewModuleRelease(relMeta, module.Module{
		Metadata: modMeta,
		Raw:      modRaw,
	}, releaseVal, dataComponents), nil
}

// resolveReleaseFile normalises the releaseFile argument:
//   - If it is a directory (detected via os.Stat), appends "release.cue".
//   - Otherwise returns it as-is.
//
// DEBT #10: uses os.Stat + IsDir() instead of extension check.
func resolveReleaseFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("releaseFile must not be empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		// Path does not exist or is inaccessible — return as-is and let CUE
		// loader surface the error with full context.
		if os.IsNotExist(err) {
			return path, nil
		}
		return "", fmt.Errorf("stat release file: %w", err)
	}
	if info.IsDir() {
		return filepath.Join(path, "release.cue"), nil
	}
	return path, nil
}

// extractReleaseMetadata decodes the release metadata struct directly from the
// CUE value using Decode(), avoiding manual field-by-field extraction.
func extractReleaseMetadata(v cue.Value, fallbackName string) (*modulerelease.ReleaseMetadata, error) {
	meta := &modulerelease.ReleaseMetadata{Name: fallbackName}

	metaVal := v.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, fmt.Errorf("metadata field not found in release")
	}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding release metadata: %w", err)
	}

	if meta.Name == "" {
		return nil, fmt.Errorf("release metadata.name is empty")
	}
	if meta.Namespace == "" {
		return nil, fmt.Errorf("release metadata.namespace is empty")
	}
	return meta, nil
}

// extractModuleInfo reads module-level metadata from the release's #module hidden
// field using Decode() into ModuleMetadata.
func extractModuleInfo(releaseVal cue.Value) (*module.ModuleMetadata, cue.Value, error) {
	moduleVal := releaseVal.LookupPath(cue.ParsePath("#module"))
	if !moduleVal.Exists() {
		return nil, cue.Value{}, fmt.Errorf("#module field not found in release value")
	}
	if err := moduleVal.Err(); err != nil {
		return nil, cue.Value{}, fmt.Errorf("evaluating #module: %w", err)
	}

	meta := &module.ModuleMetadata{}
	metaVal := moduleVal.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, cue.Value{}, fmt.Errorf("metadata field not found in #module")
	}
	if err := metaVal.Decode(meta); err != nil {
		return nil, cue.Value{}, fmt.Errorf("decoding module metadata: %w", err)
	}

	return meta, moduleVal, nil
}
