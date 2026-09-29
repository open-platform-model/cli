package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"

	"github.com/open-platform-model/library/opm/helper/platformmodule"

	"github.com/open-platform-model/cli/pkg/loader"
)

// ModuleDepsPlatformModulePath is the generated module's identity, in the
// reserved, never-published platforms namespace beside
// ClusterPlatformModulePath.
const ModuleDepsPlatformModulePath = "opmodel.dev/platforms/module-deps@v0"

// CatalogPathPrefix selects the deps that become registry entries: every
// catalog the module pins lives under it.
const CatalogPathPrefix = "opmodel.dev/catalogs/"

// The generated platform's metadata. The platform is never applied, so the
// name reaches no cluster; the type matches the local default's.
const (
	moduleDepsPlatformName = "module-deps"
	moduleDepsPlatformType = "kubernetes"
)

// localModuleFileName is the carried replacements file, relative to the
// generated module directory.
const localModuleFileName = "cue.mod/local-module.cue"

// ErrModuleDepsFile marks a module whose committed cue.mod/module.cue cannot
// be parsed: the module is invalid, not the platform.
var ErrModuleDepsFile = errors.New("reading the module's dependency pins")

// ModuleDeps is what a module-deps platform is generated from.
type ModuleDeps struct {
	// ModFile is the module's committed cue.mod/module.cue, and
	// ModFileName names it in errors.
	ModFile     []byte
	ModFileName string
	// Replacements are the replaceWith entries of the module's
	// cue.mod/local-module.cue, with ModuleRoot resolving relative
	// directories. Both empty for a module with no local context
	// (a published module).
	Replacements []loader.LocalReplacement
	ModuleRoot   string
}

// GenerateModuleDepsModule writes the platform module for deps under
// opts.CacheDir and returns its directory, the registry entries it carries
// and the replacements it carried (path -> the target written into its
// cue.mod/local-module.cue: an absolute directory, or a major-qualified
// module path).
//
// The module carries one enabled registry entry per dependency under
// CatalogPathPrefix, at the version the module pins. Its dependency list is
// the closure of those pins plus core, at the higher of the module's core pin
// and the release the kernel was verified against, so a catalog's own
// dependencies that the module's tidied list omits are present. A module
// replacement of a path the closure pins is carried into the generated
// module's local file, and the closure reads that path's requirements from
// the replacement rather than the registry. The directory is content-hashed
// under the cache exactly as GenerateClusterModule's is.
func GenerateModuleDepsModule(ctx context.Context, deps ModuleDeps, opts GenerateOptions) (dir string, entries []platformmodule.Entry, carried map[string]string, err error) {
	mf, err := modfile.Parse(deps.ModFile, deps.ModFileName)
	if err != nil {
		return "", nil, nil, fmt.Errorf("%w: parsing %s: %w", ErrModuleDepsFile, deps.ModFileName, err)
	}
	if opts.CacheDir == "" {
		return "", nil, nil, errors.New("platform cache directory is not set")
	}

	entries, roots := moduleDepsRoots(mf)
	base, err := modFileSource(opts)
	if err != nil {
		return "", nil, nil, err
	}
	src := newReplacedModFiles(base, deps)
	pinned, err := platformmodule.Closure(ctx, src, roots)
	if err != nil {
		// An unpublished pin or an unreachable registry: the error names
		// the module path and version.
		return "", nil, nil, fmt.Errorf("module deps platform: %w", err)
	}

	files, err := platformmodule.Generate(platformmodule.Input{
		Name:       moduleDepsPlatformName,
		Type:       moduleDepsPlatformType,
		ModulePath: ModuleDepsPlatformModulePath,
		Entries:    entries,
		Deps:       pinned,
	})
	if err != nil {
		return "", nil, nil, fmt.Errorf("generating module deps platform module: %w", err)
	}
	carried, err = carryReplacements(files, src.targets)
	if err != nil {
		return "", nil, nil, err
	}
	dir, err = writeCached(opts.CacheDir, files)
	if err != nil {
		return "", nil, nil, err
	}
	return dir, entries, carried, nil
}

// moduleDepsRoots selects a module's registry entries and closure roots from
// its committed module file: one enabled entry per dependency under
// CatalogPathPrefix at its bare version, in path order, and as roots the
// library's (core at the verified release plus every entry's catalog) with
// the module's own core pin added, so the closure keeps the higher of the
// two. Every other dependency stays the instance's.
func moduleDepsRoots(mf *modfile.File) ([]platformmodule.Entry, []platformmodule.Dep) {
	entries := []platformmodule.Entry{}
	for path, dep := range mf.Deps {
		if !strings.HasPrefix(path, CatalogPathPrefix) || dep == nil {
			continue
		}
		entries = append(entries, platformmodule.Entry{Path: path, Version: strings.TrimPrefix(dep.Version, "v"), Enable: true})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })

	roots := platformmodule.Roots(entries)
	if core, ok := mf.Deps[platformmodule.CorePath]; ok && core != nil && core.Version != "" {
		roots = append(roots, platformmodule.Dep{Path: platformmodule.CorePath, Version: core.Version})
	}
	return entries, roots
}

// modFileSource returns the injected module-file source, or one resolving
// through the configured registry.
func modFileSource(opts GenerateOptions) (platformmodule.ModFileSource, error) {
	if opts.ModFiles != nil {
		return opts.ModFiles, nil
	}
	src, err := platformmodule.NewRegistry(platformmodule.RegistryConfig{
		Registry:   opts.Registry,
		ClientType: "opm-cli",
		Env:        os.Environ(),
	})
	if err != nil {
		return nil, fmt.Errorf("configuring module registry: %w", err)
	}
	return src, nil
}

// replacedModFiles serves a replaced module's module file from its
// replacement (a directory's cue.mod/module.cue, or the target module's
// published file) and delegates every other lookup to base.
type replacedModFiles struct {
	base platformmodule.ModFileSource
	// targets maps a replaced major-qualified path to its replacement, a
	// directory target made absolute.
	targets map[string]loader.LocalReplacement
}

// newReplacedModFiles wraps base with the module's replacements, resolving a
// relative directory target against the module root.
func newReplacedModFiles(base platformmodule.ModFileSource, deps ModuleDeps) *replacedModFiles {
	targets := make(map[string]loader.LocalReplacement, len(deps.Replacements))
	for _, r := range deps.Replacements {
		if loader.IsDirectoryTarget(r.ReplaceWith) && !filepath.IsAbs(r.ReplaceWith) {
			r.ReplaceWith = filepath.Join(deps.ModuleRoot, filepath.FromSlash(r.ReplaceWith))
		}
		targets[r.Path] = r
	}
	return &replacedModFiles{base: base, targets: targets}
}

// ModFile implements platformmodule.ModFileSource.
func (r *replacedModFiles) ModFile(ctx context.Context, mv module.Version) (*modfile.File, error) {
	rep, ok := r.targets[mv.Path()]
	if !ok {
		return r.base.ModFile(ctx, mv)
	}
	if loader.IsDirectoryTarget(rep.ReplaceWith) {
		name := filepath.Join(rep.ReplaceWith, "cue.mod", "module.cue")
		data, err := os.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("reading the replacement of %s: %w", mv.Path(), err)
		}
		return modfile.Parse(data, name)
	}
	target, err := moduleTarget(rep)
	if err != nil {
		return nil, err
	}
	return r.base.ModFile(ctx, target)
}

// moduleTarget is the module version a module-path replacement resolves at.
func moduleTarget(rep loader.LocalReplacement) (module.Version, error) {
	if rep.TargetVersion == "" {
		return module.Version{}, fmt.Errorf("replacement of %s with %s: no version pinned for the target", rep.Path, rep.ReplaceWith)
	}
	mv, err := module.NewVersion(majorPath(rep.ReplaceWith, rep.TargetVersion), rep.TargetVersion)
	if err != nil {
		return module.Version{}, fmt.Errorf("replacement of %s with %s: %w", rep.Path, rep.ReplaceWith, err)
	}
	return mv, nil
}

// majorPath is a module-path target qualified by its major only
// ("example.com/fork@v0"), whether it was written with a major or a full
// version.
func majorPath(target, version string) string {
	path, _, _ := strings.Cut(target, "@")
	major, _, _ := strings.Cut(version, ".")
	return path + "@" + major
}

// carryReplacements adds the generated module's cue.mod/local-module.cue to
// files, holding every replacement whose path the generated dependency list
// pins, and returns what it carried. Nothing is added when nothing is
// carried. A module-path target is written major-qualified, with its version
// as a dependency entry of its own, the form cue/load resolves it from.
func carryReplacements(files platformmodule.Files, targets map[string]loader.LocalReplacement) (map[string]string, error) {
	carried := map[string]string{}
	if len(targets) == 0 {
		return carried, nil
	}
	base, err := modfile.Parse(files[platformmodule.ModuleFileName], platformmodule.ModuleFileName)
	if err != nil {
		return nil, fmt.Errorf("reading the generated module file: %w", err)
	}
	local := &modfile.File{Module: base.Module, Language: base.Language, Deps: map[string]*modfile.Dep{}}
	for path, rep := range targets {
		dep, pinned := base.Deps[path]
		if !pinned {
			continue
		}
		target := rep.ReplaceWith
		if !loader.IsDirectoryTarget(target) {
			mv, err := moduleTarget(rep)
			if err != nil {
				return nil, err
			}
			target = mv.Path()
			if _, listed := local.Deps[target]; !listed {
				local.Deps[target] = &modfile.Dep{Version: mv.Version()}
			}
		}
		local.Deps[path] = &modfile.Dep{Version: dep.Version, ReplaceWith: target}
		carried[path] = target
	}
	if len(carried) == 0 {
		return carried, nil
	}
	data, err := modfile.FormatLocal(local, base)
	if err != nil {
		return nil, fmt.Errorf("formatting the carried %s: %w", localModuleFileName, err)
	}
	files[localModuleFileName] = data
	return carried, nil
}
