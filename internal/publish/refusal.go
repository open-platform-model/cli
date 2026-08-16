package publish

import (
	"strings"

	"github.com/open-platform-model/cli/internal/output"
)

// Refusal is one gate's refusal in the fixed shape condition → evidence →
// consequence → action (0011, 06-operational.md). Disagreements print both
// values in aligned columns naming the file that declares each, and the
// action is a runnable command, not a description.
type Refusal struct {
	// Headline states the condition — the funnel prefixes "error:".
	Headline string

	// Evidence rows render as aligned columns (label, value, declaring
	// file:line, …). Nil when the refusal carries a CUE error instead.
	Evidence [][]string

	// Consequence states what would have been published and what a consumer
	// would have resolved. Skipped only where genuinely self-evident.
	Consequence string

	// Action is the runnable command that clears the condition.
	Action string

	// Err carries CUE's own error for refusals whose diagnostic is CUE's
	// (refusal 10, load failures) — surfaced through the grouped funnel,
	// which preserves file:line. When set, the fields above frame it.
	Err error
}

// Details renders evidence, consequence, and action as the refusal's detail
// block, indented for the Details funnel.
func (r Refusal) Details() string {
	var parts []string
	if len(r.Evidence) > 0 {
		parts = append(parts, output.AlignColumns("  ", r.Evidence))
	}
	if r.Consequence != "" {
		parts = append(parts, indent("  ", r.Consequence))
	}
	if r.Action != "" {
		parts = append(parts, indent("  ", r.Action))
	}
	return strings.Join(parts, "\n\n")
}

// indent prefixes every line of s with the given indent.
func indent(prefix, s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}
