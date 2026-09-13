package render

import (
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// ResolveModuleValues resolves the values sources of a module package the
// way every module-directory command layers them: each -f/--values file, in
// declaration order, as a file-backed kernel source attributed to that file;
// without -f files the module's own debugValues as the single source,
// attributed to <moduleDir>/debugValues (DebugValuesSource). The sources
// come back unvalidated: the caller checks them against the module's
// #config through the kernel (vet), or leaves that to the synthesized build
// (build, apply), so no source is validated twice.
func ResolveModuleValues(k *kernel.Kernel, pkg cue.Value, moduleDir string, valuesFiles []string) ([]kernel.Source, error) {
	if len(valuesFiles) > 0 {
		return loadValuesSources(k, valuesFiles)
	}
	src, err := DebugValuesSource(k, pkg, filepath.Join(moduleDir, "debugValues"))
	if err != nil {
		return nil, err
	}
	return []kernel.Source{src}, nil
}

// DebugValuesSource turns a module package's debugValues into a kernel
// values source — the layering policy the kernel leaves to its frontends
// (synthesis never falls back to debugValues on its own). The field is
// rendered back to canonical CUE, exactly as the kernel renders every values
// source into the synthesized package's values file, and compiled as a
// source whose Origin is the caller's name for the field (a #config
// violation is then attributed to the module's debugValues rather than to a
// file that does not exist). A module declaring no debugValues is an error
// naming the -f alternative.
func DebugValuesSource(k *kernel.Kernel, pkg cue.Value, origin string) (kernel.Source, error) {
	debugVal := pkg.LookupPath(schema.DebugValues)
	if !debugVal.Exists() {
		return kernel.Source{}, fmt.Errorf("module does not define debugValues - add debugValues or provide values with -f")
	}
	data, err := format.Node(debugVal.Syntax(cue.Final(), cue.Concrete(false)))
	if err != nil {
		return kernel.Source{}, fmt.Errorf("rendering debugValues: %w", err)
	}
	return k.LoadSourceFromBytes(origin, data)
}

// loadValuesSources loads every -f/--values file as a kernel values source,
// in declaration order (stack order for layering): each source's Origin is
// the file's absolute path, so a conflict or a schema violation is attributed
// to the file the user wrote. An empty list yields nil (no sources: the
// package's own values apply).
func loadValuesSources(k *kernel.Kernel, valuesFiles []string) ([]kernel.Source, error) {
	if len(valuesFiles) == 0 {
		return nil, nil
	}
	sources := make([]kernel.Source, 0, len(valuesFiles))
	for _, valuesFile := range valuesFiles {
		src, err := k.LoadSourceFromFile(valuesFile)
		if err != nil {
			return nil, fmt.Errorf("loading values file %q: %w", valuesFile, err)
		}
		sources = append(sources, src)
	}
	return sources, nil
}
