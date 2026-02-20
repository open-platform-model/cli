package release

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/build/component"
	"github.com/opmodel/cli/internal/core"
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
//     4c. Validate the full module tree for structural errors (fatal loading guard)
//  5. Inject values into #config via FillPath (makes #config concrete)
//  6. Extract concrete components from #components
//  7. Extract release metadata from the overlay-computed #opmReleaseMeta
func (b *Builder) Build(modulePath string, opts Options, valuesFiles []string) (*core.ModuleRelease, error) {
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

	// Extract #config for Module.Config (used by ValidateValues on the returned release).
	// Step 4b (values-against-config validation) is removed — it is now called explicitly
	// by the pipeline as rel.ValidateValues() after Build() returns.
	configDef := value.LookupPath(cue.ParsePath("#config"))

	// Step 4c: Validate the full module tree for any remaining errors.
	// This stays in Build() as a fatal loading guard — a module that fails CUE
	// structural validation cannot produce a usable release regardless.
	if allErrs := collectAllCUEErrors(value); allErrs != nil {
		return nil, &core.ValidationError{
			Message: "release validation failed",
			Cause:   allErrs,
			Details: formatCUEDetails(allErrs),
		}
	}

	// Step 5: Inject values into #config to make components concrete
	values := value.LookupPath(cue.ParsePath("values"))
	if !values.Exists() {
		return nil, &core.ValidationError{
			Message: "module missing 'values' field — provide values via values.cue or --values flag",
		}
	}

	concreteRelease := value.FillPath(cue.ParsePath("#config"), values)
	if allErrs := collectAllCUEErrors(concreteRelease); allErrs != nil {
		return nil, &core.ValidationError{
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

	// Step 7 (concrete component check) is removed from Build() — it is now called
	// explicitly by the pipeline as rel.Validate() after Build() returns.

	// Step 8: Extract release and module metadata from the CUE value
	relMeta := extractReleaseMetadata(concreteRelease, opts)
	modMeta := extractModuleMetadata(concreteRelease)

	// Collect component names and set on both metadata types
	componentNames := make([]string, 0, len(components))
	for name := range components {
		componentNames = append(componentNames, name)
	}
	relMeta.Components = componentNames
	modMeta.Components = append([]string{}, componentNames...)

	// Convert build/component.Component → core.Component
	coreComponents := make(map[string]*core.Component, len(components))
	for name, comp := range components {
		coreComponents[name] = convertComponent(comp)
	}

	// Build the core.Module embedding config and values for receiver method use
	mod := core.Module{
		Metadata:   &modMeta,
		ModulePath: modulePath,
		Config:     configDef,
		Values:     values,
	}
	mod.SetPkgName(pkgName)

	return &core.ModuleRelease{
		Metadata:   &relMeta,
		Module:     mod,
		Components: coreComponents,
		Values:     values,
	}, nil
}

// convertComponent converts a build/component.Component to a core.Component.
// Used when populating core.ModuleRelease.Components from the builder output.
// ApiVersion, Kind, Blueprints, and Spec are left as zero values until the
// component consolidation change unifies these two types.
func convertComponent(comp *component.Component) *core.Component {
	labels := comp.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	annotations := comp.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	return &core.Component{
		Metadata: &core.ComponentMetadata{
			Name:        comp.Name,
			Labels:      labels,
			Annotations: annotations,
		},
		Resources: comp.Resources,
		Traits:    comp.Traits,
		Value:     comp.Value,
	}
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
