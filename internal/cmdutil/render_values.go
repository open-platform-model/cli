package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"

	internalreleasefile "github.com/opmodel/cli/internal/releasefile"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/pkg/loader"
	"github.com/opmodel/cli/pkg/modulerelease"
)

// resolveReleaseValues resolves which loading strategy to use for a release
// file, based on whether a values file is explicitly given or can be
// auto-discovered, and returns the evaluated CUE value.
//
// Resolution order:
//  1. valuesFile is non-empty -> load the provided values files.
//  2. values.cue exists next to the release file -> auto-load that file.
//  3. Neither -> require a concrete inline values field.
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

//nolint:gocyclo // release loading combines debug-values, inline-values, and file-values cases
func loadModuleReleaseForRender(cueCtx *cue.Context, modulePath string, valuesFiles []string, debugValues bool, releaseName string) (*modulerelease.ModuleRelease, []cue.Value, error) {
	fileRelease, err := internalreleasefile.GetReleaseFile(cueCtx, modulePath)
	if err != nil {
		return nil, nil, err
	}
	if fileRelease.Kind != internalreleasefile.KindModuleRelease || fileRelease.Module == nil {
		return nil, nil, &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("unsupported release kind %q (use bundle commands for BundleRelease)", fileRelease.Kind),
		}
	}
	rel := fileRelease.Module

	var valuesVals []cue.Value
	switch {
	case debugValues && len(valuesFiles) == 0:
		modVal, modErr := loader.LoadModulePackage(cueCtx, modulePath)
		if modErr != nil {
			return nil, nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("loading module for debugValues: %w", modErr)}
		}
		debugVal := modVal.LookupPath(cue.ParsePath("debugValues"))
		if !debugVal.Exists() {
			return nil, nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("module does not define debugValues - add a debugValues field or provide a values file with -f"),
			}
		}
		if err := debugVal.Validate(cue.Concrete(true)); err != nil {
			PrintValidationError("debugValues not concrete", err)
			return nil, nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: fmt.Errorf("debugValues is not concrete - module must provide complete test values"), Printed: true}
		}
		valuesVals = []cue.Value{debugVal}
	case len(valuesFiles) > 0:
		loadedValues, loadErr := loadValuesFiles(cueCtx, valuesFiles)
		if loadErr != nil {
			return nil, nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: loadErr}
		}
		valuesVals = loadedValues
	default:
		inlineValues := rel.RawCUE.LookupPath(cue.ParsePath("values"))
		if !inlineValues.Exists() || inlineValues.Validate(cue.Concrete(true)) != nil {
			return nil, nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("release has no concrete values - provide -f <values-file> or enable debugValues")}
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

// resolveReleaseDir returns the directory containing the release file.
// When path is a directory itself, it is returned as-is.
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
