// Package config provides configuration loading and management.
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/platform"

	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// platformModuleErrType is the DetailError type used for platform module
// build failures.
const platformModuleErrType = "platform module error"

// platformFileErrType is the DetailError type used for legacy platform file
// parse/validation failures.
const platformFileErrType = "platform file error"

// PlatformDir returns the platform module directory that is sibling to the
// given (resolved) config file path, so --config/OPM_CONFIG overrides move
// both together (enhancement 0019 D5: the local default platform is a CUE
// module, not a data file).
func PlatformDir(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), PlatformDirName)
}

// LegacyPlatformFilePath returns the path the pre-0019 data-only platform
// file lived at: platform.cue beside the config file. It exists for
// migration checks (`opm config vet` fails on it, `opm config init` removes
// it) and for the render path until cli-render-switch moves resolution onto
// the module directory.
func LegacyPlatformFilePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "platform.cue")
}

// WritePlatformModule writes the seeded local default platform module into
// dir: cue.mod/module.cue (DefaultPlatformModuleFile) and platform.cue
// (DefaultPlatformCUE). Directories are created 0700 and files written 0600,
// matching the config file. Existing files are overwritten; nothing else in
// dir is touched. It is normatively offline: nothing is resolved.
func WritePlatformModule(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(PlatformModuleFileName)), 0o700); err != nil {
		return fmt.Errorf("creating platform module directory: %w", err)
	}
	files := map[string]string{
		PlatformModuleFileName: DefaultPlatformModuleFile,
		PlatformCUEFileName:    DefaultPlatformCUE,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(content), 0o600); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
	}
	return nil
}

// BuildPlatformModule builds the platform module at dir through the kernel's
// shape-gated platform loader: imports resolve (from registry, or the CUE
// module cache when warm), the value is a well-formed #Platform, and the
// schema's derived-entry tripwires (key-to-import binding, derived version)
// evaluate. registry overrides CUE_REGISTRY for the build; empty means the
// process environment.
//
// Failures surface as a DetailError naming the module directory, with the
// loader/CUE cause kept for errors.Is/As and a hint keyed on the cause: a
// missing directory or a directory that is not a CUE module points at
// `opm config init`, an unresolvable dependency at the cue.mod pin, and a
// #registry conflict at the entry's key/import pairing.
func BuildPlatformModule(ctx context.Context, dir, registry string) (*platform.Platform, error) {
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(PlatformModuleFileName))); err != nil {
		return nil, &oerrors.DetailError{
			Type:     platformModuleErrType,
			Message:  fmt.Sprintf("%s is not a platform module: %s not found", dir, PlatformModuleFileName),
			Location: dir,
			Hint:     "A platform module is a directory holding cue.mod/module.cue and a platform.cue package embedding core.#Platform. Run 'opm config init' (or --force) to seed the local default",
			Cause:    oerrors.ErrValidation,
		}
	}

	k := kernel.New(kernel.WithRegistry(registry))
	val, err := k.LoadPlatformPackage(ctx, dir, loaderfile.LoadOptions{Registry: registry})
	if err != nil {
		return nil, platformBuildError(dir, err)
	}
	p, err := k.NewPlatformFromValue(val)
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
// failure from the shape of the underlying error.
func platformBuildHint(dir string, err error) string {
	modFile := filepath.Join(dir, filepath.FromSlash(PlatformModuleFileName))
	msg := err.Error()
	switch {
	case errors.Is(err, loaderfile.ErrWrongKind), errors.Is(err, loaderfile.ErrInvalidPackage):
		return "platform.cue must be a single package embedding core.#Platform; re-run 'opm config init --force' for a fresh module"
	case strings.Contains(msg, "module not found"), strings.Contains(msg, "cannot find package"), strings.Contains(msg, "cannot expand module graph"):
		return "Pin a published build in " + modFile + ", then re-run 'opm config vet'"
	case strings.Contains(msg, "#registry"):
		return "Each #registry entry's key must equal the module path of the catalog it imports (#catalog); fix the entry named above in " + filepath.Join(dir, PlatformCUEFileName)
	default:
		return "Fix the platform module at " + dir + " (pins in " + modFile + "), then re-run 'opm config vet'"
	}
}

// LoadLegacyPlatformFile parses and validates a pre-0019 data-only platform
// file at path against the embedded #PlatformFile projection schema and
// returns the compiled value. The file MUST be data-only: any CUE import
// declaration is rejected (enhancement 0006 D39).
//
// Retained only for the render path's --platform <file> and local-default
// sources until cli-render-switch moves resolution onto platform module
// directories; that change deletes it together with schema/platform.cue.
// `opm config init` no longer writes this shape and `opm config vet`
// refuses it.
//
// The caller decides how a missing file is handled; the os.Stat error is
// returned untouched when the file does not exist.
func LoadLegacyPlatformFile(path string) (cue.Value, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return cue.Value{}, err
	}

	// Reject imports before evaluation: the legacy platform file is data
	// only. Parsing is cheap and gives a precise, early error.
	astFile, err := parser.ParseFile(path, content)
	if err != nil {
		return cue.Value{}, &oerrors.DetailError{
			Type:     platformFileErrType,
			Message:  err.Error(),
			Location: path,
			Hint:     "Fix the CUE syntax, or migrate to the platform module with 'opm config init --force'",
			Cause:    oerrors.ErrValidation,
		}
	}
	if fileHasImports(astFile) {
		return cue.Value{}, &oerrors.DetailError{
			Type:     platformFileErrType,
			Message:  "the legacy platform file must be data-only — CUE imports are not allowed",
			Location: path,
			Hint:     "A platform with imports is a module: point --platform at a platform module directory, or migrate with 'opm config init --force'",
			Cause:    oerrors.ErrValidation,
		}
	}

	ctx := cuecontext.New()
	value := ctx.CompileBytes(content, cue.Filename(path))
	if value.Err() != nil {
		return cue.Value{}, &oerrors.DetailError{
			Type:     platformFileErrType,
			Message:  value.Err().Error(),
			Location: path,
			Hint:     "Fix the CUE errors, or migrate to the platform module with 'opm config init --force'",
			Cause:    oerrors.ErrValidation,
		}
	}

	schema := ctx.CompileBytes(platformSchemaCUE, cue.Filename("schema/platform.cue"))
	if schema.Err() != nil {
		return cue.Value{}, fmt.Errorf("compiling embedded platform schema: %w", schema.Err())
	}
	def := schema.LookupPath(cue.ParsePath("#PlatformFile"))
	if !def.Exists() {
		return cue.Value{}, fmt.Errorf("embedded schema missing #PlatformFile definition")
	}

	unified := def.Unify(value)
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		return cue.Value{}, &oerrors.DetailError{
			Type:     "platform schema validation failed",
			Message:  err.Error(),
			Location: path,
			Hint:     "Check the platform file against the expected shape (name, type, registry subscriptions)",
			Cause:    oerrors.ErrValidation,
		}
	}

	return value, nil
}

// fileHasImports reports whether the parsed CUE file contains any import
// declaration. Shared by the config loader and the legacy platform file
// loader: both are data-only by contract (enhancement 0006 D39).
func fileHasImports(f *ast.File) bool {
	for _, decl := range f.Decls {
		if _, ok := decl.(*ast.ImportDecl); ok {
			return true
		}
	}
	return false
}
