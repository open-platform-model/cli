package cmdref

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLong_TabIndentedSourceSplitsProsePreformattedAndExamples(t *testing.T) {
	long := "Publish a thing to its registry, at the coordinates it\ndeclares.\n\n" +
		"\tThe pipeline reads identity/identity.cue and\n\tpushes.\n\n" +
		"\tArguments:\n\t  path    Path to the directory\n\n" +
		"\tExamples:\n\t  # First\n\t  opm thing publish\n\n\t  # Second\n\t  opm thing publish ./src"

	p := parseLong(long)

	require.Len(t, p.desc, 4)
	assert.Equal(t, block{lines: []string{"Publish a thing to its registry, at the coordinates it declares."}}, p.desc[0])
	assert.Equal(t, block{lines: []string{"The pipeline reads identity/identity.cue and pushes."}}, p.desc[1])
	assert.Equal(t, block{lines: []string{"Arguments:"}}, p.desc[2])
	assert.Equal(t, block{pre: true, lines: []string{"path    Path to the directory"}}, p.desc[3])
	assert.Equal(t, []string{"# First", "opm thing publish", "", "# Second", "opm thing publish ./src"}, p.examples)
}

func TestParseLong_UnindentedSourceKeepsNestedIndentation(t *testing.T) {
	long := "Create a package.\n\nThe package holds:\n\n  a.cue   first\n          continued\n  b.cue   second\n\nExit codes: 0 written."

	p := parseLong(long)

	require.Len(t, p.desc, 4)
	assert.Equal(t, []string{"The package holds:"}, p.desc[1].lines)
	assert.Equal(t, block{pre: true, lines: []string{"a.cue   first", "        continued", "b.cue   second"}}, p.desc[2])
	assert.Equal(t, []string{"Exit codes: 0 written."}, p.desc[3].lines)
	assert.Empty(t, p.examples)
}

func TestParseLong_ProseAfterExamplesReturnsToTheDescription(t *testing.T) {
	p := parseLong("Run it.\n\nExamples:\n  opm run\n\nSee also the guide.")

	assert.Equal(t, []string{"opm run"}, p.examples)
	require.Len(t, p.desc, 2)
	assert.Equal(t, []string{"See also the guide."}, p.desc[1].lines)
}

func TestParseLong_Empty(t *testing.T) {
	assert.Equal(t, parsedLong{}, parseLong(""))
}

func TestFormatProse(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"flags and env vars", "Use --registry, then OPM_REGISTRY.", "Use `--registry`, then `OPM_REGISTRY`."},
		{"paths and placeholders", "Pass --platform <dir> or ~/.opm/platform/.", "Pass `--platform` `<dir>` or `~/.opm/platform/`."},
		{"quoted command", "run 'opm module vet' first", "run `opm module vet` first"},
		{"apostrophes stay prose", "the cluster's Platform", "the cluster's Platform"},
		{"word pairs stay prose", "every beta/GA member", "every beta/GA member"},
		{"files", "next to values.cue and identity/identity.cue.", "next to `values.cue` and `identity/identity.cue`."},
		{"definitions", "a #ModuleInstance around it", "a `#ModuleInstance` around it"},
		{"quoted code word drops its quotes", `defaults to "<module>-debug".`, "defaults to `<module>-debug`."},
		{"lone operators are escaped", "[dir] > --platform", "`[dir]` &gt; `--platform`"},
		{"markdown characters are escaped", "#### Linux: 50% & more", `\#\#\#\# Linux: 50% &amp; more`},
		{"list marker at the start", "- not a list", `\- not a list`},
		{"ordinal at the start", "1. not a list", `1\. not a list`},
		{"whitespace collapses", "  a \t b  ", "a b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatProse(tt.in))
		})
	}
}

func TestFence_LengthensPastBacktickRuns(t *testing.T) {
	assert.Equal(t, "````text\n```go\n````\n", fence("text", []string{"```go"}))
	assert.Equal(t, "```sh\nopm\n```\n", fence("sh", []string{"opm"}))
}

func TestFenceLang(t *testing.T) {
	assert.Equal(t, "sh", fenceLang([]string{"# comment", "opm module build", ""}))
	assert.Equal(t, "text", fenceLang([]string{"source <(opm completion bash)"}))
	assert.Equal(t, "text", fenceLang([]string{"# only a comment"}))
}

func TestEscapeShortcodes(t *testing.T) {
	assert.Equal(t, "{{</* opm/x */>}} {{%/* y */%}}", escapeShortcodes("{{< opm/x >}} {{% y %}}"))
}
