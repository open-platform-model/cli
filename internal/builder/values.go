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
// If valuesFiles are provided via --values, they are the sole values source.
// values.cue in the module directory is completely ignored.
//
// If no --values are given, values.cue is looked up in mod.ModulePath as a
// conventional fallback. It is treated identically to an explicit --values
// argument — there is no semantic distinction between the two. values.cue is
// not "default values"; it is simply a regular values file with a conventional
// name. Actual defaults belong in #config inside module.cue.
//
// In both cases every file must define a top-level "values:" field, and the
// results are CUE-unified when multiple files are provided. Unification is
// commutative — ordering has no semantic effect.
//
// Returns *opmerrors.ValidationError if no values source is available, or if
// a values file does not contain a top-level "values" field.
func selectValues(ctx *cue.Context, mod *module.Module, valuesFiles []string) (cue.Value, error) {
	// If no explicit --values were provided, fall back to values.cue in the
	// module directory (treated as a regular values file).
	if len(valuesFiles) == 0 {
		fallback := filepath.Join(mod.ModulePath, "values.cue")
		if _, err := os.Stat(fallback); os.IsNotExist(err) { //nolint:gosec // path is mod.ModulePath + "values.cue"; ModulePath is validated by loader.ResolvePath
			return cue.Value{}, &opmerrors.ValidationError{
				Message: "no values file found — pass --values <file> or create values.cue in the module directory",
			}
		}
		output.Debug("using values.cue fallback", "path", fallback)
		valuesFiles = []string{fallback}
	} else {
		names := make([]string, len(valuesFiles))
		for i, vf := range valuesFiles {
			names[i] = filepath.Base(vf)
		}
		output.Debug("using values files (--values)", "files", strings.Join(names, ", "))
	}

	// Load and unify all values files through the same code path.
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
			Message: "no 'values' field found in values file(s) — each file must define a top-level 'values:' field",
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

	content, err := os.ReadFile(absPath) //nolint:gosec // absPath is resolved from caller-provided paths validated upstream
	if err != nil {
		return cue.Value{}, fmt.Errorf("reading file: %w", err)
	}

	value := ctx.CompileBytes(content, cue.Filename(absPath))
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("compiling values file %s: %w", absPath, value.Err())
	}

	return value, nil
}
