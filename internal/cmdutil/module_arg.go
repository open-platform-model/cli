package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/modref"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
)

// ModuleArgKind is what the positional argument of `opm module build` and
// `opm module apply` names.
type ModuleArgKind int

const (
	// LocalModuleDir is a module package directory on disk.
	LocalModuleDir ModuleArgKind = iota
	// PublishedModule is a published module path, resolved in a registry.
	PublishedModule
)

// ClassifyModuleArg decides by shape, before any registry access, whether arg
// names a local directory or a published module: ".", "..", a "./" or "../"
// prefix, or an absolute path is local; otherwise a first element holding a
// dot is a module path; anything else is local. A module-path-shaped argument
// that also exists as a directory is refused as ambiguous (exit 2), naming the
// "./" spelling.
func ClassifyModuleArg(arg string) (ModuleArgKind, error) {
	if isExplicitLocal(arg) {
		return LocalModuleDir, nil
	}
	first, _, _ := strings.Cut(filepath.ToSlash(arg), "/")
	if !strings.Contains(first, ".") {
		return LocalModuleDir, nil
	}
	if info, err := os.Stat(arg); err == nil && info.IsDir() {
		return 0, &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err:  fmt.Errorf("%q is both a directory and a module path; use ./%s for the directory", arg, arg),
		}
	}
	return PublishedModule, nil
}

// isExplicitLocal reports whether arg is spelled as a filesystem path.
func isExplicitLocal(arg string) bool {
	if arg == "." || arg == ".." || filepath.IsAbs(arg) {
		return true
	}
	slashed := filepath.ToSlash(arg)
	return strings.HasPrefix(slashed, "./") || strings.HasPrefix(slashed, "../")
}

// ModuleArg is the resolved positional argument of `opm module build` and
// `opm module apply`: exactly one of Dir and Published is set.
type ModuleArg struct {
	Dir       string
	Published *modref.Resolution
}

// ResolveModuleArg classifies the positional argument and resolves it: a
// local directory is checked to exist and be a directory (verb names the
// command for the file hint: "build" or "apply"); a published module path is
// parsed and resolved in the registry, and the resolution reported on
// standard error. --version with a local directory is refused (exit 2).
// Refusals print through the house funnel and exit 2; an unreachable registry
// exits 3.
func ResolveModuleArg(ctx context.Context, cfg *config.GlobalConfig, args []string, version, verb string) (*ModuleArg, error) {
	arg := ResolveModulePath(args)
	kind, err := ClassifyModuleArg(arg)
	if err != nil {
		return nil, err
	}
	if kind == LocalModuleDir {
		if version != "" {
			return nil, &opmexit.ExitError{
				Code: opmexit.ExitValidationError,
				Err:  fmt.Errorf("--version applies only to a published module; %s is a local directory", arg),
			}
		}
		if err := checkModuleDir(arg, verb); err != nil {
			return nil, err
		}
		return &ModuleArg{Dir: arg}, nil
	}

	res, err := resolvePublished(ctx, cfg.Registry, arg, version)
	if err != nil {
		return nil, moduleArgError(err)
	}
	for _, line := range modref.Report(res) {
		output.Info(line)
	}
	return &ModuleArg{Published: res}, nil
}

func resolvePublished(ctx context.Context, registry, arg, version string) (*modref.Resolution, error) {
	path, err := modref.ParsePath(arg)
	if err != nil {
		return nil, err
	}
	sel, err := modref.ParseSelector(version)
	if err != nil {
		return nil, err
	}
	src, err := modref.NewSource(registry)
	if err != nil {
		return nil, err
	}
	return modref.Resolve(ctx, src, modref.Request{
		Path:      path,
		Selector:  sel,
		CoreMajor: CoreMajor(),
		Registry:  modref.Route(registry, path),
	})
}

// CoreMajor is the major of opmodel.dev/core this CLI renders against ("v2").
func CoreMajor() string {
	return semver.Major(schema.DefaultSchemaVersion())
}

// moduleArgError maps resolver errors onto the house funnels: refusals print
// and exit 2, connectivity failures exit 3, the rest exit 1.
func moduleArgError(err error) error {
	var refusalErr *modref.RefusalError
	if errors.As(err, &refusalErr) {
		PrintRefusals([]publish.Refusal{refusalErr.Refusal})
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	var connErr *publish.ConnectivityError
	if errors.As(err, &connErr) {
		return &opmexit.ExitError{Code: opmexit.ExitConnectivityError, Err: err}
	}
	return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
}

// checkModuleDir requires a local module argument to be an existing
// directory: a CUE package spans every file in its directory, so a file is
// pointed at the instance command of the same verb.
func checkModuleDir(dir, verb string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("module path %q not found", dir)}
		}
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("stat %q: %w", dir, err)}
	}
	if !info.IsDir() {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("module %s expects a directory; CUE packages span all files in a dir. Use 'opm instance %s %s' for an instance file", verb, verb, dir),
		}
	}
	return nil
}
