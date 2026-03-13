package render

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/loader"
)

func resolveReleaseValues(cueCtx *cue.Context, rawRelease cue.Value, releaseFilePath string, valuesFiles []string) ([]cue.Value, error) {
	if len(valuesFiles) > 0 {
		return loadValuesFiles(cueCtx, valuesFiles)
	}

	releaseDir, err := resolveReleaseDir(releaseFilePath)
	if err != nil {
		return nil, err
	}
	autoValues := filepath.Join(releaseDir, "values.cue")
	if _, statErr := os.Stat(autoValues); statErr == nil {
		valuesVal, err := loader.LoadValuesFile(cueCtx, autoValues)
		if err != nil {
			return nil, err
		}
		return []cue.Value{valuesVal}, nil
	}

	valuesVal := rawRelease.LookupPath(cue.ParsePath("values"))
	if !valuesVal.Exists() || valuesVal.Validate(cue.Concrete(true)) != nil {
		return nil, fmt.Errorf("release has no concrete values - provide --values <file> or add a values.cue to the release directory")
	}
	return []cue.Value{valuesVal}, nil
}

func loadValuesFiles(cueCtx *cue.Context, valuesFiles []string) ([]cue.Value, error) {
	valuesVals := make([]cue.Value, 0, len(valuesFiles))
	for _, valuesFile := range valuesFiles {
		valuesVal, err := loader.LoadValuesFile(cueCtx, valuesFile)
		if err != nil {
			return nil, fmt.Errorf("loading values file %q: %w", valuesFile, err)
		}
		valuesVals = append(valuesVals, valuesVal)
	}
	return valuesVals, nil
}

func resolveReleaseDir(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return filepath.Dir(path), nil
		}
		return "", fmt.Errorf("stat release path: %w", err)
	}
	if info.IsDir() {
		return path, nil
	}
	return filepath.Dir(path), nil
}

func ResolveReleaseValuesForTest(cueCtx *cue.Context, rawRelease cue.Value, releaseFilePath string, valuesFiles []string) ([]cue.Value, error) {
	return resolveReleaseValues(cueCtx, rawRelease, releaseFilePath, valuesFiles)
}
