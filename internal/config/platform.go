// Package config provides configuration loading and management.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"

	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// platformFileErrType is the DetailError type used for platform file
// parse/validation failures.
const platformFileErrType = "platform file error"

// PlatformFilePath returns the platform file path that is sibling to the
// given (resolved) config file path, so --config/OPM_CONFIG overrides move
// both files together.
func PlatformFilePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "platform.cue")
}

// ValidatePlatformFile parses and validates the platform file at path
// against the embedded #PlatformFile projection schema. The file MUST be
// data-only: any CUE import declaration is rejected (enhancement 0006 D39).
//
// The caller decides how a missing file is handled; ValidatePlatformFile
// returns the os.Stat error untouched when the file does not exist.
func ValidatePlatformFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Reject imports before evaluation: the local platform file is data
	// only. Parsing is cheap and gives a precise, early error.
	astFile, err := parser.ParseFile(path, content)
	if err != nil {
		return &oerrors.DetailError{
			Type:     platformFileErrType,
			Message:  err.Error(),
			Location: path,
			Hint:     "Fix the CUE syntax; see the template written by 'opm config init'",
			Cause:    oerrors.ErrValidation,
		}
	}
	if fileHasImports(astFile) {
		return &oerrors.DetailError{
			Type:     platformFileErrType,
			Message:  "the local platform file must be data-only — CUE imports are not allowed",
			Location: path,
			Hint:     "Remove the import declarations; declare catalog subscriptions as plain data (see 'opm config init')",
			Cause:    oerrors.ErrValidation,
		}
	}

	ctx := cuecontext.New()
	value := ctx.CompileBytes(content, cue.Filename(path))
	if value.Err() != nil {
		return &oerrors.DetailError{
			Type:     platformFileErrType,
			Message:  value.Err().Error(),
			Location: path,
			Hint:     "Fix the CUE errors; see the template written by 'opm config init'",
			Cause:    oerrors.ErrValidation,
		}
	}

	schema := ctx.CompileBytes(platformSchemaCUE, cue.Filename("schema/platform.cue"))
	if schema.Err() != nil {
		return fmt.Errorf("compiling embedded platform schema: %w", schema.Err())
	}
	def := schema.LookupPath(cue.ParsePath("#PlatformFile"))
	if !def.Exists() {
		return fmt.Errorf("embedded schema missing #PlatformFile definition")
	}

	unified := def.Unify(value)
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		return &oerrors.DetailError{
			Type:     "platform schema validation failed",
			Message:  err.Error(),
			Location: path,
			Hint:     "Check the platform file against the expected shape (name, type, registry subscriptions)",
			Cause:    oerrors.ErrValidation,
		}
	}

	return nil
}

// fileHasImports reports whether the parsed CUE file contains any import
// declaration. Shared by the config and platform file loaders: both ~/.opm
// files are data-only by contract (enhancement 0006 D39).
func fileHasImports(f *ast.File) bool {
	for _, decl := range f.Decls {
		if _, ok := decl.(*ast.ImportDecl); ok {
			return true
		}
	}
	return false
}
