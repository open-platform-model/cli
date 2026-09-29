// Package instinit generates the standalone instance package `opm instance
// init` writes for a published module (0016:D1/D7/D9): cue.mod/module.cue
// pinning the module and core, instance.cue binding the module to a
// #ModuleInstance, and values.cue holding the starting values.
//
// Render is pure: every input is resolved by the caller (the module version
// by internal/modref, the values by PickValues) and the same Input always
// yields the same bytes. Write stages the files, completes the dependency
// closure with cuemod.Tidy, and moves the result into place all or nothing.
package instinit

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"
)

// ValuesSource names what populated values.cue.
type ValuesSource string

const (
	// FromInitValues is the module's initValues, the author's starting point.
	FromInitValues ValuesSource = "initValues"
	// FromDebugValues is the module's debugValues, the author's test values.
	FromDebugValues ValuesSource = "debugValues"
	// FromEmpty is `values: {}`: the module offered nothing usable.
	FromEmpty ValuesSource = "empty"
)

// LanguageVersion is the CUE language version the package's module file
// declares, the one the seeded local platform module declares.
const LanguageVersion = "v0.17.0"

// File names inside the package, slash-separated.
const (
	ModuleFile   = "cue.mod/module.cue"
	InstanceFile = "instance.cue"
	ValuesFile   = "values.cue"
)

// DefaultPackageModulePath is the package's own module path when
// --module-path is not given. The path is never published or looked up.
func DefaultPackageModulePath(name string) string {
	return "instance.local/" + name + "@v0"
}

// Input is everything Render needs; every field is resolved before Render runs.
type Input struct {
	// Name and Namespace are the instance's metadata.
	Name, Namespace string
	// PackageModulePath is the package's own major-qualified module path.
	PackageModulePath string
	// Module is the deployed module at its exact version, e.g.
	// opmodel.dev/modules/web_app@v1 at v1.0.4.
	Module module.Version
	// Core is opmodel.dev/core at the version the module itself declares.
	Core module.Version
	// Values is the CUE expression for the `values` field.
	Values []byte
	// Source is what Values came from; values.cue opens by naming it.
	Source ValuesSource
}

// Files maps a package-relative, slash-separated path to its bytes.
type Files map[string][]byte

// Render produces the three package files. It is pure and deterministic:
// the same Input yields byte-identical files.
func Render(in Input) (Files, error) {
	if in.Name == "" || in.Namespace == "" || in.PackageModulePath == "" {
		return nil, errors.New("instinit: name, namespace and package module path are required")
	}
	if !in.Module.IsValid() || !in.Core.IsValid() {
		return nil, errors.New("instinit: module and core versions are required")
	}
	if in.Module.Version() == "" || in.Core.Version() == "" {
		return nil, errors.New("instinit: module and core must carry a version")
	}

	mod, err := renderModuleFile(in)
	if err != nil {
		return nil, err
	}
	inst, err := renderInstanceFile(in)
	if err != nil {
		return nil, err
	}
	vals, err := renderValuesFile(in)
	if err != nil {
		return nil, err
	}
	return Files{ModuleFile: mod, InstanceFile: inst, ValuesFile: vals}, nil
}

// renderModuleFile pins the module and core exactly; tidy completes the
// closure after staging (design.md, Data flow).
func renderModuleFile(in Input) ([]byte, error) {
	data, err := modfile.Format(&modfile.File{
		Module:   in.PackageModulePath,
		Language: &modfile.Language{Version: LanguageVersion},
		Deps: map[string]*modfile.Dep{
			in.Module.Path(): {Version: in.Module.Version()},
			in.Core.Path():   {Version: in.Core.Version()},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("rendering %s: %w", ModuleFile, err)
	}
	return data, nil
}

// renderInstanceFile mirrors the instance package the library's synthesis
// stages in memory, as a standalone package importing the module by path.
func renderInstanceFile(in Input) ([]byte, error) {
	var b strings.Builder
	b.WriteString("package instance\n\n")
	b.WriteString("import (\n")
	fmt.Fprintf(&b, "\tcore %s\n", literal.String.Quote(in.Core.Path()))
	fmt.Fprintf(&b, "\topmModule %s\n", literal.String.Quote(in.Module.Path()))
	b.WriteString(")\n\n")
	b.WriteString("core.#ModuleInstance\n\n")
	b.WriteString("metadata: {\n")
	fmt.Fprintf(&b, "\tname:      %s\n", literal.String.Quote(in.Name))
	fmt.Fprintf(&b, "\tnamespace: %s\n", literal.String.Quote(in.Namespace))
	b.WriteString("}\n\n")
	b.WriteString("#module: opmModule\n")
	return formatFile(InstanceFile, b.String())
}

// renderValuesFile opens with a comment naming the source, then the values.
func renderValuesFile(in Input) ([]byte, error) {
	values := bytes.TrimSpace(in.Values)
	if len(values) == 0 {
		values = []byte("{}")
	}
	var b strings.Builder
	for _, line := range sourceComment(in.Source) {
		b.WriteString("// " + line + "\n")
	}
	b.WriteString("package instance\n\n")
	b.WriteString("values: ")
	b.Write(values)
	b.WriteString("\n")
	return formatFile(ValuesFile, b.String())
}

// sourceComment is the lead comment of values.cue. It reaches the user's
// tree, so it names no enhancement.
func sourceComment(src ValuesSource) []string {
	switch src {
	case FromInitValues:
		return []string{"Starting values from the module's initValues. Edit them for this instance."}
	case FromDebugValues:
		return []string{
			"Starting values from the module's debugValues, the author's test values.",
			"Review them before deploying.",
		}
	case FromEmpty:
	}
	return []string{
		"The module offers no starting values. Fill in what its #config requires;",
		"opm instance vet names every missing field.",
	}
}

// formatFile runs src through the CUE formatter so the output is canonical.
func formatFile(name, src string) ([]byte, error) {
	data, err := format.Source([]byte(src))
	if err != nil {
		return nil, fmt.Errorf("rendering %s: %w", name, err)
	}
	return data, nil
}
