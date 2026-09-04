package platform

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
)

// tempOpmDir returns a config.cue path in a fresh OPM home (the file itself
// need not exist) and optionally seeds the sibling platform/ module.
func tempOpmDir(t *testing.T, withLocalPlatform bool) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.cue")
	if withLocalPlatform {
		require.NoError(t, config.WritePlatformModule(config.PlatformDir(configPath)))
	}
	return configPath
}

// platformModuleDir writes a minimal platform module (cue.mod/module.cue and
// platform.cue) into a fresh directory and returns it.
func platformModuleDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, config.WritePlatformModule(dir))
	return dir
}

func clusterGetterReturning(spec map[string]any, name, unavailable string, err error) ClusterSpecGetter {
	return func(context.Context) (map[string]any, string, string, error) {
		return spec, name, unavailable, err
	}
}

func TestResolve_FlagWinsOverEverything(t *testing.T) {
	configPath := tempOpmDir(t, true)
	flagDir := platformModuleDir(t)

	clusterCalled := false
	getter := func(context.Context) (map[string]any, string, string, error) {
		clusterCalled = true
		return map[string]any{"type": "kubernetes"}, "cluster", "", nil
	}

	dir, res, err := Resolve(context.Background(), ResolveOptions{
		PlatformFlag: flagDir,
		ConfigPath:   configPath,
		Cluster:      getter,
	})
	require.NoError(t, err)
	assert.Equal(t, flagDir, dir)
	assert.Equal(t, SourceFlagDir, res.Source)
	assert.Equal(t, flagDir, res.Location)
	assert.Equal(t, flagDir, res.Dir)
	assert.False(t, clusterCalled, "flag override must not read the cluster")
}

func TestResolve_FlagFileRefused(t *testing.T) {
	configPath := tempOpmDir(t, true)
	file := filepath.Join(t.TempDir(), "platform.cue")
	require.NoError(t, os.WriteFile(file, []byte(`name: "x"`), 0o600))

	_, _, err := Resolve(context.Background(), ResolveOptions{PlatformFlag: file, ConfigPath: configPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is a file")
	assert.Contains(t, err.Error(), "cue.mod/module.cue")
	assert.Contains(t, err.Error(), "opm config init")
}

func TestResolve_FlagDirWithoutModuleRefused(t *testing.T) {
	configPath := tempOpmDir(t, true)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte("package platform\n"), 0o600))

	_, _, err := Resolve(context.Background(), ResolveOptions{PlatformFlag: dir, ConfigPath: configPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a platform module")
	assert.Contains(t, err.Error(), "opm config init")
}

func TestResolve_FlagMissingRefused(t *testing.T) {
	configPath := tempOpmDir(t, true)

	_, _, err := Resolve(context.Background(), ResolveOptions{PlatformFlag: filepath.Join(t.TempDir(), "nope"), ConfigPath: configPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResolve_ClusterGeneratesModuleUnderCache(t *testing.T) {
	configPath := tempOpmDir(t, true)
	src := fixtureGraph()

	dir, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster: clusterGetterReturning(map[string]any{
			"type": "kubernetes",
			"registry": map[string]any{
				"opmodel.dev/catalogs/opm@v4": map[string]any{"version": "4.0.1"},
			},
			"skewPolicy": "Refuse",
		}, "cluster", "", nil),
		ModFiles: src,
	})
	require.NoError(t, err)
	assert.Equal(t, SourceClusterCR, res.Source)
	assert.Equal(t, "cluster", res.Location)
	assert.Equal(t, dir, res.Dir)
	assert.Equal(t, "Refuse", res.SkewPolicy)
	assert.Empty(t, res.Warning)
	assert.Equal(t, config.PlatformCacheDir(configPath), filepath.Dir(dir), "the generated module lives in the OPM home cache")
	_, err = os.Stat(filepath.Join(dir, "cue.mod", "module.cue"))
	assert.NoError(t, err)
	assert.Contains(t, res.Describe(), dir)
}

func TestResolve_LegacyClusterCRFailsAtGeneration(t *testing.T) {
	configPath := tempOpmDir(t, true)
	src := fixtureGraph()

	_, _, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster: clusterGetterReturning(map[string]any{
			"type": "kubernetes",
			"registry": map[string]any{
				"opmodel.dev/catalogs/opm": map[string]any{"filter": map[string]any{"range": ">=1.0.0-0"}},
			},
		}, "cluster", "", nil),
		ModFiles: src,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEntryMissingVersion)
	assert.Contains(t, err.Error(), "predate the scalar-version subscription shape")
	assert.Empty(t, src.calls, "no registry access before the legacy-CR refusal")
}

func TestResolve_FallbackToLocalWarns(t *testing.T) {
	configPath := tempOpmDir(t, true)

	dir, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster:    clusterGetterReturning(nil, "", "no Platform CR in the cluster", nil),
	})
	require.NoError(t, err)
	assert.Equal(t, config.PlatformDir(configPath), dir)
	assert.Equal(t, SourceLocalDefault, res.Source)
	assert.Equal(t, dir, res.Dir)
	assert.NotEmpty(t, res.Warning, "cluster→local fallback must carry a warning")
	assert.Contains(t, res.Warning, "no Platform CR in the cluster")
}

// The D21 fallback is never silent: when the cluster Platform is unavailable
// and resolution drops to the local default, the provenance warning banner must
// actually reach the CLI's output sink — not merely land in Resolution.Warning.
func TestResolve_FallbackEmitsProvenanceBanner(t *testing.T) {
	configPath := tempOpmDir(t, true)

	var buf bytes.Buffer
	output.SetLogWriter(&buf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

	_, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster:    clusterGetterReturning(nil, "", "no Platform CR in the cluster", nil),
	})
	require.NoError(t, err)
	assert.Equal(t, SourceLocalDefault, res.Source)

	emitted := buf.String()
	assert.Contains(t, emitted, "falling back to the local default platform",
		"the fallback must emit the D21 provenance warning, not just record it")
	assert.Contains(t, emitted, "no Platform CR in the cluster",
		"the emitted banner must name why the cluster Platform was unavailable")
}

func TestResolve_ClusterHardErrorIsFatal(t *testing.T) {
	configPath := tempOpmDir(t, true)

	boom := errors.New("connection refused")
	_, _, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster:    clusterGetterReturning(nil, "", "", boom),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestResolve_OfflineNeverReadsCluster(t *testing.T) {
	// nil Cluster getter = offline command (build/render): local default only.
	configPath := tempOpmDir(t, true)

	dir, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
	})
	require.NoError(t, err)
	assert.Equal(t, SourceLocalDefault, res.Source)
	assert.Equal(t, config.PlatformDir(configPath), dir)
	assert.Empty(t, res.Warning, "offline local default is not a fallback")
}

func TestResolve_NoSourceAvailable(t *testing.T) {
	configPath := tempOpmDir(t, false) // no local platform/ module

	_, _, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), config.PlatformDir(configPath))
	assert.Contains(t, err.Error(), "opm config init")
}

func TestResolve_LocalDefaultMalformedRefused(t *testing.T) {
	configPath := tempOpmDir(t, false)
	dir := config.PlatformDir(configPath)
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte("package platform\n"), 0o600))

	_, _, err := Resolve(context.Background(), ResolveOptions{ConfigPath: configPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a platform module")
}

func TestResolution_Describe(t *testing.T) {
	assert.Equal(t, "platform: /p (--platform)", Resolution{Source: SourceFlagDir, Location: "/p", Dir: "/p"}.Describe())
	assert.Equal(t, "platform: cluster Platform CR cluster (generated module /c/abc)", Resolution{Source: SourceClusterCR, Location: "cluster", Dir: "/c/abc"}.Describe())
	assert.Equal(t, "platform: /home/x/.opm/platform (local default)", Resolution{Source: SourceLocalDefault, Location: "/home/x/.opm/platform", Dir: "/home/x/.opm/platform"}.Describe())
}
