// Package cuemodtest is the hermetic harness for tests that tidy CUE
// modules: an in-memory registry with a small dependency, a cold CUE module
// cache, and a consumer module that imports the dependency.
package cuemodtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/require"
)

// UnreachableRegistry is a CUE_REGISTRY mapping that refuses connections
// immediately.
const UnreachableRegistry = "127.0.0.1:1+insecure"

// Coordinates the registry serves.
const (
	DepModule    = "example.com/dep@v0"
	DepOldest    = "v0.1.0"
	DepNewest    = "v0.2.0"
	UnusedModule = "example.com/unused@v0"
	UnusedLatest = "v0.1.0"
)

// UntidyModuleCue is a consumer manifest without a deps block: the
// consumer's import of DepModule is unpinned.
const UntidyModuleCue = `module: "test.example/consumer@v0"
language: version: "v0.9.0"
`

// ConsumerSource imports DepModule, so a tidy must pin it.
const ConsumerSource = `package consumer

import "example.com/dep@v0"

v: dep.version
`

// modules is what the registry serves: two versions of the dependency (so
// "newest" is observable) and an unrelated module a test can pin without
// importing.
var modules = fstest.MapFS{
	"example.com_dep_v0.1.0/cue.mod/module.cue": {Data: []byte(`module: "example.com/dep@v0"
language: version: "v0.9.0"
`)},
	"example.com_dep_v0.1.0/dep.cue": {Data: []byte(`package dep

version: "v0.1.0"
`)},
	"example.com_dep_v0.2.0/cue.mod/module.cue": {Data: []byte(`module: "example.com/dep@v0"
language: version: "v0.9.0"
`)},
	"example.com_dep_v0.2.0/dep.cue": {Data: []byte(`package dep

version: "v0.2.0"
`)},
	"example.com_unused_v0.1.0/cue.mod/module.cue": {Data: []byte(`module: "example.com/unused@v0"
language: version: "v0.9.0"
`)},
	"example.com_unused_v0.1.0/unused.cue": {Data: []byte(`package unused
`)},
}

// Registry starts the in-memory registry and returns its CUE_REGISTRY
// mapping. The mapping names no fallback registry, so any resolution that
// escapes it fails instead of reaching the network.
func Registry(t *testing.T) string {
	t.Helper()
	reg, err := modregistrytest.New(modules, "")
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return reg.Host() + "+insecure"
}

// ColdCache points CUE_CACHE_DIR at a fresh directory for the test. The
// cache extracts module files read-only, so t.TempDir's own cleanup would
// fail on them; this cleanup chmods before removing.
func ColdCache(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("", "opm-cuemod-cache-*")
	require.NoError(t, err)
	t.Setenv("CUE_CACHE_DIR", dir)
	t.Cleanup(func() {
		if err := makeWritable(dir); err != nil {
			t.Logf("making %s writable: %v", dir, err)
		}
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("removing %s: %v", dir, err)
		}
	})
}

// makeWritable chmods everything under dir to 0o700, scoped to dir.
func makeWritable(dir string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	return fs.WalkDir(root.FS(), ".", func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return root.Chmod(p, 0o700)
	})
}

// WriteConsumer makes dir a module root holding moduleCue (verbatim, so the
// test controls its deps) and ConsumerSource, and returns dir.
func WriteConsumer(t *testing.T, dir, moduleCue string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(moduleCue), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "consumer.cue"), []byte(ConsumerSource), 0o600))
	return dir
}

// NewConsumer returns an untidy consumer in a fresh temp dir, with a cold
// cache and a live registry.
func NewConsumer(t *testing.T) (dir, registry string) {
	t.Helper()
	ColdCache(t)
	return WriteConsumer(t, t.TempDir(), UntidyModuleCue), Registry(t)
}

// ReadModule returns dir's cue.mod/module.cue.
func ReadModule(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	return string(data)
}

// Backdate sets dir's cue.mod/module.cue mtime to a fixed past instant and
// returns it, so "unchanged mtime" is observable regardless of filesystem
// timestamp granularity.
func Backdate(t *testing.T, dir string) time.Time {
	t.Helper()
	past := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "cue.mod", "module.cue"), past, past))
	return past
}

// ModTime returns dir's cue.mod/module.cue mtime.
func ModTime(t *testing.T, dir string) time.Time {
	t.Helper()
	info, err := os.Stat(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	return info.ModTime()
}
