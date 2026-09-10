package loader

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"cuelang.org/go/mod/modfile"
)

// ModuleRootFrom walks up from startDir to find the CUE module root — the
// nearest ancestor directory containing cue.mod/module.cue. Returns "" when no
// module root is found.
func ModuleRootFrom(startDir string) string {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "cue.mod", "module.cue")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// LocalReplacement is one `replaceWith` entry of a module root's
// cue.mod/local-module.cue: the major-qualified module path it redirects and
// the target as written — a directory (relative to the module root, or
// absolute) or another module path.
type LocalReplacement struct {
	Path        string
	ReplaceWith string
}

// LocalReplacements reads the `replaceWith` entries of the module rooted at
// moduleRoot, parsed the way cue/load and the library kernel read the file
// (modfile.ParseLocal against the root's cue.mod/module.cue), so the CLI and
// the kernel agree on what counts as a replacement. An absent local file, the
// normal case, yields no entries and no error. A module.cue or local file
// that cannot be read or does not parse is an error naming the file. Entries
// are sorted by path.
func LocalReplacements(moduleRoot string) ([]LocalReplacement, error) {
	if moduleRoot == "" {
		return nil, nil
	}
	localPath := filepath.Join(moduleRoot, "cue.mod", "local-module.cue")
	data, err := os.ReadFile(localPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", localPath, err)
	}

	modPath := filepath.Join(moduleRoot, "cue.mod", "module.cue")
	modData, err := os.ReadFile(modPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", modPath, err)
	}
	base, err := modfile.Parse(modData, modPath)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", modPath, err)
	}
	eff, err := modfile.ParseLocal(data, localPath, base)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", localPath, err)
	}

	var out []LocalReplacement
	for path, dep := range eff.Deps {
		if dep == nil || dep.ReplaceWith == "" {
			continue
		}
		out = append(out, LocalReplacement{Path: path, ReplaceWith: dep.ReplaceWith})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// HasLocalModuleReplacement reports whether the module rooted at moduleRoot
// carries a cue.mod/local-module.cue with at least one `replaceWith` entry.
//
// This is the render-provenance signal (enhancement 0006 D7): when the main
// module's local-module.cue replaces any dependency, the rendered bytes did not
// come from pure registry resolution. It is deliberately conservative — any
// replacement (local directory or an alternative module@version fork) marks the
// render, since a replaced dependency also changes bytes, and a present file
// that cannot be read or parsed counts as a replacement too (it only exists
// to carry one). A missing or replacement-free file returns false.
func HasLocalModuleReplacement(moduleRoot string) bool {
	entries, err := LocalReplacements(moduleRoot)
	return err != nil || len(entries) > 0
}
