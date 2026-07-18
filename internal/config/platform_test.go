// Package config provides configuration loading and management.
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writePlatform writes content as platform.cue in a fresh temp dir and
// returns its path.
func writePlatform(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "platform.cue")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestPlatformFilePath_SiblingOfConfig(t *testing.T) {
	dir := filepath.Join("custom", "dir")
	got := PlatformFilePath(filepath.Join(dir, "config.cue"))
	assert.Equal(t, filepath.Join(dir, "platform.cue"), got)
}

func TestValidatePlatformFile_DefaultTemplateIsValid(t *testing.T) {
	// The template written by `opm config init` must validate cleanly.
	path := writePlatform(t, DefaultPlatformTemplate)
	require.NoError(t, ValidatePlatformFile(path))
}

func TestValidatePlatformFile_MinimalValid(t *testing.T) {
	path := writePlatform(t, `name: "cluster"
type: "kubernetes"
`)
	require.NoError(t, ValidatePlatformFile(path))
}

func TestValidatePlatformFile_MissingFile(t *testing.T) {
	err := ValidatePlatformFile("/nonexistent/platform.cue")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err), "missing file should surface as os.IsNotExist")
}

func TestValidatePlatformFile_ImportsRejected(t *testing.T) {
	// Data-only: any CUE import declaration is rejected (0006 D39).
	path := writePlatform(t, `import "strings"

name: strings.ToLower("Cluster")
type: "kubernetes"
`)
	err := ValidatePlatformFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data-only")
}

func TestValidatePlatformFile_MissingRequiredFields(t *testing.T) {
	path := writePlatform(t, `registry: {
	"opmodel.dev/catalogs/opm": {}
}
`)
	err := ValidatePlatformFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform schema validation failed")
}

func TestValidatePlatformFile_InvalidSubscriptionShape(t *testing.T) {
	path := writePlatform(t, `name: "cluster"
type: "kubernetes"
registry: {
	"opmodel.dev/catalogs/opm": {
		filter: {
			range: 42
		}
	}
}
`)
	err := ValidatePlatformFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform schema validation failed")
}

func TestValidatePlatformFile_UnknownFieldRejected(t *testing.T) {
	path := writePlatform(t, `name: "cluster"
type: "kubernetes"
bogus: true
`)
	err := ValidatePlatformFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform schema validation failed")
}

func TestValidatePlatformFile_SyntaxError(t *testing.T) {
	path := writePlatform(t, `name: "cluster
`)
	err := ValidatePlatformFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform file error")
}
