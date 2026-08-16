package cmdutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cueedit"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/publish"
)

// writeVersionSetIdentity creates dir/identity/identity.cue with the given
// content and returns dir.
func writeVersionSetIdentity(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "identity"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "identity", "identity.cue"), []byte(content), 0o644))
	return dir
}

func TestRunVersionSet_SetsDeclaredVersion(t *testing.T) {
	dir := writeVersionSetIdentity(t, `package identity

ModulePath: "opmodel.dev/modules/web_app@v1"
Version:    "1.2.0"
`)

	require.NoError(t, RunVersionSet(publish.KindModule, "1.3.0", []string{dir}))

	data, err := os.ReadFile(filepath.Join(dir, "identity", "identity.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `Version:    "1.3.0"`)
}

func TestRunVersionSet_NoOpLeavesFileUntouched(t *testing.T) {
	content := `package identity

ModulePath: "opmodel.dev/modules/web_app@v1"
Version:    "1.3.0"
`
	dir := writeVersionSetIdentity(t, content)
	path := filepath.Join(dir, "identity", "identity.cue")
	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(path, past, past))
	before, err := os.Stat(path)
	require.NoError(t, err)

	require.NoError(t, RunVersionSet(publish.KindModule, "1.3.0", []string{dir}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, before.ModTime(), after.ModTime())
}

func TestRunVersionSet_OpenFieldGainsVersion(t *testing.T) {
	dir := writeVersionSetIdentity(t, `package identity

#VersionType: string

ModulePath: "opmodel.dev/catalogs/opm@v2"
Version:    #VersionType
`)

	require.NoError(t, RunVersionSet(publish.KindCatalog, "2.0.0", []string{dir}))

	data, err := os.ReadFile(filepath.Join(dir, "identity", "identity.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `Version:    #VersionType & "2.0.0"`)
}

func TestRunVersionSet_ShapeRefusalExitsValidation(t *testing.T) {
	dir := writeVersionSetIdentity(t, `package identity

ModulePath: "opmodel.dev/modules/web_app@v1"
`)

	err := RunVersionSet(publish.KindModule, "1.0.0", []string{dir})
	require.Error(t, err)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.True(t, exitErr.Printed)
	assert.True(t, errors.Is(err, cueedit.ErrIdentityShape))
}

func TestRunVersionSet_InvalidSemverExitsValidation(t *testing.T) {
	dir := writeVersionSetIdentity(t, `package identity

Version: "1.0.0"
`)

	err := RunVersionSet(publish.KindModule, "v1.0.0", []string{dir})
	require.Error(t, err)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Contains(t, err.Error(), "SemVer")

	// The invalid argument was rejected before any read or write.
	data, readErr := os.ReadFile(filepath.Join(dir, "identity", "identity.cue"))
	require.NoError(t, readErr)
	assert.Contains(t, string(data), `"1.0.0"`)
}
