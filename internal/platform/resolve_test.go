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

// tempOpmDir writes a config.cue path (the file itself need not exist) and
// optionally a sibling platform.cue, returning the config path.
func tempOpmDir(t *testing.T, withLocalPlatform bool) string {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.cue")
	if withLocalPlatform {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"),
			[]byte(config.DefaultPlatformTemplate), 0o600))
	}
	return configPath
}

func clusterGetterReturning(spec map[string]any, name, unavailable string, err error) ClusterSpecGetter {
	return func(context.Context) (map[string]any, string, string, error) {
		return spec, name, unavailable, err
	}
}

func TestResolve_FlagWinsOverEverything(t *testing.T) {
	configPath := tempOpmDir(t, true)
	flagFile := writePlatformFile(t, `name: "override"
type: "kubernetes"
`)

	clusterCalled := false
	getter := func(context.Context) (map[string]any, string, string, error) {
		clusterCalled = true
		return map[string]any{"type": "kubernetes"}, "cluster", "", nil
	}

	in, res, err := Resolve(context.Background(), ResolveOptions{
		PlatformFlag: flagFile,
		ConfigPath:   configPath,
		Cluster:      getter,
	})
	require.NoError(t, err)
	assert.Equal(t, "override", in.Name)
	assert.Equal(t, SourceFlagFile, res.Source)
	assert.Equal(t, flagFile, res.Location)
	assert.False(t, clusterCalled, "flag override must not read the cluster")
}

func TestResolve_ClusterUsedWhenNoFlag(t *testing.T) {
	configPath := tempOpmDir(t, true)

	in, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster: clusterGetterReturning(map[string]any{
			"type": "kubernetes",
		}, "cluster", "", nil),
	})
	require.NoError(t, err)
	assert.Equal(t, "cluster", in.Name)
	assert.Equal(t, SourceClusterCR, res.Source)
	assert.Empty(t, res.Warning)
}

func TestResolve_FallbackToLocalWarns(t *testing.T) {
	configPath := tempOpmDir(t, true)

	in, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster:    clusterGetterReturning(nil, "", "no Platform CR in the cluster", nil),
	})
	require.NoError(t, err)
	assert.Equal(t, "cluster", in.Name) // template's seeded name
	assert.Equal(t, SourceLocalDefault, res.Source)
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

	in, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
	})
	require.NoError(t, err)
	assert.Equal(t, SourceLocalDefault, res.Source)
	assert.Equal(t, "cluster", in.Name)
	assert.Empty(t, res.Warning, "offline local default is not a fallback")
}

func TestResolve_NoSourceAvailable(t *testing.T) {
	configPath := tempOpmDir(t, false) // no local platform.cue

	_, _, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opm config init")
}

func TestResolution_Describe(t *testing.T) {
	assert.Contains(t, Resolution{Source: SourceFlagFile, Location: "p.cue"}.Describe(), "--platform")
	assert.Contains(t, Resolution{Source: SourceClusterCR, Location: "cluster"}.Describe(), "cluster Platform CR")
	assert.Contains(t, Resolution{Source: SourceLocalDefault, Location: "x"}.Describe(), "local default")
}
