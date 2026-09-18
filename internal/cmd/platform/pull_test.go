package platformcmd

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/open-platform-model/library/opm/helper/platformmodule"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/platform"
)

// pullModFiles is a fixture module graph: module version string -> the paths
// and versions that module requires. A version absent from the map is an
// unpublished build. No network, no cache.
type pullModFiles struct {
	graph map[string][]platformmodule.Dep
}

func (f *pullModFiles) ModFile(_ context.Context, mv module.Version) (*modfile.File, error) {
	deps, ok := f.graph[mv.String()]
	if !ok {
		return nil, fmt.Errorf("module %s: module not found", mv)
	}
	mf := &modfile.File{Module: mv.Path(), Deps: map[string]*modfile.Dep{}}
	for _, d := range deps {
		mf.Deps[d.Path] = &modfile.Dep{Version: d.Version}
	}
	if err := mf.Init(); err != nil {
		return nil, err
	}
	return mf, nil
}

// pullGraph publishes core at the library's verified release plus the two
// catalogs the fixture Platform's status records.
func pullGraph() *pullModFiles {
	return &pullModFiles{graph: map[string][]platformmodule.Dep{
		"opmodel.dev/core@" + schema.DefaultSchemaVersion(): nil,
		"opmodel.dev/catalogs/opm@v4.0.1": {
			{Path: platformmodule.CorePath, Version: schema.DefaultSchemaVersion()},
		},
		"opmodel.dev/catalogs/k8s@v1.0.0-alpha.2": {
			{Path: platformmodule.CorePath, Version: schema.DefaultSchemaVersion()},
		},
	}}
}

// newPullDynamic builds a fake dynamic client that knows the Platform GVR,
// pre-seeded with objs.
func newPullDynamic(objs ...runtime.Object) *dynamicfake.FakeDynamicClient {
	gvrToKind := map[k8sschema.GroupVersionResource]string{
		inventory.PlatformGVR: inventory.KindPlatform + "List",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToKind, objs...)
}

func platformDoc(spec, status map[string]any, generation int64) *unstructured.Unstructured {
	obj := map[string]any{
		"apiVersion": inventory.GroupOpmodel + "/" + inventory.VersionV1Alpha1,
		"kind":       inventory.KindPlatform,
		"metadata": map[string]any{
			"name":       inventory.PlatformSingletonName,
			"generation": generation,
		},
		"spec": spec,
	}
	if status != nil {
		obj["status"] = status
	}
	return &unstructured.Unstructured{Object: obj}
}

// reconciledPlatform is the fixture cluster: one authored subscription and
// one catalog an active TransformerRegistration contributed, as the operator
// resolved them.
func reconciledPlatform() *unstructured.Unstructured {
	return platformDoc(
		map[string]any{
			"type": "kubernetes",
			"registry": map[string]any{
				"opmodel.dev/catalogs/opm@v4": map[string]any{"version": "4.0.1"},
			},
		},
		map[string]any{
			"observedGeneration": int64(7),
			"packageIdentity":    "gen-7-3f9a1c2b",
			"operatorVersion":    "v1.0.0-alpha.20",
			"registry": []any{
				map[string]any{
					"catalog": "opmodel.dev/catalogs/opm@v4",
					"version": "4.0.1",
					"enabled": true,
					"source":  platform.EntrySourceSubscription,
				},
				map[string]any{
					"catalog": "opmodel.dev/catalogs/k8s@v1",
					"version": "1.0.0-alpha.2",
					"enabled": true,
					"source":  platform.EntrySourceRegistration,
				},
			},
			"conditions": []any{
				map[string]any{"type": "Ready", "status": "True", "reason": "Generated"},
			},
		}, 7)
}

// pullConfig gives the command a fresh OPM home, so the generated-module
// cache is a temporary directory of this test's own.
func pullConfig(t *testing.T) *config.GlobalConfig {
	t.Helper()
	return &config.GlobalConfig{ConfigPath: filepath.Join(t.TempDir(), "config.cue")}
}

// runPull drives the command in process against a fake cluster and the
// fixture module graph, returning what it printed on stdout.
func runPull(t *testing.T, cfg *config.GlobalConfig, dyn *dynamicfake.FakeDynamicClient, dir string, force bool) (string, error) {
	t.Helper()

	oldOut := os.Stdout
	r, w, perr := os.Pipe()
	require.NoError(t, perr)
	os.Stdout = w

	err := RunPlatformPull(context.Background(), cfg, PullOptions{
		Dir:      dir,
		Force:    force,
		Cluster:  platform.ClusterPlatformGetterFor(dyn),
		ModFiles: pullGraph(),
	})

	require.NoError(t, w.Close())
	os.Stdout = oldOut
	raw, rerr := io.ReadAll(r)
	require.NoError(t, rerr)
	require.NoError(t, r.Close())
	return string(raw), err
}

// readTree returns every file under dir keyed by its slash-separated path
// relative to dir.
func readTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	require.NoError(t, filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	}))
	return files
}

// cachedModuleDir returns the one generated module under the OPM home cache.
func cachedModuleDir(t *testing.T, cfg *config.GlobalConfig) string {
	t.Helper()
	cache := config.PlatformCacheDir(cfg.ConfigPath)
	entries, err := os.ReadDir(cache)
	require.NoError(t, err)
	require.Len(t, entries, 1, "exactly one module was generated")
	return filepath.Join(cache, entries[0].Name())
}

// exitCode asserts err is an ExitError and returns its code.
func exitCode(t *testing.T, err error) int {
	t.Helper()
	var ee *opmexit.ExitError
	require.ErrorAs(t, err, &ee)
	return ee.Code
}

func TestRunPlatformPull_WritesTheCachedModule(t *testing.T) {
	cfg := pullConfig(t)
	dir := filepath.Join(t.TempDir(), "cluster-platform")

	_, err := runPull(t, cfg, newPullDynamic(reconciledPlatform()), dir, false)
	require.NoError(t, err)

	written := readTree(t, dir)
	modFile := written["cue.mod/module.cue"]
	require.NotEmpty(t, modFile, "the written module carries its cue.mod")
	assert.Contains(t, modFile, `"opmodel.dev/catalogs/opm@v4"`)
	assert.Contains(t, modFile, `"v4.0.1"`)
	assert.Contains(t, modFile, `"opmodel.dev/catalogs/k8s@v1"`,
		"the registration-sourced catalog is pinned too")
	assert.Contains(t, modFile, `"v1.0.0-alpha.2"`)
	assert.Contains(t, written["platform.cue"], `"opmodel.dev/catalogs/k8s@v1"`)

	assert.Equal(t, readTree(t, cachedModuleDir(t, cfg)), written,
		"the written module is the generated module, file for file")
}

func TestRunPlatformPull_TwicePullsIdenticalBytes(t *testing.T) {
	cfg := pullConfig(t)
	dyn := newPullDynamic(reconciledPlatform())
	dir := filepath.Join(t.TempDir(), "cluster-platform")

	_, err := runPull(t, cfg, dyn, dir, false)
	require.NoError(t, err)
	first := readTree(t, dir)

	_, err = runPull(t, cfg, dyn, dir, true)
	require.NoError(t, err)

	assert.Equal(t, first, readTree(t, dir))
}

func TestRunPlatformPull_NonEmptyTargetRefused(t *testing.T) {
	cfg := pullConfig(t)
	dir := t.TempDir()
	stray := filepath.Join(dir, "notes.md")
	require.NoError(t, os.WriteFile(stray, []byte("mine\n"), 0o600))

	_, err := runPull(t, cfg, newPullDynamic(reconciledPlatform()), dir, false)
	require.Error(t, err)
	assert.Equal(t, opmexit.ExitGeneralError, exitCode(t, err))
	assert.Contains(t, err.Error(), dir)
	assert.Contains(t, err.Error(), "--force")

	assert.Equal(t, map[string]string{"notes.md": "mine\n"}, readTree(t, dir),
		"a refused pull writes nothing")
}

func TestRunPlatformPull_ForceReplacesTheTarget(t *testing.T) {
	cfg := pullConfig(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stale.cue"), []byte("package old\n"), 0o600))

	_, err := runPull(t, cfg, newPullDynamic(reconciledPlatform()), dir, true)
	require.NoError(t, err)

	written := readTree(t, dir)
	assert.NotContains(t, written, "stale.cue", "--force replaces the directory's contents")
	assert.Contains(t, written, "platform.cue")
}

func TestRunPlatformPull_NoPlatformExitsNotFound(t *testing.T) {
	cfg := pullConfig(t)
	dir := filepath.Join(t.TempDir(), "cluster-platform")

	_, err := runPull(t, cfg, newPullDynamic(), dir, false)
	require.Error(t, err)
	assert.Equal(t, opmexit.ExitNotFound, exitCode(t, err))
	assert.Contains(t, err.Error(), inventory.PlatformSingletonName)
	assert.Contains(t, err.Error(), "nothing to pull")

	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr), "a failed pull creates no directory")
}

// A Platform predating the scalar-version subscription shape cannot be
// generated from: the refusal is a validation failure carrying the hint.
func TestRunPlatformPull_LegacySpecExitsValidationError(t *testing.T) {
	cfg := pullConfig(t)
	dir := filepath.Join(t.TempDir(), "cluster-platform")
	legacy := platformDoc(map[string]any{
		"type": "kubernetes",
		"registry": map[string]any{
			"opmodel.dev/catalogs/opm": map[string]any{"filter": map[string]any{"range": ">=1.0.0-0"}},
		},
	}, nil, 1)

	_, err := runPull(t, cfg, newPullDynamic(legacy), dir, false)
	require.Error(t, err)
	assert.Equal(t, opmexit.ExitValidationError, exitCode(t, err))
	assert.ErrorIs(t, err, platform.ErrEntryMissingVersion)
	assert.Contains(t, err.Error(), "predate the scalar-version subscription shape")
}

func TestRunPlatformPull_ReportNamesWhatItReproduced(t *testing.T) {
	cfg := pullConfig(t)
	dir := filepath.Join(t.TempDir(), "cluster-platform")

	report, err := runPull(t, cfg, newPullDynamic(reconciledPlatform()), dir, false)
	require.NoError(t, err)

	assert.Contains(t, report, "cluster Platform CR cluster")
	assert.Contains(t, report, "effective registry, package gen-7-3f9a1c2b")
	assert.Contains(t, report, "generation 7 (observed 7), operator v1.0.0-alpha.20")
	assert.Contains(t, report, "registry: 2 entries")
	assert.Contains(t, report, platform.EntrySourceSubscription)
	assert.Contains(t, report, platform.EntrySourceRegistration)
	assert.Contains(t, report, "opmodel.dev/catalogs/k8s@v1")
	assert.Contains(t, report, "1.0.0-alpha.2")
	assert.Contains(t, report, "enabled")
	assert.Contains(t, report, "wrote "+dir)
	assert.Contains(t, report, "opm module build --platform "+dir)
}

// Without a recorded registry the spec is what generates, and the report
// says so rather than claiming a package the operator never produced.
func TestRunPlatformPull_ReportNamesTheSpecFallback(t *testing.T) {
	cfg := pullConfig(t)
	dir := filepath.Join(t.TempDir(), "cluster-platform")
	solo := platformDoc(map[string]any{
		"type": "kubernetes",
		"registry": map[string]any{
			"opmodel.dev/catalogs/opm@v4": map[string]any{"version": "4.0.1"},
		},
	}, nil, 3)

	report, err := runPull(t, cfg, newPullDynamic(solo), dir, false)
	require.NoError(t, err)

	assert.Contains(t, report, "spec registry, no operator generation recorded")
	assert.Contains(t, report, "generation 3, no operator generation recorded")
	assert.Contains(t, report, "registry: 1 entry")
	assert.NotContains(t, report, platform.EntrySourceRegistration)
}

func TestNewPlatformPullCmd(t *testing.T) {
	cmd := NewPlatformPullCmd(&config.GlobalConfig{})

	assert.Equal(t, "pull <dir>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	// The help states the flag and the exit-code contract: a reader who
	// scripts the command needs to tell "nothing written" from "no CR".
	assert.Contains(t, cmd.Long, "--force")
	assert.Contains(t, cmd.Long, "1   <dir> is not empty")
	assert.Contains(t, cmd.Long, "2   the recorded registry does not generate")
	assert.Contains(t, cmd.Long, "3   the cluster could not be reached")
	assert.Contains(t, cmd.Long, "5   no readable Platform CR")

	require.NotNil(t, cmd.Flags().Lookup("force"))
	require.NotNil(t, cmd.Flags().Lookup("kubeconfig"), "pull reads a cluster")
	require.NotNil(t, cmd.Flags().Lookup("context"))

	// Exactly one target directory.
	assert.Error(t, cmd.Args(cmd, nil))
	assert.NoError(t, cmd.Args(cmd, []string{"./dir"}))
	assert.Error(t, cmd.Args(cmd, []string{"a", "b"}))
}
