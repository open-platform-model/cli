package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// A long description is a raw string, so a line indented with the tabs of
// the surrounding Go code prints indented. Paragraphs start in the first
// column and examples are indented with spaces.
func TestHelpTextHasNoTabIndentedLines(t *testing.T) {
	walkCommands(NewRootCmd(), func(c *cobra.Command) {
		for i, line := range strings.Split(c.Long, "\n") {
			assert.False(t, strings.HasPrefix(line, "\t"),
				"%s: line %d of the long description begins with a tab: %q", c.CommandPath(), i+1, line)
		}
	})
}

func TestRootHelpDocumentsTheExitCodes(t *testing.T) {
	long := NewRootCmd().Long
	assert.Contains(t, long, "Exit codes:")
	for _, row := range []string{
		"0  success",
		"1  general error, usage errors included",
		"2  validation error or refusal",
		"3  connectivity error",
		"4  permission denied",
		"5  not found",
	} {
		assert.Contains(t, long, row)
	}
}
