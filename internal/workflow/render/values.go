package render

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/cli/pkg/loader"
)

// unifyValuesFiles loads every -f/--values file and unifies them in
// declaration order into a single cue.Value — the kernel's synthesis and
// processing take one values input. The zero cue.Value means "no files
// given" (the caller's fallback applies).
func unifyValuesFiles(cueCtx *cue.Context, valuesFiles []string) (cue.Value, error) {
	if len(valuesFiles) == 0 {
		return cue.Value{}, nil
	}
	var unified cue.Value
	for i, valuesFile := range valuesFiles {
		valuesVal, err := loader.LoadValuesFile(cueCtx, valuesFile)
		if err != nil {
			return cue.Value{}, fmt.Errorf("loading values file %q: %w", valuesFile, err)
		}
		if i == 0 {
			unified = valuesVal
			continue
		}
		unified = unified.Unify(valuesVal)
	}
	if err := unified.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("unifying values files: %w", err)
	}
	return unified, nil
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
