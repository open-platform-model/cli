// Package cuemod runs CUE module maintenance (today: `cue mod tidy`) in
// process, without a cue executable.
//
// CUE's tidy engine lives under cuelang.org/go/internal and cannot be
// imported, so this package drives the public cuelang.org/go/cmd/cue/cmd
// command tree instead. That tree reads the module root from the process
// working directory and the registry from the process environment, so Tidy
// is the one place in the CLI that calls os.Chdir and os.Setenv. Both are
// restored before Tidy returns, under a package mutex; callers must not run
// Tidy concurrently with code that depends on either.
package cuemod

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	cuecmd "cuelang.org/go/cmd/cue/cmd"
)

// TidyOptions configures one tidy.
type TidyOptions struct {
	// Registry is a CUE_REGISTRY-syntax mapping. Empty inherits the
	// process environment and CUE's default.
	Registry string
	// Check resolves without writing and fails with *NotTidyError when
	// tidying would change a file.
	Check bool
}

// TidyResult says which files a tidy changed. Both false means already tidy.
type TidyResult struct {
	// ModuleUpdated reports that cue.mod/module.cue was rewritten.
	ModuleUpdated bool
	// LocalUpdated reports that cue.mod/local-module.cue was rewritten,
	// created or removed.
	LocalUpdated bool
}

// ErrNotModuleRoot is returned, wrapped with the absolute path checked,
// when the target directory does not exist, is not a directory, or has no
// cue.mod/module.cue. Tidy returns it before any registry access.
var ErrNotModuleRoot = errors.New("not a CUE module root")

// NotTidyError reports a Check failure. Reason is CUE's explanation (for
// example the missing dependency and the package that needs it); it may be
// empty.
type NotTidyError struct {
	Reason string
}

func (e *NotTidyError) Error() string {
	if e.Reason == "" {
		return "module is not tidy"
	}
	return "module is not tidy: " + e.Reason
}

// notTidyPrefix is how cmd/cue flattens modload.ErrModuleNotTidy (an
// internal type errors.As cannot reach): "module is not tidy, use 'cue mod
// tidy'[: <reason>]". A test pins it against the embedded CUE version.
const (
	notTidyPrefix    = "module is not tidy"
	notTidySuggested = "module is not tidy, use 'cue mod tidy'"
)

const (
	moduleFile = "cue.mod/module.cue"
	localFile  = "cue.mod/local-module.cue"
)

// processMu serializes the working-directory and environment mutation
// Tidy performs.
var processMu sync.Mutex

// Tidy tidies the CUE module rooted at dir exactly as `cue mod tidy` of the
// embedded CUE version would: missing dependencies are added at their
// latest version, unused ones removed, minimum version selection applied.
// With opts.Check it writes nothing and returns *NotTidyError when a file
// would change. dir must itself be the module root; parents are not
// searched.
func Tidy(ctx context.Context, dir string, opts TidyOptions) (TidyResult, error) {
	root, err := moduleRoot(dir)
	if err != nil {
		return TidyResult{}, err
	}

	before, err := snapshot(root)
	if err != nil {
		return TidyResult{}, err
	}

	if err := classify(runModTidy(ctx, root, opts)); err != nil {
		return TidyResult{}, err
	}

	after, err := snapshot(root)
	if err != nil {
		return TidyResult{}, err
	}
	return TidyResult{
		ModuleUpdated: !after.module.equal(before.module),
		LocalUpdated:  !after.local.equal(before.local),
	}, nil
}

// moduleRoot resolves dir to an absolute path and requires it to hold
// cue.mod/module.cue.
func moduleRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", dir, err)
	}
	info, err := os.Stat(filepath.Join(abs, filepath.FromSlash(moduleFile)))
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("%s is %w (no %s)", abs, ErrNotModuleRoot, moduleFile)
	}
	return abs, nil
}

// runModTidy executes `cue mod tidy [--check]` in root with the registry
// from opts and returns whatever cmd/cue wrote along with its raw error. It
// holds processMu for the whole call and restores the working directory and
// CUE_REGISTRY on every return path, including a panic.
func runModTidy(ctx context.Context, root string, opts TidyOptions) (output string, err error) {
	processMu.Lock()
	defer processMu.Unlock()

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("reading working directory: %w", err)
	}
	// cmd/cue's own -C flag calls os.Exit on a bad directory, which would
	// skip these restores; change directory here instead.
	if err := os.Chdir(root); err != nil {
		return "", fmt.Errorf("entering %s: %w", root, err)
	}
	defer func() {
		if cdErr := os.Chdir(wd); cdErr != nil && err == nil {
			err = fmt.Errorf("restoring working directory %s: %w", wd, cdErr)
		}
	}()

	if opts.Registry != "" {
		restore, setErr := setEnv("CUE_REGISTRY", opts.Registry)
		if setErr != nil {
			return "", setErr
		}
		defer func() {
			if envErr := restore(); envErr != nil && err == nil {
				err = envErr
			}
		}()
	}

	args := []string{"mod", "tidy"}
	if opts.Check {
		args = append(args, "--check")
	}
	c, err := cuecmd.New(args)
	if err != nil {
		return "", fmt.Errorf("building cue command: %w", err)
	}
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	err = c.Run(ctx)
	return out.String(), err
}

// setEnv sets key to value and returns a function that puts back the
// previous value, or unsets key when it was not set.
func setEnv(key, value string) (func() error, error) {
	prev, had := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		return nil, fmt.Errorf("setting %s: %w", key, err)
	}
	return func() error {
		var err error
		if had {
			err = os.Setenv(key, prev)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			return fmt.Errorf("restoring %s: %w", key, err)
		}
		return nil
	}, nil
}

// classify turns cmd/cue's result into Tidy's error contract. When cmd/cue
// printed its errors instead of returning them, output carries the text.
func classify(output string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if errors.Is(err, cuecmd.ErrPrintedError) {
		if text := strings.TrimSpace(output); text != "" {
			msg = text
		}
	}
	if strings.HasPrefix(msg, notTidyPrefix) {
		reason := strings.TrimPrefix(msg, notTidySuggested)
		if reason == msg {
			reason = strings.TrimPrefix(msg, notTidyPrefix)
		}
		return &NotTidyError{Reason: strings.TrimSpace(strings.TrimPrefix(reason, ":"))}
	}
	if msg != err.Error() {
		return errors.New(msg)
	}
	return err
}

// fileState is one file's content, or its absence.
type fileState struct {
	exists bool
	data   []byte
}

func (s fileState) equal(o fileState) bool {
	return s.exists == o.exists && bytes.Equal(s.data, o.data)
}

type moduleFiles struct {
	module fileState
	local  fileState
}

func snapshot(root string) (moduleFiles, error) {
	module, err := readState(filepath.Join(root, filepath.FromSlash(moduleFile)))
	if err != nil {
		return moduleFiles{}, err
	}
	local, err := readState(filepath.Join(root, filepath.FromSlash(localFile)))
	if err != nil {
		return moduleFiles{}, err
	}
	return moduleFiles{module: module, local: local}, nil
}

func readState(path string) (fileState, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fileState{}, nil
	}
	if err != nil {
		return fileState{}, fmt.Errorf("reading %s: %w", path, err)
	}
	return fileState{exists: true, data: data}, nil
}
