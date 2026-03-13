package render

import (
	"fmt"
	"os"
	"path/filepath"

	opmexit "github.com/opmodel/cli/internal/exit"

	"cuelang.org/go/cue"

	internalreleasefile "github.com/opmodel/cli/internal/releasefile"
	"github.com/opmodel/cli/pkg/loader"
	pkgrender "github.com/opmodel/cli/pkg/render"
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

func loadModuleReleaseForRender(cueCtx *cue.Context, modulePath string, valuesFiles []string, debugValues bool, releaseName string) (*pkgrender.ModuleRelease, []cue.Value, error) {
	fileRelease, err := internalreleasefile.GetReleaseFile(cueCtx, modulePath)
	if err != nil {
		return nil, nil, err
	}
	if fileRelease.Kind != internalreleasefile.KindModuleRelease || fileRelease.Module == nil {
		return nil, nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("unsupported release kind %q (use bundle commands for BundleRelease)", fileRelease.Kind)}
	}
	rel := fileRelease.Module

	var valuesVals []cue.Value
	switch {
	case debugValues && len(valuesFiles) == 0:
		modVal, modErr := loader.LoadModulePackage(cueCtx, modulePath)
		if modErr != nil {
			return nil, nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("loading module for debugValues: %w", modErr)}
		}
		debugVal := modVal.LookupPath(cue.ParsePath("debugValues"))
		if !debugVal.Exists() {
			return nil, nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("module does not define debugValues - add a debugValues field or provide a values file with -f")}
		}
		if err := debugVal.Validate(cue.Concrete(true)); err != nil {
			printValidationError("debugValues not concrete", err)
			return nil, nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf("debugValues is not concrete - module must provide complete test values"), Printed: true}
		}
		valuesVals = []cue.Value{debugVal}
	case len(valuesFiles) > 0:
		loadedValues, loadErr := loadValuesFiles(cueCtx, valuesFiles)
		if loadErr != nil {
			return nil, nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: loadErr}
		}
		valuesVals = loadedValues
	default:
		inlineValues := rel.RawCUE.LookupPath(cue.ParsePath("values"))
		if !inlineValues.Exists() || inlineValues.Validate(cue.Concrete(true)) != nil {
			return nil, nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("release has no concrete values - provide -f <values-file> or enable debugValues")}
		}
		valuesVals = []cue.Value{inlineValues}
	}

	if releaseName != "" {
		rel.Metadata.Name = releaseName
	}
	return rel, valuesVals, nil
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

func LoadModuleReleaseForTest(cueCtx *cue.Context, modulePath string, valuesFiles []string, debugValues bool, releaseName string) (*pkgrender.ModuleRelease, []cue.Value, error) {
	return loadModuleReleaseForRender(cueCtx, modulePath, valuesFiles, debugValues, releaseName)
}
