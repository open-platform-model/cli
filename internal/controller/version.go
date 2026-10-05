package controller

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/mod/module"
	"golang.org/x/mod/semver"

	"github.com/open-platform-model/cli/internal/publish"
)

// controllerPackageDir is the directory of the controller module that names the
// controller release a module version deploys, readable without a render:
// `cue eval ./controller -e Version`.
const controllerPackageDir = "controller"

// ModuleFetcher fetches one published module version's source tree.
// modconfig's cached registry satisfies it, as it does modref.Source.
type ModuleFetcher interface {
	Fetch(ctx context.Context, m module.Version) (module.SourceLoc, error)
}

// VersionError reports that a module version does not state, in a
// form install can trust, the controller release it deploys.
type VersionError struct {
	// ModuleVersion is the module version read, "v"-prefixed.
	ModuleVersion string
	Reason        string
}

func (e *VersionError) Error() string {
	return fmt.Sprintf("%s %s does not state the controller it deploys: %s",
		ControllerModulePath, strings.TrimPrefix(e.ModuleVersion, "v"), e.Reason)
}

// ModuleMajor returns the controller module's major-qualified path for a
// version ("opmodel.dev/modules/opm_controller@v0" for "v0.1.0").
func ModuleMajor(moduleVersion string) string {
	return ControllerModulePath + "@" + semver.Major(moduleVersion)
}

// ReadControllerVersion returns the controller release a version of the controller
// module deploys, "v"-prefixed ("v1.0.0-beta.7"), read from the module's
// controller package without a render: the package's Version field, bare
// SemVer. moduleVersion is the module's published version, "v"-prefixed.
// A fetch failure is a *publish.ConnectivityError; a module that carries no
// controller package, no Version or a Version that is not a SemVer release is
// an *VersionError.
func ReadControllerVersion(ctx context.Context, src ModuleFetcher, moduleVersion string) (string, error) {
	mv, err := module.NewVersion(ModuleMajor(moduleVersion), moduleVersion)
	if err != nil {
		return "", fmt.Errorf("forming module version %s %s: %w", ControllerModulePath, moduleVersion, err)
	}
	loc, err := src.Fetch(ctx, mv)
	if err != nil {
		return "", &publish.ConnectivityError{Op: fmt.Sprintf("fetching %s", mv), Err: err}
	}
	return readControllerVersionFrom(loc, moduleVersion)
}

// readControllerVersionFrom reads Version from the controller package of an
// already fetched module tree.
func readControllerVersionFrom(loc module.SourceLoc, moduleVersion string) (string, error) {
	refuse := func(format string, args ...any) error {
		return &VersionError{ModuleVersion: moduleVersion, Reason: fmt.Sprintf(format, args...)}
	}

	dir := path.Join(loc.Dir, controllerPackageDir)
	entries, err := fs.ReadDir(loc.FS, dir)
	if err != nil {
		return "", refuse("it has no %s package (%v)", controllerPackageDir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".cue") {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		return "", refuse("its %s package holds no CUE file", controllerPackageDir)
	}
	sort.Strings(files)

	// The package is plain data with no imports, so each file compiles on
	// its own and the files unify as one package would.
	ctx := cuecontext.New()
	pkg := ctx.CompileString("{}")
	for _, name := range files {
		data, err := fs.ReadFile(loc.FS, path.Join(dir, name))
		if err != nil {
			return "", refuse("reading %s/%s: %v", controllerPackageDir, name, err)
		}
		v := ctx.CompileBytes(data, cue.Filename(path.Join(controllerPackageDir, name)))
		if err := v.Err(); err != nil {
			return "", refuse("compiling %s/%s: %v", controllerPackageDir, name, err)
		}
		pkg = pkg.Unify(v)
	}
	if err := pkg.Err(); err != nil {
		return "", refuse("its %s package does not evaluate: %v", controllerPackageDir, err)
	}

	field := pkg.LookupPath(cue.ParsePath("Version"))
	if !field.Exists() {
		return "", refuse("its %s package has no Version", controllerPackageDir)
	}
	version, err := field.String()
	if err != nil {
		return "", refuse("%s.Version is not a concrete string: %v", controllerPackageDir, err)
	}
	tagged := "v" + version
	if strings.HasPrefix(version, "v") || !semver.IsValid(tagged) || semver.Canonical(tagged) != tagged {
		return "", refuse("%s.Version %q is not a SemVer release such as \"1.0.0-beta.7\"", controllerPackageDir, version)
	}
	return tagged, nil
}
