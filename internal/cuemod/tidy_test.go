package cuemod

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests in this package mutate the process working directory and
// CUE_REGISTRY through Tidy; none of them may call t.Parallel.

// unreachableRegistry refuses connections immediately.
const unreachableRegistry = "127.0.0.1:1+insecure"

// registryModules is what the in-memory registry serves: two versions of
// the dependency (so "newest" is observable) and an unrelated module the
// prune test pins without importing.
var registryModules = fstest.MapFS{
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

// consumerSource imports the dependency, so a tidy must pin it.
const consumerSource = `package consumer

import "example.com/dep@v0"

v: dep.version
`

// testRegistry starts the in-memory registry and returns its CUE_REGISTRY
// mapping. The mapping names no fallback registry, so any resolution that
// escapes it fails instead of reaching the network.
func testRegistry(t *testing.T) string {
	t.Helper()
	reg, err := modregistrytest.New(registryModules, "")
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return reg.Host() + "+insecure"
}

// coldCUECache points CUE_CACHE_DIR at a fresh directory. The cache
// extracts module files read-only, so t.TempDir's own cleanup would fail on
// them; this cleanup chmods before removing.
func coldCUECache(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("", "opm-cuemod-cache-*")
	require.NoError(t, err)
	t.Setenv("CUE_CACHE_DIR", dir)
	t.Cleanup(func() {
		_ = filepath.WalkDir(dir, func(p string, _ fs.DirEntry, err error) error {
			if err == nil {
				_ = os.Chmod(p, 0o700)
			}
			return nil
		})
		_ = os.RemoveAll(dir)
	})
}

// writeConsumer writes a module root holding module.cue (verbatim, so a
// test controls its deps) and consumerSource.
func writeConsumer(t *testing.T, dir, moduleCue string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(moduleCue), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "consumer.cue"), []byte(consumerSource), 0o600))
	return dir
}

const untidyModule = `module: "test.example/consumer@v0"
language: version: "v0.9.0"
`

// newConsumer returns an untidy consumer (its import has no deps entry)
// with a cold cache and a live test registry.
func newConsumer(t *testing.T) (dir, registry string) {
	t.Helper()
	coldCUECache(t)
	return writeConsumer(t, t.TempDir(), untidyModule), testRegistry(t)
}

func readModule(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	return string(data)
}

// backdate sets module.cue's mtime to a fixed past instant, so "unchanged
// mtime" is observable regardless of filesystem timestamp granularity.
func backdate(t *testing.T, dir string) time.Time {
	t.Helper()
	past := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "cue.mod", "module.cue"), past, past))
	return past
}

func mtime(t *testing.T, dir string) time.Time {
	t.Helper()
	info, err := os.Stat(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	return info.ModTime()
}

func TestTidy_HermeticHarness(t *testing.T) {
	dir, registry := newConsumer(t)

	res, err := Tidy(context.Background(), dir, TidyOptions{Registry: registry})
	require.NoError(t, err)
	assert.True(t, res.ModuleUpdated)
	assert.False(t, res.LocalUpdated)
	assert.Contains(t, readModule(t, dir), `"example.com/dep@v0"`)
}

func TestTidy_RepeatedCallsInOneProcess(t *testing.T) {
	dir, registry := newConsumer(t)
	ctx := context.Background()

	first, err := Tidy(ctx, dir, TidyOptions{Registry: registry})
	require.NoError(t, err)
	assert.True(t, first.ModuleUpdated)

	second, err := Tidy(ctx, dir, TidyOptions{Registry: registry})
	require.NoError(t, err, "a second cuecmd.New/Run in the same process must work")
	assert.Equal(t, TidyResult{}, second)

	require.NoError(t, checkTidy(ctx, t, dir, registry))
}

// checkTidy is a test shorthand for a --check tidy.
func checkTidy(ctx context.Context, t *testing.T, dir, registry string) error {
	t.Helper()
	_, err := Tidy(ctx, dir, TidyOptions{Registry: registry, Check: true})
	return err
}

func TestTidy_RegistryOptionWinsOverAmbientEnv(t *testing.T) {
	dir, registry := newConsumer(t)
	// The ambient mapping is dead: success proves cmd/cue read
	// opts.Registry, which Tidy sets after New and before Run.
	t.Setenv("CUE_REGISTRY", unreachableRegistry)

	_, err := Tidy(context.Background(), dir, TidyOptions{Registry: registry})
	require.NoError(t, err)
	assert.Equal(t, unreachableRegistry, os.Getenv("CUE_REGISTRY"))
}

func TestTidy_NothingPrintedOnSuccess(t *testing.T) {
	dir, registry := newConsumer(t)
	root, err := filepath.Abs(dir)
	require.NoError(t, err)

	out, err := runModTidy(context.Background(), root, TidyOptions{Registry: registry})
	require.NoError(t, err)
	assert.Empty(t, out)

	out, err = runModTidy(context.Background(), root, TidyOptions{Registry: registry, Check: true})
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestTidy_AddsMissingDependencyAtNewestVersion(t *testing.T) {
	dir, registry := newConsumer(t)

	res, err := Tidy(context.Background(), dir, TidyOptions{Registry: registry})
	require.NoError(t, err)
	assert.Equal(t, TidyResult{ModuleUpdated: true}, res)

	got := readModule(t, dir)
	assert.Contains(t, got, `"example.com/dep@v0"`)
	assert.Contains(t, got, `"v0.2.0"`)
	assert.NotContains(t, got, `"v0.1.0"`)
}

func TestTidy_PrunesUnusedDependency(t *testing.T) {
	coldCUECache(t)
	registry := testRegistry(t)
	dir := writeConsumer(t, t.TempDir(), `module: "test.example/consumer@v0"
language: version: "v0.9.0"
deps: {
	"example.com/dep@v0": v: "v0.1.0"
	"example.com/unused@v0": v: "v0.1.0"
}
`)

	res, err := Tidy(context.Background(), dir, TidyOptions{Registry: registry})
	require.NoError(t, err)
	assert.True(t, res.ModuleUpdated)

	got := readModule(t, dir)
	assert.NotContains(t, got, "example.com/unused")
	// Minimum version selection keeps the existing pin; tidy does not
	// upgrade a dependency that is already satisfied.
	assert.Contains(t, got, `"v0.1.0"`)
}

func TestTidy_AlreadyTidyWritesNothing(t *testing.T) {
	dir, registry := newConsumer(t)
	ctx := context.Background()
	_, err := Tidy(ctx, dir, TidyOptions{Registry: registry})
	require.NoError(t, err)
	before := readModule(t, dir)
	past := backdate(t, dir)

	res, err := Tidy(ctx, dir, TidyOptions{Registry: registry})
	require.NoError(t, err)
	assert.Equal(t, TidyResult{}, res)
	assert.Equal(t, before, readModule(t, dir))
	assert.True(t, mtime(t, dir).Equal(past), "an already tidy module.cue must not be rewritten")
}

func TestTidy_CheckOnUntidyModule(t *testing.T) {
	dir, registry := newConsumer(t)
	past := backdate(t, dir)

	err := checkTidy(context.Background(), t, dir, registry)

	var notTidy *NotTidyError
	require.ErrorAs(t, err, &notTidy)
	assert.Contains(t, notTidy.Reason, "example.com/dep@v0")
	assert.NotContains(t, notTidy.Reason, "cue mod tidy", "the cue command suggestion is the caller's to replace")
	assert.Equal(t, untidyModule, readModule(t, dir))
	assert.True(t, mtime(t, dir).Equal(past), "--check must not touch module.cue")
	_, statErr := os.Stat(filepath.Join(dir, "cue.mod", "local-module.cue"))
	assert.ErrorIs(t, statErr, fs.ErrNotExist)
}

func TestTidy_CheckOnTidyModule(t *testing.T) {
	dir, registry := newConsumer(t)
	ctx := context.Background()
	_, err := Tidy(ctx, dir, TidyOptions{Registry: registry})
	require.NoError(t, err)

	assert.NoError(t, checkTidy(ctx, t, dir, registry))
}

// countingRegistry is an HTTP server that records every request, for
// asserting that a refusal happens before any registry access.
func countingRegistry(t *testing.T) (string, *atomic.Int64) {
	t.Helper()
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://") + "+insecure", &hits
}

func TestTidy_NotModuleRoot(t *testing.T) {
	coldCUECache(t)
	registry, hits := countingRegistry(t)

	parent := writeConsumer(t, t.TempDir(), untidyModule)
	sub := filepath.Join(parent, "sub")
	require.NoError(t, os.Mkdir(sub, 0o750))
	file := filepath.Join(parent, "consumer.cue")

	for name, dir := range map[string]string{
		"missing":                  filepath.Join(parent, "nope"),
		"regular file":             file,
		"no cue.mod, parent has":   sub,
		"cue.mod without manifest": makeEmptyCueMod(t),
	} {
		t.Run(name, func(t *testing.T) {
			for _, check := range []bool{false, true} {
				_, err := Tidy(context.Background(), dir, TidyOptions{Registry: registry, Check: check})
				require.ErrorIs(t, err, ErrNotModuleRoot)
				abs, absErr := filepath.Abs(dir)
				require.NoError(t, absErr)
				assert.Contains(t, err.Error(), abs)
			}
		})
	}
	assert.Zero(t, hits.Load(), "a refused root must not reach the registry")
	assert.Equal(t, untidyModule, readModule(t, parent), "the parent module must not be tidied")
}

func makeEmptyCueMod(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "cue.mod"), 0o750))
	return dir
}

func TestTidy_RestoresProcessState(t *testing.T) {
	for _, ambient := range []struct {
		name  string
		value string
		set   bool
	}{
		{name: "CUE_REGISTRY set", value: "ambient.example+insecure", set: true},
		{name: "CUE_REGISTRY unset"},
	} {
		t.Run(ambient.name, func(t *testing.T) {
			// t.Setenv first so the test's own cleanup restores the
			// original value; then unset for the "unset" case.
			t.Setenv("CUE_REGISTRY", ambient.value)
			if !ambient.set {
				require.NoError(t, os.Unsetenv("CUE_REGISTRY"))
			}
			wd, err := os.Getwd()
			require.NoError(t, err)

			assertRestored := func(t *testing.T, step string) {
				t.Helper()
				now, err := os.Getwd()
				require.NoError(t, err)
				assert.Equal(t, wd, now, "%s: working directory", step)
				got, had := os.LookupEnv("CUE_REGISTRY")
				assert.Equal(t, ambient.set, had, "%s: CUE_REGISTRY presence", step)
				assert.Equal(t, ambient.value, got, "%s: CUE_REGISTRY value", step)
			}

			ctx := context.Background()

			dir, registry := newConsumer(t)
			_, err = Tidy(ctx, dir, TidyOptions{Registry: registry})
			require.NoError(t, err)
			assertRestored(t, "successful tidy")

			failing, _ := newConsumer(t)
			_, err = Tidy(ctx, failing, TidyOptions{Registry: unreachableRegistry})
			require.Error(t, err)
			var notTidy *NotTidyError
			require.NotErrorAs(t, err, &notTidy, "an unreachable registry is a resolution failure")
			assertRestored(t, "failing tidy")

			untidy, registry := newConsumer(t)
			err = checkTidy(ctx, t, untidy, registry)
			require.ErrorAs(t, err, &notTidy)
			assertRestored(t, "check failure")
		})
	}
}

// TestTidy_NotTidyWordingPinned fails when the embedded CUE version stops
// flattening modload.ErrModuleNotTidy into the text classify matches.
func TestTidy_NotTidyWordingPinned(t *testing.T) {
	dir, registry := newConsumer(t)
	root, err := filepath.Abs(dir)
	require.NoError(t, err)

	_, err = runModTidy(context.Background(), root, TidyOptions{Registry: registry, Check: true})
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), notTidySuggested+": "),
		"cmd/cue's not-tidy wording changed; update classify: %q", err.Error())
}

func TestClassify(t *testing.T) {
	for _, tc := range []struct {
		name       string
		output     string
		err        error
		wantReason string
		notTidy    bool
		wantMsg    string
	}{
		{name: "success"},
		{
			name:       "not tidy with reason",
			err:        errors.New("module is not tidy, use 'cue mod tidy': missing dependency providing package example.com/dep@v0"),
			notTidy:    true,
			wantReason: "missing dependency providing package example.com/dep@v0",
		},
		{
			name:    "not tidy without reason",
			err:     errors.New("module is not tidy, use 'cue mod tidy'"),
			notTidy: true,
		},
		{
			name:    "other error passes through",
			err:     errors.New("cannot find module providing package example.com/dep@v0"),
			wantMsg: "cannot find module providing package example.com/dep@v0",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := classify(tc.output, tc.err)
			if tc.err == nil {
				assert.NoError(t, err)
				return
			}
			var notTidy *NotTidyError
			if tc.notTidy {
				require.ErrorAs(t, err, &notTidy)
				assert.Equal(t, tc.wantReason, notTidy.Reason)
				return
			}
			require.NotErrorAs(t, err, &notTidy)
			assert.Equal(t, tc.wantMsg, err.Error())
		})
	}
}
