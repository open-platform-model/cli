package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_VersionSkipsConfigLoad(t *testing.T) {
	tmpDir := t.TempDir()
	badConfig := filepath.Join(tmpDir, "bad-config.cue")
	require.NoError(t, os.WriteFile(badConfig, []byte("this is not valid CUE !!!"), 0o600))

	origConfig := os.Getenv("OPM_CONFIG")
	require.NoError(t, os.Setenv("OPM_CONFIG", badConfig))
	defer func() {
		if origConfig == "" {
			_ = os.Unsetenv("OPM_CONFIG")
			return
		}
		_ = os.Setenv("OPM_CONFIG", origConfig)
	}()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestRootCmd_ConfigInitSkipsConfigLoad(t *testing.T) {
	tmpHome := t.TempDir()
	badConfig := filepath.Join(tmpHome, "bad-config.cue")
	require.NoError(t, os.WriteFile(badConfig, []byte("this is not valid CUE !!!"), 0o600))

	origConfig := os.Getenv("OPM_CONFIG")
	origHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("OPM_CONFIG", badConfig))
	require.NoError(t, os.Setenv("HOME", tmpHome))
	defer func() {
		if origConfig == "" {
			_ = os.Unsetenv("OPM_CONFIG")
		} else {
			_ = os.Setenv("OPM_CONFIG", origConfig)
		}
		if origHome == "" {
			_ = os.Unsetenv("HOME")
		} else {
			_ = os.Setenv("HOME", origHome)
		}
	}()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"config", "init"})

	err := cmd.Execute()
	assert.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(tmpHome, ".opm", "config.cue"))
	assert.NoError(t, statErr)
}

func TestRootCmd_ConfigVetSkipsConfigLoad(t *testing.T) {
	tmpHome := t.TempDir()
	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(filepath.Join(opmDir, "cue.mod"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(opmDir, "cue.mod", "module.cue"), []byte(`module: "test.example.com/opm@v0"
language: version: "v0.15.0"
`), 0o600))
	badConfig := filepath.Join(opmDir, "config.cue")
	require.NoError(t, os.WriteFile(badConfig, []byte("this is not valid CUE !!!"), 0o600))

	origConfig := os.Getenv("OPM_CONFIG")
	origHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("OPM_CONFIG", badConfig))
	require.NoError(t, os.Setenv("HOME", tmpHome))
	defer func() {
		if origConfig == "" {
			_ = os.Unsetenv("OPM_CONFIG")
		} else {
			_ = os.Setenv("OPM_CONFIG", origConfig)
		}
		if origHome == "" {
			_ = os.Unsetenv("HOME")
		} else {
			_ = os.Setenv("HOME", origHome)
		}
	}()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"config", "vet"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "configuration error:")
}
