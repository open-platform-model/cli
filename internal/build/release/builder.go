package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/output"
)

// Builder creates a concrete release from a module directory.
//
// It uses a CUE overlay to compute release metadata (identity, labels)
// via the CUE uuid package, and FillPath to make components concrete.
type Builder struct {
	cueCtx   *cue.Context
	registry string // CUE_REGISTRY value for module dependency resolution
}

// NewBuilder creates a new Builder.
func NewBuilder(ctx *cue.Context, registry string) *Builder {
	return &Builder{
		cueCtx:   ctx,
		registry: registry,
	}
}

// Build creates a concrete release by loading the module with a CUE overlay
// that computes release metadata (identity, labels) via the uuid package.
//
// The build process:
//  1. Determine package name (from opts.PkgName or detectPackageName fallback)
//  2. Generate AST overlay and format to bytes
//  3. Load the module directory with the overlay file
//  4. Unify with additional values files (--values)
//  5. Inject values into #config via FillPath (makes #config concrete)
//  6. Extract concrete components from #components
//  7. Validate all components are fully concrete
//  8. Extract release metadata from the overlay-computed #opmReleaseMeta
func (b *Builder) Build(modulePath string, opts Options, valuesFiles []string) (*BuiltRelease, error) {
	output.Debug("building release",
		"path", modulePath,
		"name", opts.Name,
		"namespace", opts.Namespace,
	)

	// Set CUE_REGISTRY if configured
	if b.registry != "" {
		os.Setenv("CUE_REGISTRY", b.registry)
		defer os.Unsetenv("CUE_REGISTRY")
	}

	// Step 1: Determine CUE package name
	pkgName := opts.PkgName
	if pkgName == "" {
		// Fallback: detect package name via a minimal load (backward compatibility)
		var err error
		pkgName, err = b.detectPackageName(modulePath)
		if err != nil {
			return nil, fmt.Errorf("detecting package name: %w", err)
		}
	}

	// Step 2: Generate the overlay via typed AST construction and format to bytes
	overlayFile := generateOverlayAST(pkgName, opts)
	overlayCUE, err := format.Node(overlayFile)
	if err != nil {
		return nil, fmt.Errorf("formatting overlay AST: %w", err)
	}

	// Step 3: Load the module with the overlay
	overlayPath := filepath.Join(modulePath, "opm_release_overlay.cue")
	cfg := &load.Config{
		Dir: modulePath,
		Overlay: map[string]load.Source{
			overlayPath: load.FromBytes(overlayCUE),
		},
	}

	// Stub out values files that should not participate in CUE module loading.
	//
	// CUE's load.Instances loads ALL .cue files in the directory that share
	// the same package declaration. When multiple values*.cue files exist
	// (e.g., values.cue, values_staging.cue, values_production.cue), they
	// would all be unified — causing conflicts since they define competing
	// concrete values for the same fields.
	//
	// With --values/-f: stub ALL values*.cue files; only the explicitly
	//   specified files contribute values (loaded externally via CompileBytes).
	// Without --values/-f: stub all values*.cue EXCEPT values.cue, so only
	//   the base values.cue participates in the module.
	stubPkg := []byte(fmt.Sprintf("package %s\n", pkgName))
	valuesOnDisk, _ := filepath.Glob(filepath.Join(modulePath, "values*.cue")) //nolint:errcheck // Glob only errors on malformed patterns, not file system errors
	for _, vf := range valuesOnDisk {
		if len(valuesFiles) > 0 {
			// With -f: stub every values file — external files take full precedence
			output.Debug("stubbing %s (--values flag overrides all values files)", filepath.Base(vf))
			cfg.Overlay[vf] = load.FromBytes(stubPkg)
		} else if filepath.Base(vf) != "values.cue" {
			// Without -f: stub environment-specific overrides, keep only values.cue
			output.Debug("stubbing %s (only values.cue is used without --values flag)", filepath.Base(vf))
			cfg.Overlay[vf] = load.FromBytes(stubPkg)
		}
	}

	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s", modulePath)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("loading module with overlay: %w", inst.Err)
	}

	value := b.cueCtx.BuildInstance(inst)

	// Step 4: Unify with additional values files
	for _, valuesFile := range valuesFiles {
		valuesValue, err := b.loadValuesFile(valuesFile)
		if err != nil {
			return nil, fmt.Errorf("loading values file %s: %w", valuesFile, err)
		}
		value = value.Unify(valuesValue)
	}

	// Step 4b: Validate values against #config using recursive field walking.
	configDef := value.LookupPath(cue.ParsePath("#config"))
	valuesVal := value.LookupPath(cue.ParsePath("values"))
	if configDef.Exists() && valuesVal.Exists() {
		if allErrs := validateValuesAgainstConfig(configDef, valuesVal); allErrs != nil {
			return nil, &ValidationError{
				Message: "values do not satisfy #config schema",
				Cause:   allErrs,
				Details: formatCUEDetails(allErrs),
			}
		}
	}

	// Step 4c: Validate the full module tree for any remaining errors
	if allErrs := collectAllCUEErrors(value); allErrs != nil {
		return nil, &ValidationError{
			Message: "release validation failed",
			Cause:   allErrs,
			Details: formatCUEDetails(allErrs),
		}
	}

	// Step 5: Inject values into #config to make components concrete
	values := value.LookupPath(cue.ParsePath("values"))
	if !values.Exists() {
		return nil, &ValidationError{
			Message: "module missing 'values' field — provide values via values.cue or --values flag",
		}
	}

	concreteRelease := value.FillPath(cue.ParsePath("#config"), values)
	if allErrs := collectAllCUEErrors(concreteRelease); allErrs != nil {
		return nil, &ValidationError{
			Message: "failed to inject values into #config",
			Cause:   allErrs,
			Details: formatCUEDetails(allErrs),
		}
	}

	// Step 6: Extract concrete components from #components
	components, err := extractComponentsFromDefinition(concreteRelease)
	if err != nil {
		return nil, err
	}

	// Step 7: Validate all components are concrete (collect all errors)
	var concreteErrors []error
	for name, comp := range components {
		if err := comp.Value.Validate(cue.Concrete(true)); err != nil {
			concreteErrors = append(concreteErrors, fmt.Errorf("component %q: %w", name, err))
		}
	}
	if len(concreteErrors) > 0 {
		var details strings.Builder
		for _, cerr := range concreteErrors {
			details.WriteString(formatCUEDetails(cerr))
			details.WriteByte('\n')
		}
		return nil, &ValidationError{
			Message: fmt.Sprintf("%d component(s) have non-concrete values - check that all required values are provided", len(concreteErrors)),
			Cause:   concreteErrors[0],
			Details: strings.TrimSpace(details.String()),
		}
	}

	// Step 8: Extract release metadata from overlay-computed #opmReleaseMeta
	metadata := extractReleaseMetadata(concreteRelease, opts)

	return &BuiltRelease{
		Value:      concreteRelease,
		Components: components,
		Metadata:   metadata,
	}, nil
}

// InspectModule extracts module metadata from a module directory using AST
// inspection without CUE evaluation.
func (b *Builder) InspectModule(modulePath string) (*module.Inspection, error) {
	if b.registry != "" {
		os.Setenv("CUE_REGISTRY", b.registry)
		defer os.Unsetenv("CUE_REGISTRY")
	}

	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s", modulePath)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("loading module for inspection: %w", inst.Err)
	}

	name, defaultNamespace := module.ExtractMetadataFromAST(inst.Files)

	output.Debug("inspected module via AST",
		"pkgName", inst.PkgName,
		"name", name,
		"defaultNamespace", defaultNamespace,
	)

	return &module.Inspection{
		Name:             name,
		DefaultNamespace: defaultNamespace,
		PkgName:          inst.PkgName,
	}, nil
}

// detectPackageName loads the module directory minimally to determine the CUE package name.
func (b *Builder) detectPackageName(modulePath string) (string, error) {
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return "", fmt.Errorf("no CUE instances found in %s", modulePath)
	}
	inst := instances[0]
	if inst.Err != nil {
		return "", fmt.Errorf("loading module for package detection: %w", inst.Err)
	}
	if inst.PkgName == "" {
		return "", fmt.Errorf("module has no package name: %s", modulePath)
	}
	return inst.PkgName, nil
}

// loadValuesFile loads a single values file and compiles it.
func (b *Builder) loadValuesFile(path string) (cue.Value, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return cue.Value{}, fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return cue.Value{}, fmt.Errorf("file not found: %s", absPath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return cue.Value{}, fmt.Errorf("reading file: %w", err)
	}

	value := b.cueCtx.CompileBytes(content, cue.Filename(absPath))
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("compiling values: %w", value.Err())
	}

	return value, nil
}

// CueContext returns the CUE context (used by pipeline for fallback metadata extraction).
func (b *Builder) CueContext() *cue.Context {
	return b.cueCtx
}
