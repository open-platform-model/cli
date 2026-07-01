package cmdutil_test

import (
	"testing"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/output"
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
