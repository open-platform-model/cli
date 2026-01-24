package cue

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

var (
	// ErrModuleNotFound is returned when the module directory doesn't exist or is invalid.
	ErrModuleNotFound = errors.New("module not found")
	// ErrInvalidModule is returned when the module fails CUE validation.
	ErrInvalidModule = errors.New("invalid module")
)

// Loader loads OPM modules and bundles.
type Loader struct {
	ctx *cue.Context
}

// NewLoader creates a new Loader with a fresh CUE context.
func NewLoader() *Loader {
	return &Loader{
		ctx: cuecontext.New(),
	}
}

// LoadModule loads a module from a directory with optional values files.
func (l *Loader) LoadModule(ctx context.Context, dir string, valuesFiles []string) (*Module, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving module path: %w", err)
	}

	// Load the CUE instances from the directory
	cfg := &load.Config{
		Dir: absDir,
	}

	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("%w: no CUE instances found in %s", ErrModuleNotFound, absDir)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidModule, inst.Err)
	}

	// Build the CUE value
	value := l.ctx.BuildInstance(inst)
	if value.Err() != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidModule, value.Err())
	}

	// Load and unify values files
	if len(valuesFiles) > 0 {
		valuesLoader := NewValuesLoader(l.ctx)
		for _, vf := range valuesFiles {
			valuesValue, err := valuesLoader.LoadFile(ctx, vf)
			if err != nil {
				return nil, fmt.Errorf("loading values file %s: %w", vf, err)
			}
			value = value.Unify(valuesValue)
			if value.Err() != nil {
				return nil, fmt.Errorf("unifying values file %s: %w", vf, value.Err())
			}
		}
	}

	// Extract metadata
	metadata, err := extractMetadata(value)
	if err != nil {
		return nil, fmt.Errorf("extracting metadata: %w", err)
	}

	return &Module{
		Metadata:    metadata,
		Root:        value,
		Dir:         absDir,
		ValuesFiles: valuesFiles,
	}, nil
}

// LoadBundle loads a bundle from a directory with optional values files.
func (l *Loader) LoadBundle(ctx context.Context, dir string, valuesFiles []string) (*Bundle, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving bundle path: %w", err)
	}

	// Load the CUE instances from the directory
	cfg := &load.Config{
		Dir: absDir,
	}

	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("%w: no CUE instances found in %s", ErrModuleNotFound, absDir)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidModule, inst.Err)
	}

	// Build the CUE value
	value := l.ctx.BuildInstance(inst)
	if value.Err() != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidModule, value.Err())
	}

	// Load and unify values files
	if len(valuesFiles) > 0 {
		valuesLoader := NewValuesLoader(l.ctx)
		for _, vf := range valuesFiles {
			valuesValue, err := valuesLoader.LoadFile(ctx, vf)
			if err != nil {
				return nil, fmt.Errorf("loading values file %s: %w", vf, err)
			}
			value = value.Unify(valuesValue)
			if value.Err() != nil {
				return nil, fmt.Errorf("unifying values file %s: %w", vf, value.Err())
			}
		}
	}

	// Extract bundle metadata
	bundleMetadata, err := extractBundleMetadata(value)
	if err != nil {
		return nil, fmt.Errorf("extracting bundle metadata: %w", err)
	}

	return &Bundle{
		Metadata: bundleMetadata,
		Modules:  make(map[string]*Module),
		Root:     value,
		Dir:      absDir,
	}, nil
}

// extractMetadata extracts module metadata from a CUE value.
func extractMetadata(v cue.Value) (ModuleMetadata, error) {
	var metadata ModuleMetadata

	// Look for metadata field
	metaField := v.LookupPath(cue.ParsePath("metadata"))
	if !metaField.Exists() {
		// Try looking at root level for backward compatibility
		metaField = v
	}

	// Extract apiVersion
	if apiV := metaField.LookupPath(cue.ParsePath("apiVersion")); apiV.Exists() {
		if s, err := apiV.String(); err == nil {
			metadata.APIVersion = s
		}
	}

	// Extract name (required)
	nameField := metaField.LookupPath(cue.ParsePath("name"))
	if !nameField.Exists() {
		return metadata, fmt.Errorf("%w: missing required field 'name'", ErrInvalidModule)
	}
	name, err := nameField.String()
	if err != nil {
		return metadata, fmt.Errorf("%w: invalid name field: %v", ErrInvalidModule, err)
	}
	metadata.Name = name

	// Extract version (required)
	versionField := metaField.LookupPath(cue.ParsePath("version"))
	if !versionField.Exists() {
		return metadata, fmt.Errorf("%w: missing required field 'version'", ErrInvalidModule)
	}
	version, err := versionField.String()
	if err != nil {
		return metadata, fmt.Errorf("%w: invalid version field: %v", ErrInvalidModule, err)
	}
	metadata.Version = version

	// Extract description (optional)
	if descField := metaField.LookupPath(cue.ParsePath("description")); descField.Exists() {
		if desc, err := descField.String(); err == nil {
			metadata.Description = desc
		}
	}

	return metadata, nil
}

// extractBundleMetadata extracts bundle metadata from a CUE value.
func extractBundleMetadata(v cue.Value) (BundleMetadata, error) {
	var metadata BundleMetadata

	// Look for metadata field
	metaField := v.LookupPath(cue.ParsePath("metadata"))
	if !metaField.Exists() {
		metaField = v
	}

	// Extract apiVersion
	if apiV := metaField.LookupPath(cue.ParsePath("apiVersion")); apiV.Exists() {
		if s, err := apiV.String(); err == nil {
			metadata.APIVersion = s
		}
	}

	// Extract name (required)
	nameField := metaField.LookupPath(cue.ParsePath("name"))
	if !nameField.Exists() {
		return metadata, fmt.Errorf("%w: missing required field 'name' in bundle", ErrInvalidModule)
	}
	name, err := nameField.String()
	if err != nil {
		return metadata, fmt.Errorf("%w: invalid name field in bundle: %v", ErrInvalidModule, err)
	}
	metadata.Name = name

	// Extract version (optional for bundles)
	if versionField := metaField.LookupPath(cue.ParsePath("version")); versionField.Exists() {
		if version, err := versionField.String(); err == nil {
			metadata.Version = version
		}
	}

	return metadata, nil
}
