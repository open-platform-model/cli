// Package cmdutiltest holds fixtures shared by the tests of the command
// helpers and the render workflow.
package cmdutiltest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// MinimalModule is a module package body that passes the kernel's module
// shape gate without importing core, so no registry is needed.
const MinimalModule = `package demo

kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v0"
	version:    "0.1.0"
}
`

// WriteMinimalModule writes a module package that needs no registry, a
// cue.mod without dependencies and MinimalModule as module.cue, into a temp
// directory and returns it.
func WriteMinimalModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"),
		[]byte("module: \"example.com/modules/demo@v0\"\nlanguage: version: \"v0.17.0\"\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(MinimalModule), 0o600))
	return dir
}
