package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/token"

	"github.com/opmodel/cli/internal/output"
)

// opmNamespaceUUID is the UUID v5 namespace for computing deterministic identities.
// Computed as: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("opmodel.dev")).String()
// Used by the CUE overlay to compute release identity via uuid.SHA1.
const opmNamespaceUUID = "c1cbe76d-5687-5a47-bfe6-83b081b15413"

// ReleaseBuilder creates a concrete release from a module directory.
//
// It uses a CUE overlay to compute release metadata (identity, labels)
// via the CUE uuid package, and FillPath to make components concrete.
type ReleaseBuilder struct {
	cueCtx   *cue.Context
	registry string // CUE_REGISTRY value for module dependency resolution
}

// NewReleaseBuilder creates a new ReleaseBuilder.
func NewReleaseBuilder(ctx *cue.Context, registry string) *ReleaseBuilder {
	return &ReleaseBuilder{
		cueCtx:   ctx,
		registry: registry,
	}
}

// ReleaseOptions configures release building.
type ReleaseOptions struct {
	Name      string // Release name (defaults to module name)
	Namespace string // Required: target namespace
	PkgName   string // Internal: CUE package name (set by InspectModule, skip detectPackageName)
}

// ModuleInspection contains metadata extracted from a module directory
// via AST inspection without CUE evaluation.
type ModuleInspection struct {
	Name             string // metadata.name (empty if not a string literal)
	DefaultNamespace string // metadata.defaultNamespace (empty if not a string literal)
	PkgName          string // CUE package name from inst.PkgName
}

// BuiltRelease is the result of building a release.
type BuiltRelease struct {
	Value      cue.Value                   // The concrete module value (with #config injected)
	Components map[string]*LoadedComponent // Concrete components by name
	Metadata   ReleaseMetadata
}

// ReleaseMetadata contains release-level metadata.
type ReleaseMetadata struct {
	Name      string
	Namespace string
	Version   string
	FQN       string
	Labels    map[string]string
	// Identity is the module identity UUID (from #Module.metadata.identity).
	Identity string
	// ReleaseIdentity is the release identity UUID.
	// Computed by the CUE overlay via uuid.SHA1(OPMNamespace, "fqn:name:namespace").
	ReleaseIdentity string
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
func (b *ReleaseBuilder) Build(modulePath string, opts ReleaseOptions, valuesFiles []string) (*BuiltRelease, error) {
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
	overlayFile := b.generateOverlayAST(pkgName, opts)
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

	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s", modulePath)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("loading module with overlay: %w", inst.Err)
	}

	value := b.cueCtx.BuildInstance(inst)
	if value.Err() != nil {
		return nil, fmt.Errorf("building module with overlay: %w", value.Err())
	}

	// Step 4: Unify with additional values files
	for _, valuesFile := range valuesFiles {
		valuesValue, err := b.loadValuesFile(valuesFile)
		if err != nil {
			return nil, fmt.Errorf("loading values file %s: %w", valuesFile, err)
		}
		value = value.Unify(valuesValue)
		if value.Err() != nil {
			return nil, fmt.Errorf("unifying values from %s: %w", valuesFile, value.Err())
		}
	}

	// Step 5: Inject values into #config to make components concrete
	values := value.LookupPath(cue.ParsePath("values"))
	if !values.Exists() {
		return nil, &ReleaseValidationError{
			Message: "module missing 'values' field - ensure module uses #config pattern",
		}
	}

	concreteModule := value.FillPath(cue.ParsePath("#config"), values)
	if concreteModule.Err() != nil {
		return nil, &ReleaseValidationError{
			Message: "failed to inject values into #config",
			Cause:   concreteModule.Err(),
		}
	}

	// Step 6: Extract concrete components from #components
	components, err := b.extractComponentsFromDefinition(concreteModule)
	if err != nil {
		return nil, err
	}

	// Step 7: Validate components are concrete
	for name, comp := range components {
		if err := comp.Value.Validate(cue.Concrete(true)); err != nil {
			return nil, &ReleaseValidationError{
				Message: fmt.Sprintf("component %q has non-concrete values - check that all required values are provided", name),
				Cause:   err,
			}
		}
	}

	// Step 8: Extract release metadata from overlay-computed #opmReleaseMeta
	metadata := b.extractReleaseMetadata(concreteModule, opts)

	output.Debug("release built successfully",
		"name", metadata.Name,
		"namespace", metadata.Namespace,
		"components", len(components),
	)

	return &BuiltRelease{
		Value:      concreteModule,
		Components: components,
		Metadata:   metadata,
	}, nil
}

// BuildFromValue creates a concrete release from an already-loaded module value.
// This is the legacy path for modules that don't import opmodel.dev/core@v0
// (e.g., test fixtures with inline CUE).
//
// The build process:
//  1. Extract values from module.values
//  2. Inject values into #config via FillPath (makes #config concrete)
//  3. Extract components from #components
//  4. Validate all components are fully concrete
//  5. Extract metadata from the module (including identity)
func (b *ReleaseBuilder) BuildFromValue(moduleValue cue.Value, opts ReleaseOptions) (*BuiltRelease, error) {
	output.Debug("building release (legacy)", "name", opts.Name, "namespace", opts.Namespace)

	// Step 1: Extract values from module
	values := moduleValue.LookupPath(cue.ParsePath("values"))
	if !values.Exists() {
		return nil, &ReleaseValidationError{
			Message: "module missing 'values' field - ensure module uses #config pattern",
		}
	}

	// Step 2: Inject values into #config to make it concrete
	concreteModule := moduleValue.FillPath(cue.ParsePath("#config"), values)
	if concreteModule.Err() != nil {
		return nil, &ReleaseValidationError{
			Message: "failed to inject values into #config",
			Cause:   concreteModule.Err(),
		}
	}

	// Step 3: Extract concrete components from #components
	components, err := b.extractComponentsFromDefinition(concreteModule)
	if err != nil {
		return nil, err
	}

	// Step 4: Validate components are concrete
	for name, comp := range components {
		if err := comp.Value.Validate(cue.Concrete(true)); err != nil {
			return nil, &ReleaseValidationError{
				Message: fmt.Sprintf("component %q has non-concrete values - check that all required values are provided", name),
				Cause:   err,
			}
		}
	}

	// Step 5: Extract metadata from the module
	metadata := b.extractMetadataFromModule(concreteModule, opts)

	output.Debug("release built successfully",
		"name", metadata.Name,
		"namespace", metadata.Namespace,
		"components", len(components),
	)

	return &BuiltRelease{
		Value:      concreteModule,
		Components: components,
		Metadata:   metadata,
	}, nil
}

// detectPackageName loads the module directory minimally to determine the CUE package name.
// This is the backward-compatibility fallback when opts.PkgName is not set.
// Retained for direct Build() callers that don't go through InspectModule.
// Prefer using InspectModule which also extracts metadata without evaluation.
func (b *ReleaseBuilder) detectPackageName(modulePath string) (string, error) {
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

// InspectModule extracts module metadata from a module directory using AST
// inspection without CUE evaluation. It performs a single load.Instances call,
// reads inst.PkgName, and walks inst.Files to extract metadata.name and
// metadata.defaultNamespace as string literals.
//
// If metadata fields are not string literals (e.g., computed expressions),
// the corresponding fields in ModuleInspection will be empty.
func (b *ReleaseBuilder) InspectModule(modulePath string) (*ModuleInspection, error) {
	// Set CUE_REGISTRY if configured (needed for modules with registry imports)
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

	name, defaultNamespace := extractMetadataFromAST(inst.Files)

	output.Debug("inspected module via AST",
		"pkgName", inst.PkgName,
		"name", name,
		"defaultNamespace", defaultNamespace,
	)

	return &ModuleInspection{
		Name:             name,
		DefaultNamespace: defaultNamespace,
		PkgName:          inst.PkgName,
	}, nil
}

// extractMetadataFromAST walks CUE AST files to extract metadata.name and
// metadata.defaultNamespace as string literals without CUE evaluation.
// Returns empty strings for fields that are not static string literals.
func extractMetadataFromAST(files []*ast.File) (name, defaultNamespace string) {
	for _, file := range files {
		for _, decl := range file.Decls {
			field, ok := decl.(*ast.Field)
			if !ok {
				continue
			}
			ident, ok := field.Label.(*ast.Ident)
			if !ok || ident.Name != "metadata" {
				continue
			}
			structLit, ok := field.Value.(*ast.StructLit)
			if !ok {
				continue
			}
			n, ns := extractFieldsFromMetadataStruct(structLit)
			if n != "" && name == "" {
				name = n
			}
			if ns != "" && defaultNamespace == "" {
				defaultNamespace = ns
			}
		}
		if name != "" && defaultNamespace != "" {
			break
		}
	}
	return name, defaultNamespace
}

// extractFieldsFromMetadataStruct scans a metadata struct literal for
// name and defaultNamespace fields with string literal values.
func extractFieldsFromMetadataStruct(s *ast.StructLit) (name, defaultNamespace string) {
	for _, elt := range s.Elts {
		innerField, ok := elt.(*ast.Field)
		if !ok {
			continue
		}
		innerIdent, ok := innerField.Label.(*ast.Ident)
		if !ok {
			continue
		}
		lit, ok := innerField.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		switch innerIdent.Name {
		case "name":
			name = strings.Trim(lit.Value, `"`)
		case "defaultNamespace":
			defaultNamespace = strings.Trim(lit.Value, `"`)
		}
	}
	return name, defaultNamespace
}

// generateOverlayAST builds the CUE overlay file as a typed AST.
//
// The overlay adds a #opmReleaseMeta definition to the module's CUE package.
// This definition computes:
//   - Release identity (uuid.SHA1 of fqn:name:namespace)
//   - Standard release labels (module-release.opmodel.dev/*)
//   - Module labels (inherited from module.metadata.labels)
//
// Key rules:
//   - Field labels referenced from nested scopes (name, namespace, fqn, version, identity)
//     use ast.NewIdent (unquoted identifier labels) for CUE scope resolution.
//   - Label keys with special characters use ast.NewString (quoted string labels).
//   - astutil.Resolve is called after construction to wire up scope references.
func (b *ReleaseBuilder) generateOverlayAST(pkgName string, opts ReleaseOptions) *ast.File {
	// Build the uuid.SHA1(...) call expression
	uuidCall := ast.NewCall(
		&ast.SelectorExpr{
			X:   ast.NewIdent("uuid"),
			Sel: ast.NewIdent("SHA1"),
		},
		ast.NewString(opmNamespaceUUID),
		// CUE string interpolation: "\(fqn):\(name):\(namespace)"
		// Interpolation Elts are interleaved: string fragments include
		// quote chars and \( / ) delimiters, matching parser output.
		&ast.Interpolation{
			Elts: []ast.Expr{
				ast.NewLit(token.STRING, `"\(`),
				ast.NewIdent("fqn"),
				ast.NewLit(token.STRING, `):\(`),
				ast.NewIdent("name"),
				ast.NewLit(token.STRING, `):\(`),
				ast.NewIdent("namespace"),
				ast.NewLit(token.STRING, `)"`),
			},
		},
	)

	// identity: string & uuid.SHA1(...)
	identityExpr := &ast.BinaryExpr{
		X:  ast.NewIdent("string"),
		Op: token.AND,
		Y:  uuidCall,
	}

	// labels: metadata.labels & { ... }
	// Label keys use ast.NewString (quoted) because they contain special chars.
	// The values (name, version, identity) are ast.NewIdent references to sibling fields.
	labelsExpr := &ast.BinaryExpr{
		X: &ast.SelectorExpr{
			X:   ast.NewIdent("metadata"),
			Sel: ast.NewIdent("labels"),
		},
		Op: token.AND,
		Y: ast.NewStruct(
			ast.NewString("module-release.opmodel.dev/name"), ast.NewIdent("name"),
			ast.NewString("module-release.opmodel.dev/version"), ast.NewIdent("version"),
			ast.NewString("module-release.opmodel.dev/uuid"), ast.NewIdent("identity"),
		),
	}

	// Build #opmReleaseMeta struct with *ast.Field entries using ast.NewIdent labels.
	// Using ast.NewIdent for labels produces unquoted identifiers,
	// which CUE can resolve as references from nested scopes.
	releaseMetaStruct := ast.NewStruct(
		&ast.Field{Label: ast.NewIdent("name"), Value: ast.NewString(opts.Name)},
		&ast.Field{Label: ast.NewIdent("namespace"), Value: ast.NewString(opts.Namespace)},
		&ast.Field{
			Label: ast.NewIdent("fqn"),
			Value: &ast.SelectorExpr{
				X:   ast.NewIdent("metadata"),
				Sel: ast.NewIdent("fqn"),
			},
		},
		&ast.Field{
			Label: ast.NewIdent("version"),
			Value: &ast.SelectorExpr{
				X:   ast.NewIdent("metadata"),
				Sel: ast.NewIdent("version"),
			},
		},
		&ast.Field{Label: ast.NewIdent("identity"), Value: identityExpr},
		&ast.Field{Label: ast.NewIdent("labels"), Value: labelsExpr},
	)

	file := &ast.File{
		Decls: []ast.Decl{
			&ast.Package{Name: ast.NewIdent(pkgName)},
			&ast.ImportDecl{
				Specs: []*ast.ImportSpec{
					ast.NewImport(nil, "uuid"),
				},
			},
			&ast.Field{
				Label: ast.NewIdent("#opmReleaseMeta"),
				Value: releaseMetaStruct,
			},
		},
	}

	// Resolve scope references so that identifiers like `name` inside the
	// labels struct can find the `name` field in the parent #opmReleaseMeta struct.
	astutil.Resolve(file, func(_ token.Pos, msg string, args ...interface{}) {
		// Ignore resolution errors — some references (like `metadata`) are external
	})

	return file
}

// loadValuesFile loads a single values file and compiles it.
func (b *ReleaseBuilder) loadValuesFile(path string) (cue.Value, error) {
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

// extractComponentsFromDefinition extracts components from #components (definition).
func (b *ReleaseBuilder) extractComponentsFromDefinition(concreteModule cue.Value) (map[string]*LoadedComponent, error) {
	componentsValue := concreteModule.LookupPath(cue.ParsePath("#components"))
	if !componentsValue.Exists() {
		return nil, fmt.Errorf("module missing '#components' field")
	}

	components := make(map[string]*LoadedComponent)

	iter, err := componentsValue.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating components: %w", err)
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		compValue := iter.Value()

		comp := b.extractComponent(name, compValue)
		components[name] = comp
	}

	return components, nil
}

// extractComponent extracts a single component with its metadata.
func (b *ReleaseBuilder) extractComponent(name string, value cue.Value) *LoadedComponent {
	comp := &LoadedComponent{
		Name:        name,
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Resources:   make(map[string]cue.Value),
		Traits:      make(map[string]cue.Value),
		Value:       value,
	}

	// Extract metadata.name if present, otherwise use field name
	metaName := value.LookupPath(cue.ParsePath("metadata.name"))
	if metaName.Exists() {
		if str, err := metaName.String(); err == nil {
			comp.Name = str
		}
	}

	// Extract #resources
	resourcesValue := value.LookupPath(cue.ParsePath("#resources"))
	if resourcesValue.Exists() {
		iter, err := resourcesValue.Fields()
		if err == nil {
			for iter.Next() {
				fqn := iter.Selector().Unquoted()
				comp.Resources[fqn] = iter.Value()
			}
		}
	}

	// Extract #traits
	traitsValue := value.LookupPath(cue.ParsePath("#traits"))
	if traitsValue.Exists() {
		iter, err := traitsValue.Fields()
		if err == nil {
			for iter.Next() {
				fqn := iter.Selector().Unquoted()
				comp.Traits[fqn] = iter.Value()
			}
		}
	}

	// Extract annotations from metadata
	b.extractAnnotations(value, comp.Annotations)

	// Extract labels from metadata
	labelsValue := value.LookupPath(cue.ParsePath("metadata.labels"))
	if labelsValue.Exists() {
		iter, err := labelsValue.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					comp.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}

	return comp
}

// extractAnnotations extracts annotations from component metadata into the target map.
// CUE annotation values (bool, string) are converted to strings.
func (b *ReleaseBuilder) extractAnnotations(value cue.Value, annotations map[string]string) {
	annotationsValue := value.LookupPath(cue.ParsePath("metadata.annotations"))
	if !annotationsValue.Exists() {
		return
	}
	iter, err := annotationsValue.Fields()
	if err != nil {
		return
	}
	for iter.Next() {
		key := iter.Selector().Unquoted()
		v := iter.Value()
		switch v.Kind() {
		case cue.BoolKind:
			if b, err := v.Bool(); err == nil {
				if b {
					annotations[key] = "true"
				} else {
					annotations[key] = "false"
				}
			}
		default:
			if str, err := v.String(); err == nil {
				annotations[key] = str
			}
		}
	}
}

// extractReleaseMetadata extracts release metadata from the overlay-computed
// #opmReleaseMeta definition and module metadata.
//
// The overlay computes identity, labels, fqn, and version via CUE.
// Module identity comes from metadata.identity (computed by #Module).
func (b *ReleaseBuilder) extractReleaseMetadata(concreteModule cue.Value, opts ReleaseOptions) ReleaseMetadata {
	metadata := ReleaseMetadata{
		Name:      opts.Name,
		Namespace: opts.Namespace,
		Labels:    make(map[string]string),
	}

	// Extract from overlay-computed #opmReleaseMeta
	relMeta := concreteModule.LookupPath(cue.ParsePath("#opmReleaseMeta"))
	if relMeta.Exists() && relMeta.Err() == nil {
		// Version
		if v := relMeta.LookupPath(cue.ParsePath("version")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.Version = str
			}
		}

		// FQN
		if v := relMeta.LookupPath(cue.ParsePath("fqn")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.FQN = str
			}
		}

		// Release identity (computed by CUE uuid.SHA1)
		if v := relMeta.LookupPath(cue.ParsePath("identity")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.ReleaseIdentity = str
			}
		}

		// Labels (includes module labels + standard release labels)
		if labelsVal := relMeta.LookupPath(cue.ParsePath("labels")); labelsVal.Exists() {
			iter, err := labelsVal.Fields()
			if err == nil {
				for iter.Next() {
					if str, err := iter.Value().String(); err == nil {
						metadata.Labels[iter.Selector().Unquoted()] = str
					}
				}
			}
		}
	} else {
		// Fallback: extract from module metadata directly (no overlay)
		b.extractMetadataFallback(concreteModule, &metadata)
	}

	// Extract module identity from metadata.identity (always from module, not release)
	if v := concreteModule.LookupPath(cue.ParsePath("metadata.identity")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Identity = str
		}
	}

	return metadata
}

// extractMetadataFallback extracts metadata from module fields when overlay is not available.
func (b *ReleaseBuilder) extractMetadataFallback(concreteModule cue.Value, metadata *ReleaseMetadata) {
	if v := concreteModule.LookupPath(cue.ParsePath("metadata.version")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Version = str
		}
	}

	if v := concreteModule.LookupPath(cue.ParsePath("metadata.fqn")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.FQN = str
		}
	}
	if metadata.FQN == "" {
		if v := concreteModule.LookupPath(cue.ParsePath("metadata.apiVersion")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.FQN = str
			}
		}
	}

	if labelsVal := concreteModule.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		iter, err := labelsVal.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					metadata.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}
}

// extractMetadataFromModule extracts metadata from a module value (legacy path).
// Used by BuildFromValue for modules without overlay support.
func (b *ReleaseBuilder) extractMetadataFromModule(concreteModule cue.Value, opts ReleaseOptions) ReleaseMetadata {
	metadata := ReleaseMetadata{
		Name:      opts.Name,
		Namespace: opts.Namespace,
		Labels:    make(map[string]string),
	}

	if v := concreteModule.LookupPath(cue.ParsePath("metadata.version")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Version = str
		}
	}

	if v := concreteModule.LookupPath(cue.ParsePath("metadata.fqn")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.FQN = str
		}
	}
	if metadata.FQN == "" {
		if v := concreteModule.LookupPath(cue.ParsePath("metadata.apiVersion")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.FQN = str
			}
		}
	}

	if v := concreteModule.LookupPath(cue.ParsePath("metadata.identity")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Identity = str
		}
	}

	if labelsVal := concreteModule.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		iter, err := labelsVal.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					metadata.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}

	return metadata
}
