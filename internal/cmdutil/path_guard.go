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
		return nil
	}

	if info.IsDir() {
		if hasFile(absPath, "release.cue") {
			return fmt.Errorf("path %q is a release package, not a module - use 'opm release'", absPath)
		}
		return nil
	}

	if filepath.Base(absPath) == "release.cue" {
		return fmt.Errorf("path %q is a release file, not a module - use 'opm release'", absPath)
	}

	return nil
}

func ValidateReleaseInputPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving release path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil
	}

	if info.IsDir() {
		if hasFile(absPath, "release.cue") {
			return nil
		}
		if hasFile(absPath, "module.cue") {
			return fmt.Errorf("path %q is a module package, not a release - use 'opm module' or point to a release.cue file", absPath)
		}
		return nil
	}

	if filepath.Base(absPath) == "module.cue" {
		return fmt.Errorf("path %q is a module file, not a release - use 'opm module' or point to a release.cue file", absPath)
	}

	return nil
}

func hasFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
