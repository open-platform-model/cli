package cmdutil

import (
	"testing"

	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseManifestOutputFormat(t *testing.T) {
	format, err := ParseManifestOutputFormat("yaml")
	require.NoError(t, err)
	assert.Equal(t, output.FormatYAML, format)

	_, err = ParseManifestOutputFormat("table")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestFormatApplySummary(t *testing.T) {
	summary := FormatApplySummary(&kubernetes.ApplyResult{Applied: 5, Created: 2, Configured: 1, Unchanged: 2})
	assert.Equal(t, "applied 5 resources successfully (2 created, 1 configured, 2 unchanged)", summary)
}
