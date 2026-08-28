package scaffold

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/cueedit"
	"github.com/open-platform-model/cli/internal/publish"
)

// repairLanguageVersion is the CUE language version a created cue.mod
// declares — structure, not identity, so writing it invents nothing (D20).
const repairLanguageVersion = "v0.17.0"

// RepairAction is one file `init` will create or edit, stated so the second
// confirmation gives the author something to judge: the file, and for an
// edit, the current value beside its replacement (D20).
type RepairAction struct {
	// File is the dir-relative path.
	File string
	// Create reports a file that does not exist yet; false is an edit.
	Create bool
	// Field names what changes within the file.
	Field string
	// Current is the value an edit replaces; empty for a create.
	Current string
	// New is the value written.
	New string

	apply func() error
}

// RepairPlan is everything repair mode decided before touching a byte.
type RepairPlan struct {
	// Dir is the module directory under repair.
	Dir string
	// ModulePath is the identity every action aligns to — always sourced
	// from the tree or the command's argument, never invented.
	ModulePath string
	// Actions in application order. Empty means nothing to repair.
	Actions []RepairAction
}

// DetectRepair inspects an existing tree and builds the repair plan: a
// missing or disagreeing cue.mod `module:` line, a missing identity package,
// or a disagreeing identity ModulePath (D20's repairable set — deeper
// conformance stays with `opm module vet`, which the command points at
// afterwards).
//
// argPath is the module path the invocation supplied, "" for none. Identity
// resolution never invents: the argument wins when given; otherwise the
// tree's own statements (identity, then cue.mod) must agree — a tree that
// disagrees with itself with no argument to arbitrate is refused, as is a
// tree that states no path at all.
//
// The kernel loads the tree best-effort to source a version for identity
// creation from metadata.version — the tree's own statement. k may be nil
// when no load should be attempted.
func DetectRepair(ctx context.Context, k *kernel.Kernel, registry, dir, argPath string) (*RepairPlan, error) {
	cueModPath, cueModErr := cueedit.ReadCueModModule(dir)
	if cueModErr != nil && !errors.Is(cueModErr, cueedit.ErrCueModShape) {
		return nil, cueModErr
	}
	idPath, idErr := cueedit.ReadIdentityModulePath(dir)
	if idErr != nil && !errors.Is(idErr, cueedit.ErrIdentityShape) {
		return nil, idErr
	}

	target, err := resolveRepairPath(argPath, idPath, cueModPath, dir)
	if err != nil {
		return nil, err
	}
	if err := ValidateNewModulePath(target); err != nil {
		return nil, err
	}

	p := &RepairPlan{Dir: dir, ModulePath: target}

	if err := planCueMod(p, cueModPath, cueModErr, target); err != nil {
		return nil, err
	}
	if err := planIdentity(ctx, k, registry, p, idPath, idErr, target); err != nil {
		return nil, err
	}
	return p, nil
}

// resolveRepairPath picks the one module path every action aligns to.
func resolveRepairPath(argPath, idPath, cueModPath, dir string) (string, error) {
	if argPath != "" {
		return argPath, nil
	}
	switch {
	case idPath != "" && cueModPath != "" && idPath != cueModPath:
		return "", &RefusalError{publish.Refusal{
			Headline: fmt.Sprintf("%s disagrees with itself about where it lives, and init will not choose", dir),
			Evidence: [][]string{
				{"identity", idPath, filepath.Join("identity", "identity.cue")},
				{"cue.mod", cueModPath, filepath.Join("cue.mod", "module.cue")},
			},
			Consequence: "Repair aligns the tree to one identity; picking a side here would invent\nthe answer.",
			Action:      fmt.Sprintf("State which is right:  opm mod init %s", orPlaceholder(idPath)),
		}}
	case idPath != "":
		return idPath, nil
	case cueModPath != "":
		return cueModPath, nil
	default:
		return "", &RefusalError{publish.Refusal{
			Headline:    fmt.Sprintf("%s states no module path anywhere", dir),
			Consequence: "Repair writes structure; only you supply identity (D20). There is nothing\nin this tree to align to.",
			Action:      "Pass the path:  opm mod init <module-path>",
		}}
	}
}

func orPlaceholder(s string) string {
	if s == "" {
		return "<module-path>"
	}
	return s
}

// planCueMod plans the cue.mod half: create it when absent, realign its
// module line when it disagrees — refusing the realignment when it would
// strand self-imports of the current path, because repair never rewrites
// imports in a tree the author owns.
func planCueMod(p *RepairPlan, cueModPath string, cueModErr error, target string) error {
	file := filepath.Join("cue.mod", "module.cue")
	dir := p.Dir

	if cueModErr != nil {
		if _, statErr := os.Stat(filepath.Join(dir, file)); statErr == nil {
			// Present but malformed: not repairable without guessing what the
			// author meant the rest of the file to say.
			return fmt.Errorf("repair cannot rewrite a malformed %s: %w", file, cueModErr)
		}
		content := fmt.Sprintf("module: %q\nlanguage: {\n\tversion: %q\n}\nsource: {\n\tkind: \"self\"\n}\n", target, repairLanguageVersion)
		p.Actions = append(p.Actions, RepairAction{
			File: file, Create: true, Field: "module", New: target,
			apply: func() error {
				if err := os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644) //nolint:gosec // G306: repaired source files are project files, not secrets
			},
		})
		return nil
	}

	if cueModPath == target {
		return nil
	}
	strands, err := cueedit.ListSelfImports(dir, cueModPath)
	if err != nil {
		return err
	}
	if len(strands) > 0 {
		sort.Strings(strands)
		ev := make([][]string, 0, len(strands)+2)
		ev = append(ev, []string{"cue.mod", cueModPath}, []string{"target", target})
		for _, f := range strands {
			ev = append(ev, []string{"imports it", f})
		}
		return &RefusalError{publish.Refusal{
			Headline:    fmt.Sprintf("rewriting cue.mod to %s would strand self-imports of %s", target, cueModPath),
			Evidence:    ev,
			Consequence: "Repair never rewrites imports in a tree you own — that wholesale rewrite\nis correct only on a fresh clone (init-from-template).",
			Action:      "Update the listed imports to the target path first, then rerun.",
		}}
	}
	p.Actions = append(p.Actions, RepairAction{
		File: file, Field: "module", Current: cueModPath, New: target,
		apply: func() error {
			_, err := cueedit.SetCueModModule(dir, target)
			return err
		},
	})
	return nil
}

// planIdentity plans the identity half: realign a present ModulePath, or
// create the whole package when absent — sourcing the version from the
// tree's own metadata.version, never choosing one.
func planIdentity(ctx context.Context, k *kernel.Kernel, registry string, p *RepairPlan, idPath string, idErr error, target string) error {
	file := filepath.Join("identity", "identity.cue")
	dir := p.Dir

	if idErr == nil {
		if idPath == target || idPath == "" {
			// "" means the field exists but is not a rewritable literal;
			// vet's full conformance check owns that report.
			return nil
		}
		p.Actions = append(p.Actions, RepairAction{
			File: file, Field: "ModulePath", Current: idPath, New: target,
			apply: func() error {
				_, err := cueedit.SetIdentityModulePath(dir, target)
				return err
			},
		})
		return nil
	}

	if _, statErr := os.Stat(filepath.Join(dir, file)); statErr == nil {
		return fmt.Errorf("repair cannot rewrite a malformed %s: %w", file, idErr)
	}

	version, err := statedVersion(ctx, k, registry, dir)
	if err != nil {
		return err
	}
	content := identityFileContent(target, version)
	p.Actions = append(p.Actions, RepairAction{
		File: file, Create: true, Field: "ModulePath, Version", New: fmt.Sprintf("%s, %s", target, version),
		apply: func() error {
			if err := os.MkdirAll(filepath.Join(dir, "identity"), 0o755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644) //nolint:gosec // G306: repaired source files are project files, not secrets
		},
	})
	return nil
}

// statedVersion reads the version the tree itself states (metadata.version),
// the only source identity creation may use — init does not choose versions
// (D20).
func statedVersion(ctx context.Context, k *kernel.Kernel, registry, dir string) (string, error) {
	refuse := func(why string) error {
		return &RefusalError{publish.Refusal{
			Headline:    fmt.Sprintf("%s states no version init could adopt", dir),
			Evidence:    [][]string{{"looked at", "metadata.version", why}},
			Consequence: "Creating identity/identity.cue needs a Version, and init does not choose\none (D20) — that would move the invention upstream of the publish gates.",
			Action:      "State it in module.cue (metadata.version) or write identity/identity.cue\nyourself, then rerun.",
		}}
	}
	if k == nil {
		return "", refuse("tree not loaded")
	}
	val, err := k.LoadModulePackage(ctx, dir, loaderfile.LoadOptions{Registry: registry})
	if err != nil {
		return "", refuse(fmt.Sprintf("tree does not load: %v", err))
	}
	version, err := val.LookupPath(cue.ParsePath("metadata.version")).String()
	if err != nil || version == "" {
		return "", refuse("not stated")
	}
	if err := cueedit.CheckVersion(version); err != nil {
		return "", refuse(fmt.Sprintf("%q is not a bare SemVer", version))
	}
	return version, nil
}

// identityFileContent is the identity package repair creates — the same
// shape the templates carry, with the tree's own values.
func identityFileContent(modulePath, version string) string {
	return fmt.Sprintf(`// Package identity is the single source of this module's path and version
// (core #IdentityPackage, enhancements 0010 D38 / 0011 D12). It sits at the
// bottom of the module's import graph — no intra-module imports, no core
// import; validation is external (a publishing tool unifies this package
// against core's #IdentityPackage).
package identity

// ModulePath is the module's complete CUE module path, major suffix included
// — byte-identical to the module field in cue.mod/module.cue.
ModulePath: %q

// Version is the module's bare SemVer; its major must agree with ModulePath's.
// A plain literal: the kernel's loader gate requires a concrete value, and a
// defaulted disjunction is not one. Written by opm module version set.
Version: %q
`, modulePath, version)
}

// Apply runs every planned action in order.
func (p *RepairPlan) Apply() error {
	for _, a := range p.Actions {
		if err := a.apply(); err != nil {
			return fmt.Errorf("repairing %s: %w", a.File, err)
		}
	}
	return nil
}

// Describe renders the plan for the second confirmation: every file to be
// created or edited, and for an edit the current value beside its
// replacement — the author must have something to judge (D20).
func (p *RepairPlan) Describe() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Repair %s to %s:\n\n", p.Dir, p.ModulePath)
	for _, a := range p.Actions {
		if a.Create {
			fmt.Fprintf(&b, "  create  %-24s %s = %s\n", a.File, a.Field, a.New)
			continue
		}
		fmt.Fprintf(&b, "  edit    %-24s %s: %s -> %s\n", a.File, a.Field, a.Current, a.New)
	}
	return b.String()
}
