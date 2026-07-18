package loader

import (
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
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

// HasLocalModuleReplacement reports whether the module rooted at moduleRoot
// carries a cue.mod/local-module.cue with at least one `replaceWith` entry.
//
// This is the render-provenance signal (enhancement 0006 D7): when the main
// module's local-module.cue replaces any dependency, the rendered bytes did not
// come from pure registry resolution. It is deliberately conservative — any
// replacement (local directory or an alternative module@version fork) marks the
// render, since a replaced dependency also changes bytes. A missing,
// unreadable, or replacement-free file returns false.
func HasLocalModuleReplacement(moduleRoot string) bool {
	if moduleRoot == "" {
		return false
	}

	data, err := os.ReadFile(filepath.Join(moduleRoot, "cue.mod", "local-module.cue"))
	if err != nil {
		return false // absent (first-class) or unreadable → treat as registry
	}

	v := cuecontext.New().CompileBytes(data)
	if v.Err() != nil {
		// A present-but-unparseable local-module.cue only exists to carry
		// replacements; conservatively treat it as a local render.
		return true
	}

	deps := v.LookupPath(cue.ParsePath("deps"))
	if !deps.Exists() {
		return false
	}
	iter, err := deps.Fields()
	if err != nil {
		return true
	}
	for iter.Next() {
		if iter.Value().LookupPath(cue.ParsePath("replaceWith")).Exists() {
			return true
		}
	}
	return false
}
