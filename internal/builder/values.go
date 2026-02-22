package builder

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/core/module"
	opmerrors "github.com/opmodel/cli/internal/errors"
)

// selectValues determines the CUE values to use for this build.
//
// If valuesFiles are provided, each file is loaded and unified in order;
// later files take precedence on conflict. The top-level "values" field is
// extracted from the unified result.
//
// If no valuesFiles are given, mod.Values (from values.cue loaded at module
// load time) is returned as-is.
//
// Returns *opmerrors.ValidationError if no values are available.
func selectValues(ctx *cue.Context, mod *module.Module, valuesFiles []string) (cue.Value, error) {
	if len(valuesFiles) > 0 {
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

	if !mod.Values.Exists() {
		return cue.Value{}, &opmerrors.ValidationError{
			Message: "module missing 'values' field — provide values via values.cue or --values flag",
		}
	}
	return mod.Values, nil
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
