package instinit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cuemod"
	"github.com/open-platform-model/cli/internal/publish"
)

func testFiles() Files {
	return Files{
		ModuleFile:   []byte("module: \"instance.local/web@v0\"\n"),
		InstanceFile: []byte("package instance\n"),
		ValuesFile:   []byte("package instance\n\nvalues: {}\n"),
	}
}

func tidyOK(_ context.Context, _ string, _ cuemod.TidyOptions) (cuemod.TidyResult, error) {
	return cuemod.TidyResult{ModuleUpdated: true}, nil
}

func tidyFails(err error) func(context.Context, string, cuemod.TidyOptions) (cuemod.TidyResult, error) {
	return func(context.Context, string, cuemod.TidyOptions) (cuemod.TidyResult, error) {
		return cuemod.TidyResult{}, err
	}
}

// entries lists the names in dir.
func entries(t *testing.T, dir string) []string {
	t.Helper()
	des, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(des))
	for _, de := range des {
		names = append(names, de.Name())
	}
	return names
}

func TestWrite_SuccessLeavesOnlyTarget(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "web")
	var tidiedIn string
	ops := writeOps{
		tidy: func(_ context.Context, dir string, opts cuemod.TidyOptions) (cuemod.TidyResult, error) {
			tidiedIn = dir
			assert.Equal(t, "reg", opts.Registry)
			assert.False(t, opts.Check)
			return cuemod.TidyResult{}, nil
		},
		rename: os.Rename,
	}

	require.NoError(t, write(context.Background(), target, testFiles(), "reg", ops))

	assert.Equal(t, []string{"web"}, entries(t, parent))
	assert.NotEqual(t, target, tidiedIn, "tidy runs on the staging directory, not the target")
	assert.Equal(t, parent, filepath.Dir(tidiedIn), "staging is a sibling of the target")
	for name, want := range testFiles() {
		got, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(name)))
		require.NoError(t, err)
		assert.Equal(t, string(want), string(got))
	}
	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}

func TestWrite_FailureLeavesNothing(t *testing.T) {
	for name, ops := range map[string]writeOps{
		"tidy fails":   {tidy: tidyFails(errors.New("cannot find module providing package x")), rename: os.Rename},
		"rename fails": {tidy: tidyOK, rename: func(string, string) error { return errors.New("rename refused") }},
	} {
		t.Run(name, func(t *testing.T) {
			parent := t.TempDir()
			target := filepath.Join(parent, "web")

			err := write(context.Background(), target, testFiles(), "reg", ops)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "initializing "+target)
			var connErr *publish.ConnectivityError
			assert.False(t, errors.As(err, &connErr))
			assert.Empty(t, entries(t, parent), "neither the target nor a staging directory may remain")
		})
	}
}

func TestWrite_TidyConnectivityFailure(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "web")
	cause := errors.New(`cannot fetch opmodel.dev/core@v2.0.0: cannot do HTTP request: Get "http://127.0.0.1:1/": dial tcp: connection refused`)
	ops := writeOps{tidy: tidyFails(cause), rename: os.Rename}

	err := write(context.Background(), target, testFiles(), "reg", ops)
	var connErr *publish.ConnectivityError
	require.ErrorAs(t, err, &connErr)
	assert.Contains(t, connErr.Op, "resolving the dependencies of "+target)
	assert.Contains(t, connErr.Op, "(registry reg)")
	assert.ErrorIs(t, err, cause)
	assert.Empty(t, entries(t, parent))
}

func TestWrite_TargetAppearedMeanwhile(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "web")
	ops := writeOps{
		tidy: func(context.Context, string, cuemod.TidyOptions) (cuemod.TidyResult, error) {
			return cuemod.TidyResult{}, os.Mkdir(target, 0o755)
		},
		rename: os.Rename,
	}

	err := write(context.Background(), target, testFiles(), "reg", ops)
	require.Error(t, err)
	assert.Equal(t, []string{"web"}, entries(t, parent), "only the directory someone else created remains")
	assert.Empty(t, entries(t, target), "the other directory is left untouched")
}
