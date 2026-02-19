package module

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/output"
)

// ResolvePath validates and resolves the module directory path.
func ResolvePath(modulePath string) (string, error) {
	absPath, err := filepath.Abs(modulePath)
	if err != nil {
		return "", fmt.Errorf("resolving module path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("module directory not found: %s", absPath)
	}

	cueModPath := filepath.Join(absPath, "cue.mod")
	if _, err := os.Stat(cueModPath); os.IsNotExist(err) {
		return "", fmt.Errorf("not a CUE module: missing cue.mod/ directory in %s", absPath)
	}

	return absPath, nil
}

// InspectModule extracts module metadata from a module directory using AST
// inspection without CUE evaluation. It performs a single load.Instances call,
// reads inst.PkgName, and walks inst.Files to extract metadata.name and
// metadata.defaultNamespace as string literals.
//
// If metadata fields are not string literals (e.g., computed expressions),
// the corresponding fields in Inspection will be empty.
func InspectModule(modulePath, registry string) (*Inspection, error) {
	if registry != "" {
		os.Setenv("CUE_REGISTRY", registry)
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

	name, defaultNamespace := ExtractMetadataFromAST(inst.Files)

	output.Debug("inspected module via AST",
		"pkgName", inst.PkgName,
		"name", name,
		"defaultNamespace", defaultNamespace,
	)

	return &Inspection{
		Name:             name,
		DefaultNamespace: defaultNamespace,
		PkgName:          inst.PkgName,
	}, nil
}

// ExtractMetadata does a lightweight CUE load to extract module name
// and defaultNamespace without building the full module release.
// Falls back to CUE evaluation when AST inspection returns empty strings.
func ExtractMetadata(cueCtx *cue.Context, modulePath, registry string) (*MetadataPreview, error) {
	if registry != "" {
		os.Setenv("CUE_REGISTRY", registry)
		defer os.Unsetenv("CUE_REGISTRY")
	}

	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s", modulePath)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("loading module: %w", inst.Err)
	}

	value := cueCtx.BuildInstance(inst)
	if value.Err() != nil {
		return nil, fmt.Errorf("building module: %w", value.Err())
	}

	meta := &MetadataPreview{}

	// Extract name from metadata
	for _, path := range []string{"metadata.name", "module.metadata.name"} {
		if v := value.LookupPath(cue.ParsePath(path)); v.Exists() {
			if str, err := v.String(); err == nil {
				meta.Name = str
				break
			}
		}
	}

	// Extract defaultNamespace from metadata
	for _, path := range []string{"metadata.defaultNamespace", "module.metadata.defaultNamespace"} {
		if v := value.LookupPath(cue.ParsePath(path)); v.Exists() {
			if str, err := v.String(); err == nil {
				meta.DefaultNamespace = str
				break
			}
		}
	}

	output.Debug("extracted module metadata",
		"name", meta.Name,
		"defaultNamespace", meta.DefaultNamespace,
	)

	return meta, nil
}
