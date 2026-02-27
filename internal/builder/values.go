package builder

import (
	"os"
	"path/filepath"
	"strings"

	opmerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
)

// resolveValuesFiles returns the list of values file paths to use for this build.
//
// If valuesFiles is non-empty (from --values flags), it is returned as-is.
// Otherwise, values.cue in the module directory is used as a conventional
// fallback. values.cue is treated identically to an explicit --values file —
// there is no semantic distinction. Actual defaults belong in #config.
//
// Returns *opmerrors.ValidationError if no values source is available.
func resolveValuesFiles(modulePath string, valuesFiles []string) ([]string, error) {
	if len(valuesFiles) > 0 {
		names := make([]string, len(valuesFiles))
		for i, vf := range valuesFiles {
			names[i] = filepath.Base(vf)
		}
		output.Debug("using values files (--values)", "files", strings.Join(names, ", "))
		return valuesFiles, nil
	}

	fallback := filepath.Join(modulePath, "values.cue")
	if _, err := os.Stat(fallback); os.IsNotExist(err) { //nolint:gosec // path is modulePath + "values.cue"; modulePath validated by loader.ResolvePath
		return nil, &opmerrors.ValidationError{
			Message: "no values file found — pass --values <file> or create values.cue in the module directory",
		}
	}

	output.Debug("using values.cue fallback", "path", fallback)
	return []string{fallback}, nil
}
