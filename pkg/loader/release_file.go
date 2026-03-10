package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
)

// LoadReleaseFile loads a #ModuleRelease or #BundleRelease from a standalone
// .cue file. CUE imports (including registry module references) are resolved
// via load.Instances() using the file's parent directory for cue.mod resolution.
//
// The returned cue.Value may have #module unfilled if the release file does not
// import a module. The caller is responsible for filling #module via FillPath
// when --module is provided.
//
// Returns the evaluated CUE value and the directory used for CUE resolution.
func LoadReleaseFile(ctx *cue.Context, filePath, registry string) (cue.Value, string, error) {
	var err error
	filePath, err = resolveReleaseFile(filePath)
	if err != nil {
		return cue.Value{}, "", err
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("resolving release file path: %w", err)
	}

	parentDir := filepath.Dir(absPath)

	// Set CUE_REGISTRY env var if registry is non-empty, restoring original after.
	if registry != "" {
		orig, hadOrig := os.LookupEnv("CUE_REGISTRY")
		if err := os.Setenv("CUE_REGISTRY", registry); err != nil {
			return cue.Value{}, "", fmt.Errorf("setting CUE_REGISTRY: %w", err)
		}
		defer func() {
			if hadOrig {
				_ = os.Setenv("CUE_REGISTRY", orig)
			} else {
				_ = os.Unsetenv("CUE_REGISTRY")
			}
		}()
	}

	cfg := &load.Config{
		Dir: parentDir,
	}
	instances := load.Instances([]string{filepath.Base(absPath)}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, "", fmt.Errorf("no CUE instances found for %s", absPath)
	}
	if instances[0].Err != nil {
		return cue.Value{}, "", fmt.Errorf("loading release file: %w", instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, "", fmt.Errorf("building release file: %w", err)
	}

	return val, parentDir, nil
}

// LoadReleaseFileWithValues loads a #ModuleRelease or #BundleRelease from a
// release .cue file and a separate values file.
//
// The loading follows a gate-first pattern:
//  1. Load release.cue alone (values stay abstract — no eager conflict evaluation).
//  2. Load values file separately.
//  3. Gate: validate values against #module.#config before filling.
//  4. Fill values into the release via FillPath.
//
// This ensures any schema violation is caught by the gate and surfaced as a
// *ConfigError with structured positions, rather than as a raw CUE build error.
//
// filePath may be a directory (release.cue is assumed) or a direct .cue path.
// valuesFile is the path to the values CUE file. If empty, values.cue in the
// same directory as the release file is used; an error is returned if it does
// not exist.
//
// registry behavior is identical to LoadReleaseFile.
func LoadReleaseFileWithValues(ctx *cue.Context, filePath, valuesFile, registry string) (cue.Value, string, error) {
	// Resolve values file path before calling LoadReleaseFile so we can error
	// early if it is missing.
	var err error
	filePath, err = resolveReleaseFile(filePath)
	if err != nil {
		return cue.Value{}, "", err
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("resolving release file path: %w", err)
	}
	parentDir := filepath.Dir(absPath)

	if valuesFile == "" {
		valuesFile = filepath.Join(parentDir, "values.cue")
	}
	if _, statErr := os.Stat(valuesFile); os.IsNotExist(statErr) {
		return cue.Value{}, "", fmt.Errorf("values file %q not found", valuesFile)
	} else if statErr != nil {
		return cue.Value{}, "", fmt.Errorf("accessing values file %q: %w", valuesFile, statErr)
	}

	// Step 1: Load release.cue alone. values: _ stays abstract, so BuildInstance
	// does not evaluate the module schema against the values yet.
	releaseVal, dir, err := LoadReleaseFile(ctx, filePath, registry)
	if err != nil {
		return cue.Value{}, "", err
	}

	// Step 2: Load values file separately (returns the "values" field if present).
	valuesVal, err := LoadValuesFile(ctx, valuesFile)
	if err != nil {
		return cue.Value{}, "", err
	}

	// Step 3: Gate — validate values against #module.#config.
	// Extract release name for error context; fall back to directory name.
	releaseName := filepath.Base(dir)
	if nameVal := releaseVal.LookupPath(cue.ParsePath("metadata.name")); nameVal.Exists() {
		if n, strErr := nameVal.String(); strErr == nil && n != "" {
			releaseName = n
		}
	}
	configSchema := releaseVal.LookupPath(cue.ParsePath("#module.#config"))
	if cfgErr := ValidateConfig(configSchema, valuesVal, "module", releaseName); cfgErr != nil {
		return cue.Value{}, "", cfgErr
	}

	// Step 4: Fill values into the release.
	releaseVal = releaseVal.FillPath(cue.ParsePath("values"), valuesVal)
	if err := releaseVal.Err(); err != nil {
		return cue.Value{}, "", fmt.Errorf("filling values into release: %w", err)
	}

	return releaseVal, dir, nil
}

// LoadModulePackage loads a module CUE package from a directory and returns
// the raw cue.Value. Used by the --module flag to inject a local module into
// a release file that does not import one from a registry.
func LoadModulePackage(ctx *cue.Context, dirPath string) (cue.Value, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return cue.Value{}, fmt.Errorf("resolving module directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return cue.Value{}, fmt.Errorf("accessing module directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return cue.Value{}, fmt.Errorf("module path %q is not a directory", absDir)
	}

	cfg := &load.Config{
		Dir: absDir,
	}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, fmt.Errorf("no CUE instances found in %s", absDir)
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading module package from %s: %w", absDir, instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building module package from %s: %w", absDir, err)
	}

	return val, nil
}

// LoadValuesFile loads a standalone CUE values file and returns the concrete
// values as a cue.Value. The function first tries to extract a "values" field
// from the loaded file (the standard OPM values file shape); if no such field
// exists the whole evaluated file value is returned instead.
//
// This is used by module-only vet validation when -f is provided but there is
// no release.cue in the module directory.
func LoadValuesFile(ctx *cue.Context, path string) (cue.Value, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return cue.Value{}, fmt.Errorf("resolving values file path: %w", err)
	}
	if _, statErr := os.Stat(absPath); statErr != nil {
		if os.IsNotExist(statErr) {
			return cue.Value{}, fmt.Errorf("values file %q not found", path)
		}
		return cue.Value{}, fmt.Errorf("accessing values file %q: %w", path, statErr)
	}

	parentDir := filepath.Dir(absPath)
	cfg := &load.Config{
		Dir: parentDir,
	}
	instances := load.Instances([]string{filepath.Base(absPath)}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, fmt.Errorf("no CUE instances found for %s", path)
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading values file: %w", instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building values file: %w", err)
	}

	// Standard OPM values files wrap values in a "values" field.
	// Return that field when it exists so the caller gets the raw config value.
	if valuesField := val.LookupPath(cue.ParsePath("values")); valuesField.Exists() && valuesField.Err() == nil {
		return valuesField, nil
	}

	return val, nil
}

// LoadReleasePackageWithValue loads a release CUE package (release.cue) and
// unifies it with a pre-loaded CUE value (e.g. debugValues). This is used by
// the debugValues path to avoid writing a temp file.
//
// The gate is applied before filling: values are validated against
// #module.#config so that schema violations are surfaced as *ConfigError rather
// than as raw CUE fill errors.
func LoadReleasePackageWithValue(ctx *cue.Context, releaseFile string, valuesVal cue.Value) (cue.Value, string, error) {
	releaseFile, err := resolveReleaseFile(releaseFile)
	if err != nil {
		return cue.Value{}, "", err
	}
	releaseDir := filepath.Dir(releaseFile)
	releaseBase := filepath.Base(releaseFile)

	cfg := &load.Config{
		Dir: releaseDir,
	}
	instances := load.Instances([]string{releaseBase}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, "", fmt.Errorf("no CUE instances found for %s", releaseBase)
	}
	if instances[0].Err != nil {
		return cue.Value{}, "", fmt.Errorf("loading release package: %w", instances[0].Err)
	}

	pkg := ctx.BuildInstance(instances[0])
	if err := pkg.Err(); err != nil {
		return cue.Value{}, "", fmt.Errorf("building release package: %w", err)
	}

	// Gate — validate values against #module.#config before filling.
	releaseName := filepath.Base(releaseDir)
	if nameVal := pkg.LookupPath(cue.ParsePath("metadata.name")); nameVal.Exists() {
		if n, strErr := nameVal.String(); strErr == nil && n != "" {
			releaseName = n
		}
	}
	configSchema := pkg.LookupPath(cue.ParsePath("#module.#config"))
	if cfgErr := ValidateConfig(configSchema, valuesVal, "module", releaseName); cfgErr != nil {
		return cue.Value{}, "", cfgErr
	}

	// Fill values into the release package.
	pkg = pkg.FillPath(cue.ParsePath("values"), valuesVal)
	if err := pkg.Err(); err != nil {
		return cue.Value{}, "", fmt.Errorf("filling values into release package: %w", err)
	}

	return pkg, releaseDir, nil
}
