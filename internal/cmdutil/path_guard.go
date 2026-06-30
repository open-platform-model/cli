package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"
)

func ValidateModuleInputPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving module path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking module path %q: %w", absPath, err)
	}

	// Was: detected release.cue (enhancement 0002 D9; instance-file convention).
	if info.IsDir() {
		if hasFile(absPath, "instance.cue") {
			return fmt.Errorf("path %q is an instance package, not a module - use 'opm instance'", absPath)
		}
		return nil
	}

	if filepath.Base(absPath) == "instance.cue" {
		return fmt.Errorf("path %q is an instance file, not a module - use 'opm instance'", absPath)
	}

	return nil
}

func ValidateInstanceInputPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving instance path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking instance path %q: %w", absPath, err)
	}

	// Was: detected release.cue (enhancement 0002 D9; instance-file convention).
	if info.IsDir() {
		if hasFile(absPath, "instance.cue") {
			return nil
		}
		if hasFile(absPath, "module.cue") {
			return fmt.Errorf("path %q is a module package, not an instance - use 'opm module' or point to an instance.cue file", absPath)
		}
		return nil
	}

	if filepath.Base(absPath) == "module.cue" {
		return fmt.Errorf("path %q is a module file, not an instance - use 'opm module' or point to an instance.cue file", absPath)
	}

	return nil
}

func hasFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
