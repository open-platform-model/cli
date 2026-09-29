package modref

import (
	"fmt"
	"strings"
)

// Report renders a resolution as the lines a command prints on standard
// error before it renders (0016:D5:R3): one selection line, then one line per
// higher major the walk skipped.
func Report(r *Resolution) []string {
	var how string
	switch r.Strategy {
	case Exact:
		how = "pinned"
	case FloatMajor:
		how = "newest in " + r.Major
	case HighestCompatibleMajor:
		how = "highest major on core " + r.CoreMajor
	}
	lines := make([]string, 0, 1+len(r.Skipped))
	lines = append(lines, fmt.Sprintf("Resolved %s -> %s %s (%s)", r.Path, r.Major, strings.TrimPrefix(r.Version, "v"), how))
	for _, s := range r.Skipped {
		lines = append(lines, fmt.Sprintf("  skipped %s: %s", s.Major, s.Reason))
	}
	return lines
}
