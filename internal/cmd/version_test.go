// Package cmd provides CLI command implementations.
package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/version"
)

func TestNewVersionCmd(t *testing.T) {
	cmd := NewVersionCmd(nil)

	assert.Equal(t, "version", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestVersionCmd_Execute(t *testing.T) {
	cmd := NewVersionCmd(nil)

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	// Note: output.Println writes to stdout, not cmd.SetOut()
	// We just verify the command executes without error
	err := cmd.Execute()
	assert.NoError(t, err)
}

// runVersionCmd runs 'opm version' with args and returns its standard output.
func runVersionCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	// Through the root, as a user runs it: the root silences cobra's usage text.
	cmd := NewRootCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(append([]string{"version"}, args...))
	err := cmd.Execute()
	return stdout.String(), err
}

func TestVersionCmd_DefaultIsTheTextForm(t *testing.T) {
	for _, args := range [][]string{nil, {"-o", "text"}, {"--output", "text"}} {
		out, err := runVersionCmd(t, args...)
		require.NoError(t, err)
		assert.Equal(t, version.Get().String()+"\n", out)
	}
}

func TestVersionCmd_OutputJSON(t *testing.T) {
	out, err := runVersionCmd(t, "--output", "json")
	require.NoError(t, err)

	var got map[string]string
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	info := version.Get()
	assert.Equal(t, map[string]string{
		"version":       info.Version,
		"gitCommit":     info.GitCommit,
		"buildDate":     info.BuildDate,
		"goVersion":     info.GoVersion,
		"cueSDKVersion": info.CUESDKVersion,
	}, got)
	assert.True(t, bytes.HasSuffix([]byte(out), []byte("}\n")), "the object ends with one newline")
}

func TestVersionCmd_OutputYAML(t *testing.T) {
	out, err := runVersionCmd(t, "-o", "yaml")
	require.NoError(t, err)

	var got map[string]string
	require.NoError(t, yaml.Unmarshal([]byte(out), &got))
	assert.Equal(t, version.Get().Version, got["version"])
	assert.Len(t, got, 5)
	assert.Contains(t, got, "cueSDKVersion")
}

func TestVersionCmd_InvalidOutputFormat(t *testing.T) {
	out, err := runVersionCmd(t, "-o", "toml")
	require.Error(t, err)
	assert.Equal(t, `invalid output format "toml" (valid: text, json, yaml)`, err.Error())
	assert.Empty(t, out, "nothing is printed on standard output")

	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
}
