package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
)

// LoadOptions configures instance-file loading behavior.
type LoadOptions struct {
	// Registry overrides the CUE registry used while loading an instance file.
	// Empty means use the current process environment.
	Registry string
}

// LoadInstanceFile loads a #ModuleInstance from a standalone .cue file. CUE
// imports (including registry module references) are resolved via
// load.Instances() using the file's parent directory for cue.mod resolution.
//
// The returned cue.Value may have #module unfilled if the instance file does not
// import a module. The instance file must import a module to fill #module.
//
// Returns the evaluated CUE value and the directory used for CUE resolution.
func LoadInstanceFile(ctx *cue.Context, filePath string, opts LoadOptions) (cue.Value, string, error) {
	var err error
	filePath, err = resolveInstanceFile(filePath)
	if err != nil {
		return cue.Value{}, "", err
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("resolving instance file path: %w", err)
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
		return cue.Value{}, "", fmt.Errorf("loading instance file: %w", instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, "", fmt.Errorf("building instance file: %w", err)
	}

	return val, parentDir, nil
}

// resolveInstanceFile resolves either an instance directory or direct file path.
// Inside a directory the loader requires instance.cue (was release.cue, 0002 D9);
// instance.cue is not accepted as a fallback (D8 hard-rename, no alias).
//
// Was: resolveReleaseFile (enhancement 0002 D8 hard-rename).
func resolveInstanceFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("instance path must not be empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("instance path %q not found", path)
		}
		return "", fmt.Errorf("stat instance path: %w", err)
	}
	if info.IsDir() {
		instancePath := filepath.Join(path, "instance.cue")
		if _, err := os.Stat(instancePath); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("instance path %q does not contain instance.cue", path)
			}
			return "", fmt.Errorf("stat instance file: %w", err)
		}
		return instancePath, nil
	}
	return path, nil
}
