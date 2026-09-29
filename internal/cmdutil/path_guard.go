package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	liberrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/kernel"
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

// ModulePackageError reports, for a failed instance acquire of dir, whether
// the package there is a module rather than an instance, and if so the
// refusal pointing at moduleCmd, the module command that does what the
// instance command was asked to ("opm module build" for instance build).
// With no counterpart (moduleCmd empty: diff, the cluster queries) the
// refusal points at the module command group. An instance command decides
// by what the package is, not by its file names: only a shape-gate kind
// mismatch is examined, and the kernel's module acquire is the judge. Nil
// when err is anything else or the package is not a module either.
func ModulePackageError(ctx context.Context, k *kernel.Kernel, dir, moduleCmd string, err error) error {
	if !errors.Is(err, liberrors.ErrWrongKind) {
		return nil
	}
	if _, modErr := k.AcquireModuleFromDir(ctx, dir); modErr != nil {
		return nil //nolint:nilerr // not a module either: the caller reports the instance acquire's own error
	}
	abs, absErr := filepath.Abs(dir)
	if absErr != nil {
		abs = dir
	}
	if moduleCmd == "" {
		return fmt.Errorf("%s is a module, not an instance; the 'opm module' commands take a module directory", abs)
	}
	return fmt.Errorf("%s is a module, not an instance; run: %s %s", abs, moduleCmd, abs)
}

func hasFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// InstanceDir returns the CUE package directory for an instance path: the
// path itself when it is a directory, else its parent. A path that does not
// exist resolves to its parent, so the kernel's acquire is what reports a
// missing package. Shared by the render path and the instance argument
// resolution of the cluster-query commands.
func InstanceDir(path string) (string, error) {
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
