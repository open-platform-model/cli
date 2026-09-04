package render

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-platform-model/library/opm/kernel"
)

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
