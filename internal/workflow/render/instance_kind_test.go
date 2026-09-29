package render

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	liberrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/kernel"
)

// writeMinimalModule writes a module package that needs no registry: a
// cue.mod without dependencies and a package declaring kind "Module".
func writeMinimalModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"),
		[]byte("module: \"example.com/modules/demo@v0\"\nlanguage: version: \"v0.17.0\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"),
		[]byte("package demo\n\nkind: \"Module\"\n"), 0o644))
	return dir
}

// Instance acquisition decides by package kind: a module package handed to
// AcquireInstanceFromDir fails the shape gate with ErrWrongKind, which is
// what `opm instance build` keys its module refusal on.
func TestAcquireInstanceFromDir_ModulePackageIsWrongKind(t *testing.T) {
	k := kernel.New(kernel.WithRegistry("127.0.0.1:1+insecure"))
	_, err := k.AcquireInstanceFromDir(context.Background(), writeMinimalModule(t))
	require.ErrorIs(t, err, liberrors.ErrWrongKind)
}
