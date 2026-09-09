package render

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// DebugValuesSource turns a module's debugValues into a kernel values source —
// the layering policy the kernel leaves to its frontends (synthesis never
// falls back to debugValues on its own). The field is rendered back to
// canonical CUE, exactly as the kernel renders every values source into the
// synthesized package's values file, and compiled as a source whose Origin is
// the caller's name for the field (a #config violation is then attributed to
// the module's debugValues rather than to a file that does not exist). A
// module declaring no debugValues is an error naming the -f alternative.
func DebugValuesSource(k *kernel.Kernel, mod *module.Module, origin string) (kernel.Source, error) {
	debugVal := mod.Package.LookupPath(schema.DebugValues)
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

// resolveInstanceDir returns the CUE package directory for an instance path:
// the path itself when it is a directory, else its parent.
func resolveInstanceDir(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return filepath.Dir(path), nil
		}
		return "", fmt.Errorf("stat instance path: %w", err)
	}
	if info.IsDir() {
		return path, nil
	}
	return filepath.Dir(path), nil
}
