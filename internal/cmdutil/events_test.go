package cmdutil

import (
	"testing"

	"github.com/opmodel/cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEventsOptions(t *testing.T) {
	opts, err := ParseEventsOptions("1h", "Warning", "json", false)
	require.NoError(t, err)
	assert.Equal(t, "Warning", opts.EventType)
	assert.Equal(t, output.FormatJSON, opts.OutputFormat)

	_, err = ParseEventsOptions("1h", "Warning", "json", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "watch mode only supports table output")

	_, err = ParseEventsOptions("1h", "Error", "table", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --type")
}
