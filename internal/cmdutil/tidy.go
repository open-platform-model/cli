package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/cuemod"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// RunTidy is the shared body of `opm module tidy` and `opm catalog tidy`:
// tidy the artifact's cue.mod exactly as `cue mod tidy` would, resolving
// through the CLI's registry (cfg.Registry). Tidy is a CUE module operation,
// so the kind only changes wording; OPM identity is never read. Exit codes:
// 0 tidied, already tidy or check passed; 1 resolution or registry failure;
// 2 not a module root, or not tidy under check.
func RunTidy(ctx context.Context, cfg *config.GlobalConfig, kind publish.Kind, pathArgs []string, check bool) error {
	dir := ResolveModulePath(pathArgs)

	var registry string
	if cfg != nil {
		registry = cfg.Registry
	}
	res, err := cuemod.Tidy(ctx, dir, cuemod.TidyOptions{Registry: registry, Check: check})
	if err != nil {
		return tidyError(kind, dir, err)
	}

	updated := updatedFiles(res)
	if len(updated) == 0 {
		output.Info(fmt.Sprintf("%s already tidy; nothing written", capitalize(string(kind))))
		return nil
	}
	output.Info(output.FormatCheckmark(fmt.Sprintf("Tidied %s: updated %s", kind, strings.Join(updated, " and "))))
	return nil
}

func updatedFiles(res cuemod.TidyResult) []string {
	var files []string
	if res.ModuleUpdated {
		files = append(files, "cue.mod/module.cue")
	}
	if res.LocalUpdated {
		files = append(files, "cue.mod/local-module.cue")
	}
	return files
}

// tidyError maps cuemod failures to exit codes. A missing module root and a
// failed check are refusals (2) whose hint names the command that fixes
// them; anything else is a resolution failure (1), prefixed with the
// absolute path so the resolver's text says which tree it was resolving.
func tidyError(kind publish.Kind, dir string, err error) error {
	var notTidy *cuemod.NotTidyError
	switch {
	case errors.Is(err, cuemod.ErrNotModuleRoot):
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err: &oerrors.DetailError{
				Type:    "not a module root",
				Message: err.Error(),
				Hint:    notModuleRootHint(kind, dir),
				Cause:   err,
			},
		}
	case errors.As(err, &notTidy):
		msg := fmt.Sprintf("%s is not tidy", kind)
		if notTidy.Reason != "" {
			msg += ": " + notTidy.Reason
		}
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err: &oerrors.DetailError{
				Type:    "not tidy",
				Message: msg,
				Hint:    fmt.Sprintf("Run 'opm %s tidy'", kind),
				Cause:   err,
			},
		}
	}
	return &opmexit.ExitError{
		Code: opmexit.ExitGeneralError,
		Err:  fmt.Errorf("tidying %s: %w", absOrSelf(dir), err),
	}
}

// notModuleRootHint points an existing module directory without cue.mod at
// `opm module init`. There is no catalog init, and a path that is not a
// directory at all needs correcting rather than initializing.
func notModuleRootHint(kind publish.Kind, dir string) string {
	if kind != publish.KindModule {
		return ""
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return ""
	}
	return "Run 'opm module init' to create one"
}

func absOrSelf(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
