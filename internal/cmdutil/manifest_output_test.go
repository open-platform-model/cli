package cmdutil_test

import (
	"testing"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseManifestOutputFormat(t *testing.T) {
	format, err := cmdutil.ParseManifestOutputFormat("yaml")
	require.NoError(t, err)
	assert.Equal(t, output.FormatYAML, format)

	_, err = cmdutil.ParseManifestOutputFormat("table")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}
