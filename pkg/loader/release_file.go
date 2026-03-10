package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
)

// LoadOptions configures release-file loading behavior.
type LoadOptions struct {
	// Registry overrides the CUE registry used while loading a release file.
	// Empty means use the current process environment.
	Registry string
}

// LoadReleaseFile loads a #ModuleRelease or #BundleRelease from a standalone
// .cue file. CUE imports (including registry module references) are resolved
// via load.Instances() using the file's parent directory for cue.mod resolution.
//
// The returned cue.Value may have #module unfilled if the release file does not
// import a module. The caller is responsible for filling #module via FillPath
// when --module is provided.
//
// Returns the evaluated CUE value and the directory used for CUE resolution.
func LoadReleaseFile(ctx *cue.Context, filePath string, opts LoadOptions) (cue.Value, string, error) {
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

	// Set CUE_REGISTRY env var if Registry is non-empty, restoring original after.
	if opts.Registry != "" {
		orig, hadOrig := os.LookupEnv("CUE_REGISTRY")
		if err := os.Setenv("CUE_REGISTRY", opts.Registry); err != nil {
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
