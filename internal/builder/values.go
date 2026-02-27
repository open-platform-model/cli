package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/core/module"
	opmerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
)

// selectValues determines the CUE values to use for this build.
//
// If valuesFiles are provided, they are the sole values source. Each file is
// loaded and CUE-unified (unification is commutative — no ordering). The
// top-level "values" field is extracted from the unified result. values.cue
// in the module directory is completely ignored.
//
// If no valuesFiles are given, values.cue is discovered from mod.ModulePath.
// If found, it is loaded via ctx.CompileBytes and the "values" field is extracted.
// If not found, a *opmerrors.ValidationError is returned — building without any
// values source is not supported.
//
// Returns *opmerrors.ValidationError if no values source is available, or if
// the values source does not contain a top-level "values" field.
func selectValues(ctx *cue.Context, mod *module.Module, valuesFiles []string) (cue.Value, error) {
	if len(valuesFiles) > 0 {
		names := make([]string, len(valuesFiles))
		for i, vf := range valuesFiles {
			names[i] = filepath.Base(vf)
		}
		output.Debug("using values files (--values)", "files", strings.Join(names, ", "))

		var unified cue.Value
		for i, path := range valuesFiles {
			v, err := loadValuesFile(ctx, path)
			if err != nil {
				return cue.Value{}, fmt.Errorf("loading values file %s: %w", path, err)
			}
			if i == 0 {
				unified = v
			} else {
				unified = unified.Unify(v)
			}
		}
		if err := unified.Err(); err != nil {
			return cue.Value{}, fmt.Errorf("unifying values files: %w", err)
		}

		values := unified.LookupPath(cue.ParsePath("values"))
		if !values.Exists() {
			return cue.Value{}, &opmerrors.ValidationError{
				Message: "no 'values' field found in provided values files — files must define a top-level 'values:' field",
			}
		}
		return values, nil
	}

	// No --values provided: discover values.cue from the module directory.
	valuesPath := filepath.Join(mod.ModulePath, "values.cue")
	if _, err := os.Stat(valuesPath); os.IsNotExist(err) { //nolint:gosec // path is mod.ModulePath + "values.cue"; ModulePath is validated by loader.ResolvePath
		return cue.Value{}, &opmerrors.ValidationError{
			Message: "no values source available — add values.cue to the module directory or pass --values flag",
		}
	}

	output.Debug("using default values.cue", "path", valuesPath)

	content, err := os.ReadFile(valuesPath) //nolint:gosec // path is mod.ModulePath + "values.cue"; ModulePath is validated by loader.ResolvePath
	if err != nil {
		return cue.Value{}, fmt.Errorf("reading values.cue: %w", err)
	}

	compiled := ctx.CompileBytes(content, cue.Filename(valuesPath))
	if err := compiled.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("compiling values.cue: %w", err)
	}

	values := compiled.LookupPath(cue.ParsePath("values"))
	if !values.Exists() {
		return cue.Value{}, &opmerrors.ValidationError{
			Message: "no 'values' field found in values.cue — values.cue must define a top-level 'values:' field",
		}
	}

	return values, nil
}

// loadValuesFile loads and compiles a single CUE values file.
func loadValuesFile(ctx *cue.Context, path string) (cue.Value, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return cue.Value{}, fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return cue.Value{}, fmt.Errorf("file not found: %s", absPath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return cue.Value{}, fmt.Errorf("reading file: %w", err)
	}

	value := ctx.CompileBytes(content, cue.Filename(absPath))
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("compiling values file %s: %w", absPath, value.Err())
	}

	return value, nil
}
