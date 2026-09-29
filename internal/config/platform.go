// Package config provides configuration loading and management.
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue/ast"

	liberrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/platform"

	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// platformModuleErrType is the DetailError type used for platform module
// build failures.
const platformModuleErrType = "platform module error"

// PlatformCacheDir returns the directory generated platform modules are
// cached under: cache/platforms beside the config file, so --config/OPM_CONFIG
// overrides move it together with the config. A cluster Platform CR or a
// render's own deps are generated into <PlatformCacheDir>/<content-hash>/
// before a render acquires the module. Derived state: safe to delete at any
// time.
func PlatformCacheDir(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "cache", "platforms")
}

// LegacyPlatformFilePath returns the path the pre-0019 data-only platform
// file lived at: platform.cue beside the config file. It exists for
// migration checks only (`opm config vet` fails on it, `opm config init`
// removes it); nothing reads the file.
func LegacyPlatformFilePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "platform.cue")
}

// LegacyPlatformDirPath returns the directory the retired local default
// platform module lived at: platform/ beside the config file. No command
// reads it; `opm config vet` warns when it is still on disk.
func LegacyPlatformDirPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "platform")
}

// BuildPlatformModule builds the platform module at dir through the kernel's
// shape-gated platform acquire: imports resolve (from registry, or the CUE
// module cache when warm), the value is a well-formed #Platform, and the
// schema's derived-entry tripwires (key-to-import binding, derived version)
// evaluate. registry overrides CUE_REGISTRY for the build; empty means the
// process environment.
//
// Failures surface as a DetailError naming the module directory, with the
// loader/CUE cause kept for errors.Is/As and a hint keyed on the cause: a
// missing directory or a directory that is not a CUE module names the
// expected shape, an unresolvable dependency the cue.mod pin, and a
// #registry conflict the entry's key/import pairing.
func BuildPlatformModule(ctx context.Context, dir, registry string) (*platform.Platform, error) {
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(PlatformModuleFileName))); err != nil {
		return nil, &oerrors.DetailError{
			Type:     platformModuleErrType,
			Message:  fmt.Sprintf("%s is not a platform module: %s not found", dir, PlatformModuleFileName),
			Location: dir,
			Hint:     "A platform module is a directory holding cue.mod/module.cue and a platform.cue package embedding core.#Platform; 'opm platform pull <dir>' captures a cluster's",
			Cause:    oerrors.ErrValidation,
		}
	}

	// The registry mapping is supplied once, at kernel construction; the
	// acquire verb takes no per-call override.
	k := NewKernel(registry)
	p, err := k.AcquirePlatformFromDir(ctx, dir)
	if err != nil {
		return nil, platformBuildError(dir, err)
	}
	return p, nil
}

// platformBuildError wraps a loader/CUE failure for dir in a DetailError
// with a cause-specific hint.
func platformBuildError(dir string, err error) error {
	return &oerrors.DetailError{
		Type:     platformModuleErrType,
		Message:  err.Error(),
		Location: dir,
		Hint:     platformBuildHint(dir, err),
		Cause:    fmt.Errorf("%w: %w", oerrors.ErrValidation, err),
	}
}

// platformBuildHint picks the remediation for a platform module build
// failure from the shape of the underlying error. The hints name no command
// to re-run: the caller decides what the user runs next.
func platformBuildHint(dir string, err error) string {
	modFile := filepath.Join(dir, filepath.FromSlash(PlatformModuleFileName))
	msg := err.Error()
	switch {
	case errors.Is(err, liberrors.ErrWrongKind), errors.Is(err, liberrors.ErrInvalidPackage):
		return "platform.cue must be a single package embedding core.#Platform"
	case strings.Contains(msg, "module not found"), strings.Contains(msg, "cannot find package"), strings.Contains(msg, "cannot expand module graph"):
		return "Pin a published build in " + modFile + ", then try again"
	case strings.Contains(msg, "#registry"):
		return "Each #registry entry's key must equal the module path of the catalog it imports (#catalog); fix the entry named above in " + filepath.Join(dir, PlatformCUEFileName)
	default:
		return "Fix the platform module at " + dir + " (pins in " + modFile + "), then try again"
	}
}

// fileHasImports reports whether the parsed CUE file contains any import
// declaration. The config file is data-only by contract (0006:D39).
func fileHasImports(f *ast.File) bool {
	for _, decl := range f.Decls {
		if _, ok := decl.(*ast.ImportDecl); ok {
			return true
		}
	}
	return false
}
