package render

import (
	"fmt"
	"path/filepath"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/pkg/loader"
)

// replacedByInstance is the kernel's By value on a replacement row the
// instance-side input supplied (kernel.Replacement: "platform" or
// "instance"). A module render's synthesized instance is that input, so a
// module directory's own replacements come back under it too.
const replacedByInstance = "instance"

// replacementWarnings words the D19 (enhancement 0010) render warnings from
// data: one line per replacement the kernel honored (rows, already in path
// order), naming the replaced path, the directory or module its bytes were
// served from and which input supplied it; then one line per `replaceWith`
// in the module context's own cue.mod/local-module.cue that no instance row
// carries — the platform names that path, so the module's redirect was inert
// and the platform's bytes rendered. Platform-side entries are only ever
// reported through rows: the platform is authoritative, nothing of its file
// can be inert. A clean context yields nothing.
//
// moduleRoot is the effective module context (the instance file's module
// root, or the module directory); "" reads no file. A local file the reader
// cannot parse yields no inert entries: the kernel already refused such a
// render, so the builder never sees one in practice.
func replacementWarnings(rows []kernel.Replacement, moduleRoot string) []string {
	warnings := make([]string, 0, len(rows))
	honored := make(map[string]bool, len(rows))
	for _, r := range rows {
		warnings = append(warnings, fmt.Sprintf(
			"local replacement in effect: %s served from %s (%s); rendered bytes may not correspond to any published build",
			r.Path, r.Target, r.By))
		if r.By == replacedByInstance {
			honored[r.Path] = true
		}
	}
	entries, err := loader.LocalReplacements(moduleRoot)
	if err != nil {
		return warnings
	}
	localFile := filepath.Join(moduleRoot, "cue.mod", "local-module.cue")
	for _, e := range entries {
		if honored[e.Path] {
			continue
		}
		warnings = append(warnings, fmt.Sprintf(
			"local replacement of %s in %s is ignored: the platform names that path; redirect it in the platform module's cue.mod/local-module.cue",
			e.Path, localFile))
	}
	return warnings
}
