package loader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modfile"

	oerrors "github.com/opmodel/cli/pkg/errors"
)

const catalogModulePath = "opmodel.dev/core/v1alpha1@v1"

// SynthesizeOptions configures synthesis of a #ModuleInstance wrapper from a
// module-package directory.
type SynthesizeOptions struct {
	// Name overrides the synthetic metadata.name. When empty, the loader
	// derives "<module.metadata.name>-debug" from the module's metadata.
	Name string

	// Namespace overrides the synthetic metadata.namespace. When empty, the
	// loader uses "default".
	Namespace string
}

// SynthesisResult is what SynthesizeModuleInstanceFromPackage returns. The
// callers pass Spec into module.ParseModuleInstance together with values; Module
// is exposed so callers can read debugValues without re-loading.
type SynthesisResult struct {
	// Spec is the synthesized cue.Value shaped like a #ModuleInstance, with
	// #module and metadata filled. values is left for downstream filling by
	// module.ParseModuleInstance.
	Spec cue.Value

	// ModuleValue is the loaded user-module package value (for debugValues
	// lookup or other module-side fields).
	ModuleValue cue.Value

	// ModuleName is the module's metadata.name (best-effort, may be empty).
	ModuleName string
}

// SynthesizeModuleInstanceFromPackage loads the user's module CUE package and
// composes a #ModuleInstance wrapper around it via a small synthetic CUE module
// that imports the catalog at the same version the user's modfile pins. The
// returned spec is ready to feed into module.ParseModuleInstance together with
// values from -f files or the module's debugValues.
//
// Was: SynthesizeModuleReleaseFromPackage (enhancement 0002 D8 hard-rename).
func SynthesizeModuleInstanceFromPackage(ctx *cue.Context, modulePath string, opts SynthesizeOptions) (*SynthesisResult, error) {
	absModule, err := filepath.Abs(modulePath)
	if err != nil {
		return nil, fmt.Errorf("resolving module directory: %w", err)
	}

	info, err := os.Stat(absModule)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("module path %q not found", modulePath)
		}
		return nil, fmt.Errorf("accessing module directory %q: %w", absModule, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("module path %q is not a directory", absModule)
	}

	catalogVersion, err := readModuleCatalogPin(absModule)
	if err != nil {
		return nil, err
	}

	// Load user module first — surfaces dep-resolution errors before we set
	// up the synth anchor.
	modVal, err := loadUserModule(ctx, absModule)
	if err != nil {
		return nil, err
	}

	moduleName, _ := lookupString(modVal, "metadata.name")

	synthName := opts.Name
	if synthName == "" {
		if moduleName == "" {
			return nil, fmt.Errorf("module has no metadata.name and no --name override was provided")
		}
		synthName = moduleName + "-debug"
	}
	synthNamespace := opts.Namespace
	if synthNamespace == "" {
		synthNamespace = "default"
	}

	synthVal, err := loadSynthWrapper(ctx, catalogVersion)
	if err != nil {
		return nil, err
	}

	spec := synthVal.
		FillPath(cue.MakePath(cue.Def("module")), modVal).
		FillPath(cue.ParsePath("metadata.name"), synthName).
		FillPath(cue.ParsePath("metadata.namespace"), synthNamespace)
	if err := spec.Err(); err != nil {
		return nil, fmt.Errorf("composing synthetic release: %w", err)
	}

	return &SynthesisResult{
		Spec:        spec,
		ModuleValue: modVal,
		ModuleName:  moduleName,
	}, nil
}

// readModuleCatalogPin walks up from modulePath looking for a cue.mod/module.cue,
// parses it, and returns the version string pinned for opmodel.dev/core/v1alpha1@v1.
func readModuleCatalogPin(modulePath string) (string, error) {
	modfilePath, err := findModuleCueFile(modulePath)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(modfilePath)
	if err != nil {
		return "", fmt.Errorf("reading module's cue.mod/module.cue: %w", err)
	}

	mf, err := modfile.Parse(data, modfilePath)
	if err != nil {
		return "", fmt.Errorf("reading module's cue.mod/module.cue: %w", err)
	}

	dep, ok := mf.Deps[catalogModulePath]
	if !ok || dep == nil || dep.Version == "" {
		return "", &oerrors.DetailError{
			Type:    "module missing catalog dep",
			Message: fmt.Sprintf("module must declare %q as a dependency to be buildable", catalogModulePath),
			Hint:    fmt.Sprintf("add a dep on %q to %s", catalogModulePath, modfilePath),
		}
	}

	return dep.Version, nil
}

// findModuleCueFile walks from start up to the filesystem root looking for
// cue.mod/module.cue. Mirrors CUE's own loader behavior for module roots.
func findModuleCueFile(start string) (string, error) {
	dir := start
	for {
		candidate := filepath.Join(dir, "cue.mod", "module.cue")
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", &oerrors.DetailError{
				Type:    "module missing cue.mod",
				Message: fmt.Sprintf("no cue.mod/module.cue found at or above %q", start),
				Hint:    "initialize the module with `cue mod init` and add the catalog as a dep",
			}
		}
		dir = parent
	}
}

// loadUserModule loads the user's module CUE package as a whole.
func loadUserModule(ctx *cue.Context, modulePath string) (cue.Value, error) {
	cfg := &load.Config{Dir: modulePath}
	insts := load.Instances([]string{"."}, cfg)
	if len(insts) == 0 {
		return cue.Value{}, fmt.Errorf("no CUE instances found in %s", modulePath)
	}
	if insts[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading module package from %s: %w", modulePath, insts[0].Err)
	}
	val := ctx.BuildInstance(insts[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building module package from %s: %w", modulePath, err)
	}
	return val, nil
}

// loadSynthWrapper builds and loads the in-memory synth CUE module that imports
// `opmodel.dev/core/v1alpha1/modulerelease@v1` and applies #ModuleRelease at
// the top level. The temp anchor is removed before this function returns.
//
// FLAG (0002, out of X1 scope): the catalog import path
// `opmodel.dev/core/v1alpha1/modulerelease@v1` and the `#ModuleRelease`
// definition are the catalog-side wire contract. core@v1 ships a single `core`
// package, so this path is pre-existing drift unrelated to this rename; it is
// left verbatim here and tracked as a separate catalog-pin follow-up.
func loadSynthWrapper(ctx *cue.Context, catalogVersion string) (cue.Value, error) {
	anchor, err := os.MkdirTemp("", "opm-synth-")
	if err != nil {
		return cue.Value{}, fmt.Errorf("creating synth anchor: %w", err)
	}
	defer os.RemoveAll(anchor)

	absAnchor, err := filepath.Abs(anchor)
	if err != nil {
		return cue.Value{}, fmt.Errorf("resolving synth anchor path: %w", err)
	}

	modfileSrc := fmt.Sprintf(`module: "opm.local/synth@v0"
language: {
	version: "v0.16.0"
}
source: {
	kind: "self"
}
deps: {
	%q: {
		v: %q
	}
}
`, catalogModulePath, catalogVersion)

	wrapperSrc := `package synth

import mr "opmodel.dev/core/v1alpha1/modulerelease@v1"

mr.#ModuleRelease
`

	modfilePath := filepath.ToSlash(filepath.Join(absAnchor, "cue.mod", "module.cue"))
	wrapperPath := filepath.ToSlash(filepath.Join(absAnchor, "wrapper.cue"))

	cfg := &load.Config{
		Dir: absAnchor,
		Overlay: map[string]load.Source{
			modfilePath: load.FromString(modfileSrc),
			wrapperPath: load.FromString(wrapperSrc),
		},
	}

	insts := load.Instances([]string{"."}, cfg)
	if len(insts) == 0 {
		return cue.Value{}, errors.New("no CUE instances found for synth wrapper")
	}
	if insts[0].Err != nil {
		return cue.Value{}, fmt.Errorf("resolving catalog dep for synth wrapper: %w", insts[0].Err)
	}

	val := ctx.BuildInstance(insts[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("resolving catalog dep for synth wrapper: %w", err)
	}
	return val, nil
}

func lookupString(v cue.Value, path string) (string, bool) {
	field := v.LookupPath(cue.ParsePath(path))
	if !field.Exists() {
		return "", false
	}
	s, err := field.String()
	if err != nil {
		return "", false
	}
	return s, true
}
