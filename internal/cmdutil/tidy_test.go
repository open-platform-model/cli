package cmdutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/cuemod"
	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// tidied returns a consumer that a first tidy has already made tidy, with
// its module.cue backdated so a rewrite would show in the mtime.
func tidied(t *testing.T) (dir, registry string) {
	t.Helper()
	dir, registry = cuemodtest.NewConsumer(t)
	_, err := cuemod.Tidy(context.Background(), dir, cuemod.TidyOptions{Registry: registry})
	require.NoError(t, err)
	cuemodtest.Backdate(t, dir)
	return dir, registry
}

func untidy(t *testing.T) (dir, registry string) {
	t.Helper()
	dir, registry = cuemodtest.NewConsumer(t)
	cuemodtest.Backdate(t, dir)
	return dir, registry
}

// No test here may call t.Parallel: RunTidy changes the working directory
// and CUE_REGISTRY for the duration of the call.
func TestRunTidy(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kind  publish.Kind
		setup func(t *testing.T) (dir string, cfg *config.GlobalConfig)
		check bool

		wantCode    int // 0 means success
		wantLog     string
		wantMsg     string // substring of the DetailError message or the plain error
		wantHint    string // exact; "" asserts no hint
		wantWritten bool   // module.cue expected to change
		wantAbsPath bool   // the message names the absolute path checked
	}{
		{
			name: "module tidied",
			kind: publish.KindModule,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				dir, reg := untidy(t)
				return dir, &config.GlobalConfig{Registry: reg}
			},
			wantLog:     "Tidied module: updated cue.mod/module.cue",
			wantWritten: true,
		},
		{
			name: "catalog tidied",
			kind: publish.KindCatalog,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				dir, reg := untidy(t)
				return dir, &config.GlobalConfig{Registry: reg}
			},
			wantLog:     "Tidied catalog: updated cue.mod/module.cue",
			wantWritten: true,
		},
		{
			name: "already tidy writes nothing",
			kind: publish.KindModule,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				dir, reg := tidied(t)
				return dir, &config.GlobalConfig{Registry: reg}
			},
			wantLog: "Module already tidy; nothing written",
		},
		{
			name: "check passes on a tidy catalog",
			kind: publish.KindCatalog,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				dir, reg := tidied(t)
				return dir, &config.GlobalConfig{Registry: reg}
			},
			check:   true,
			wantLog: "Catalog already tidy; nothing written",
		},
		{
			name: "check fails on an untidy module",
			kind: publish.KindModule,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				dir, reg := untidy(t)
				return dir, &config.GlobalConfig{Registry: reg}
			},
			check:    true,
			wantCode: opmexit.ExitValidationError,
			wantMsg:  "module is not tidy: missing dependency providing package " + cuemodtest.DepModule,
			wantHint: "Run 'opm module tidy'",
		},
		{
			name: "check fails on an untidy catalog",
			kind: publish.KindCatalog,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				dir, reg := untidy(t)
				return dir, &config.GlobalConfig{Registry: reg}
			},
			check:    true,
			wantCode: opmexit.ExitValidationError,
			wantMsg:  "catalog is not tidy: missing dependency providing package " + cuemodtest.DepModule,
			wantHint: "Run 'opm catalog tidy'",
		},
		{
			name: "module dir without cue.mod points at init",
			kind: publish.KindModule,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				return t.TempDir(), &config.GlobalConfig{Registry: cuemodtest.UnreachableRegistry}
			},
			wantCode:    opmexit.ExitValidationError,
			wantMsg:     "is not a CUE module root (no cue.mod/module.cue)",
			wantAbsPath: true,
			wantHint:    "Run 'opm module init --dir %s' to create one", // %s: the path as given
		},
		{
			name: "catalog dir without cue.mod names only the path",
			kind: publish.KindCatalog,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				return t.TempDir(), &config.GlobalConfig{Registry: cuemodtest.UnreachableRegistry}
			},
			wantCode:    opmexit.ExitValidationError,
			wantMsg:     "is not a CUE module root (no cue.mod/module.cue)",
			wantAbsPath: true,
		},
		{
			name: "missing path gets no init hint",
			kind: publish.KindModule,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				return filepath.Join(t.TempDir(), "nope"), &config.GlobalConfig{}
			},
			wantCode:    opmexit.ExitValidationError,
			wantMsg:     "is not a CUE module root (no cue.mod/module.cue)",
			wantAbsPath: true,
		},
		{
			name: "unreachable registry is a resolution failure",
			kind: publish.KindModule,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				dir, _ := untidy(t)
				return dir, &config.GlobalConfig{Registry: cuemodtest.UnreachableRegistry}
			},
			wantCode: opmexit.ExitGeneralError,
			wantMsg:  "connection refused",
		},
		{
			name: "configured registry wins over ambient CUE_REGISTRY",
			kind: publish.KindModule,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				dir, reg := untidy(t)
				t.Setenv("CUE_REGISTRY", cuemodtest.UnreachableRegistry)
				return dir, &config.GlobalConfig{Registry: reg}
			},
			wantLog:     "Tidied module: updated cue.mod/module.cue",
			wantWritten: true,
		},
		{
			name: "no configured registry falls back to CUE_REGISTRY",
			kind: publish.KindModule,
			setup: func(t *testing.T) (string, *config.GlobalConfig) {
				dir, reg := untidy(t)
				t.Setenv("CUE_REGISTRY", reg)
				return dir, nil
			},
			wantLog:     "Tidied module: updated cue.mod/module.cue",
			wantWritten: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, cfg := tc.setup(t)
			before, _ := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))

			var logBuf bytes.Buffer
			output.SetupLogging(output.LogConfig{})
			output.SetLogWriter(&logBuf)

			err := RunTidy(context.Background(), cfg, tc.kind, []string{dir}, tc.check)

			after, _ := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
			assert.Equal(t, tc.wantWritten, !bytes.Equal(before, after), "module.cue written")

			if tc.wantCode == 0 {
				require.NoError(t, err)
				assert.Equal(t, 1, strings.Count(strings.TrimSpace(logBuf.String()), "\n")+1, "one outcome line: %q", logBuf.String())
				assert.Contains(t, logBuf.String(), tc.wantLog)
				return
			}

			var exitErr *opmexit.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, tc.wantCode, exitErr.Code)
			abs, absErr := filepath.Abs(dir)
			require.NoError(t, absErr)

			var detail *oerrors.DetailError
			if errors.As(err, &detail) {
				assert.Contains(t, detail.Message, tc.wantMsg)
				assert.Equal(t, expandHint(tc.wantHint, dir), detail.Hint)
				assert.NotContains(t, err.Error(), "cue mod tidy", "the cue tool's own suggestion must not leak")
				if tc.wantAbsPath {
					assert.Contains(t, detail.Message, abs)
				}
				return
			}
			assert.Empty(t, tc.wantHint, "a plain error carries no hint")
			assert.True(t, strings.HasPrefix(err.Error(), "tidying "+abs+": "), "resolution failures are prefixed with the path: %q", err.Error())
			assert.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}

// expandHint fills a hint template's %s with the path the test passed.
func expandHint(hint, dir string) string {
	if strings.Contains(hint, "%s") {
		return fmt.Sprintf(hint, dir)
	}
	return hint
}

// Without a path argument init already acts on the current directory, so
// the hint carries no --dir.
func TestRunTidy_NotModuleRootInCurrentDirectory(t *testing.T) {
	t.Chdir(t.TempDir())

	err := RunTidy(context.Background(), &config.GlobalConfig{Registry: cuemodtest.UnreachableRegistry}, publish.KindModule, nil, false)

	var detail *oerrors.DetailError
	require.ErrorAs(t, err, &detail)
	assert.Equal(t, "Run 'opm module init' to create one", detail.Hint)
}

func TestRunTidy_DefaultsToCurrentDirectory(t *testing.T) {
	dir, reg := untidy(t)
	t.Chdir(dir)

	require.NoError(t, RunTidy(context.Background(), &config.GlobalConfig{Registry: reg}, publish.KindModule, nil, false))
	assert.Contains(t, cuemodtest.ReadModule(t, dir), `"`+cuemodtest.DepModule+`"`)
}
