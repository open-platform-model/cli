package loader

import (
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/experiments/factory/internal/core/module"
	"github.com/opmodel/cli/experiments/factory/internal/core/modulerelease"
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

// LoadRelease loads a ModuleRelease from a release.cue file and an optional
// values file.
//
// releaseFile is the path to the release.cue file (or a directory, in which
// case release.cue is assumed to live inside it).
//
// valuesFile is the path to the values CUE file to load alongside release.cue.
// If empty, values.cue in the same directory as the release file is used.
//
// The release package must apply #ModuleRelease at package scope (not as a named
// field). The two files are loaded together as a single CUE instance so that the
// release can reference values defined in the values file.
//
// Example:
//
//	release, err := loader.LoadRelease(
//	    cueCtx,
//	    "/path/to/v1alpha1/examples/releases/minecraft/release.cue",
//	    "",  // defaults to values.cue in the same directory
//	)
func LoadRelease(cueCtx *cue.Context, releaseFile string, valuesFile string) (*modulerelease.ModuleRelease, error) {
	pkg, releaseDir, err := LoadReleasePackage(cueCtx, releaseFile, valuesFile)
	if err != nil {
		return nil, err
	}
	return LoadModuleReleaseFromValue(cueCtx, pkg, filepath.Base(releaseDir))
}

// LoadModuleReleaseFromValue decodes a ModuleRelease from an already-loaded
// CUE package value. The fallbackName is used as metadata.name if the CUE
// value does not provide one (typically the release directory name).
//
// This is called by LoadRelease after loading the CUE package, or directly
// from cmd/main.go when the package has already been loaded for kind detection.
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

	return &modulerelease.ModuleRelease{
		Metadata: relMeta,
		Module: module.Module{
			Metadata: modMeta,
			Raw:      modRaw,
		},
		DataComponents: dataComponents,
		Schema:         releaseVal,
	}, nil
}

// resolveReleaseFile normalises the releaseFile argument:
//   - If it is a directory, appends "release.cue".
//   - Otherwise returns it as-is.
func resolveReleaseFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("releaseFile must not be empty")
	}
	// If the path has no .cue extension, treat it as a directory.
	if filepath.Ext(path) != ".cue" {
		return filepath.Join(path, "release.cue"), nil
	}
	return path, nil
}

// finalizeValue produces a constraint-free data value from a CUE value by
// stripping schema constraints and taking defaults.
//
// It uses Syntax(cue.Final()) to materialise the value into an ast.Expr, then
// recompiles it with BuildExpr. This removes matchN validators, close()
// enforcement, and definition fields, leaving a plain data value suitable for
// FillPath injection into transformers.
//
// This single strategy is sufficient because finalizeValue is only ever called
// on concrete component values — self-contained Kubernetes resource specs with
// all #config references resolved through CUE unification. Such values never
// carry import declarations or unresolved definition references, so Syntax always
// returns ast.Expr (not *ast.File). If it ever returns something else, that
// indicates a bug upstream (schema constraints not resolved before finalization)
// and we surface a clear error rather than silently degrading.
func finalizeValue(cueCtx *cue.Context, v cue.Value) (cue.Value, error) {
	syntaxNode := v.Syntax(cue.Final())

	expr, ok := syntaxNode.(ast.Expr)
	if !ok {
		return cue.Value{}, fmt.Errorf(
			"finalization produced %T instead of ast.Expr; "+
				"value likely contains unresolved imports or definition fields "+
				"that should have been resolved upstream",
			syntaxNode,
		)
	}

	dataVal := cueCtx.BuildExpr(expr)
	if err := dataVal.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building finalized value: %w", err)
	}
	return dataVal, nil
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
