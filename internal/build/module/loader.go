package module

import (
	"fmt"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/output"
)

// Load constructs a *core.Module by resolving the module path, running AST
// inspection, and populating module metadata. It is the primary constructor
// for core.Module in the build pipeline.
//
// Load calls mod.ResolvePath() internally — the returned *core.Module always
// has a validated, absolute ModulePath. The cueCtx parameter is reserved for
// future use (e.g., if a CUE evaluation step is reintroduced); currently only
// AST inspection is performed.
//
// If AST inspection cannot extract metadata.name as a string literal (e.g.,
// computed expressions), Metadata.Name will be empty. mod.Validate() will
// catch this and return a fatal error before the build phase.
func Load(_ *cue.Context, modulePath, registry string) (*core.Module, error) {
	mod := &core.Module{ModulePath: modulePath}

	// Step 1: Resolve and validate the module path
	if err := mod.ResolvePath(); err != nil {
		return nil, err
	}

	// Step 2: AST inspection — extract name, defaultNamespace, pkgName
	inspection, err := inspectModule(mod.ModulePath, registry)
	if err != nil {
		return nil, err
	}

	// Step 3: Populate metadata from inspection
	mod.Metadata = &core.ModuleMetadata{
		Name:             inspection.Name,
		DefaultNamespace: inspection.DefaultNamespace,
	}
	mod.SetPkgName(inspection.PkgName)

	output.Debug("loaded module",
		"path", mod.ModulePath,
		"name", mod.Metadata.Name,
		"defaultNamespace", mod.Metadata.DefaultNamespace,
		"pkgName", mod.PkgName(),
	)

	return mod, nil
}

// inspectModule extracts module metadata from a module directory using AST
// inspection without CUE evaluation. It performs a single load.Instances call,
// reads inst.PkgName, and walks inst.Files to extract metadata.name and
// metadata.defaultNamespace as string literals.
//
// If metadata fields are not string literals (e.g., computed expressions),
// the corresponding fields in Inspection will be empty.
func inspectModule(modulePath, registry string) (*Inspection, error) {
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
