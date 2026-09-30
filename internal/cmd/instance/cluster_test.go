package instance

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// captureLog redirects the CLI's log sink for the test and returns it.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	output.SetLogWriter(&buf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })
	return &buf
}

// freshClient clears the process-wide cached Kubernetes client around a test
// that builds one from its own kubeconfig.
func freshClient(t *testing.T) {
	t.Helper()
	kubernetes.ResetClient()
	t.Cleanup(kubernetes.ResetClient)
}

// writeKubeconfig writes a one-context kubeconfig pointing at server.
func writeKubeconfig(t *testing.T, server string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(path, []byte(`apiVersion: v1
kind: Config
clusters:
- name: c
  cluster:
    server: `+server+`
contexts:
- name: ctx
  context:
    cluster: c
    user: u
current-context: ctx
users:
- name: u
  user:
    token: t
`), 0o600))
	return path
}

func TestOptionalClusterGetter_OfflineAndFlagSkipTheCluster(t *testing.T) {
	freshClient(t)
	kf := cmdutil.K8sFlags{Kubeconfig: writeKubeconfig(t, "https://127.0.0.1:1")}
	assert.Nil(t, optionalClusterGetter(&config.GlobalConfig{}, kf, "", true), "--offline never builds a getter")
	assert.Nil(t, optionalClusterGetter(&config.GlobalConfig{}, kf, "./p", false), "--platform wins; the cluster is not read")
}

func TestOptionalClusterGetter_NoContextIsSilent(t *testing.T) {
	buf := captureLog(t)
	for name, path := range map[string]string{
		"missing": filepath.Join(t.TempDir(), "missing"),
		"empty":   writeEmpty(t),
	} {
		t.Run(name, func(t *testing.T) {
			freshClient(t)
			assert.Nil(t, optionalClusterGetter(&config.GlobalConfig{}, cmdutil.K8sFlags{Kubeconfig: path}, "", false))
		})
	}
	assert.Empty(t, buf.String(), "no kubeconfig context is not a fallback: nothing is warned")
}

func writeEmpty(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "empty")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	return path
}

func TestOptionalClusterGetter_BrokenKubeconfigWarns(t *testing.T) {
	freshClient(t)
	buf := captureLog(t)
	kf := cmdutil.K8sFlags{Kubeconfig: writeKubeconfig(t, "https://127.0.0.1:1"), Context: "no-such-context"}
	assert.Nil(t, optionalClusterGetter(&config.GlobalConfig{}, kf, "", false))
	assert.Contains(t, buf.String(), "cluster Platform not used")
}

// TestInstanceBuild_UnreachableClusterFallsBackToDeps covers "Unreachable
// cluster degrades to the deps": the kubeconfig context names a server that
// refuses connections, and the build completes against the instance
// package's own deps with exactly one cluster warning.
func TestInstanceBuild_UnreachableClusterFallsBackToDeps(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed tests")
	}
	instanceFile, err := filepath.Abs(filepath.Join("..", "..", "..", "examples", "instances", "podinfo", "instance.cue"))
	require.NoError(t, err)
	// OPM_REGISTRY when set: PR CI seeds the tree's fixtures into a job-local
	// registry, and the examples pin the tree's fixture version.
	registry := "opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
	if r := os.Getenv("OPM_REGISTRY"); r != "" {
		registry = r
	}
	if _, err := config.NewKernel(registry).SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}
	freshClient(t)
	buf := captureLog(t)

	cfg := &config.GlobalConfig{ConfigPath: filepath.Join(t.TempDir(), "config.cue"), Registry: registry}
	lookup := clusterLookup{k8s: cmdutil.K8sFlags{Kubeconfig: writeKubeconfig(t, "https://127.0.0.1:1")}}
	outDir := t.TempDir()
	require.NoError(t, runInstanceBuild(instanceFile, cfg, &cmdutil.InstanceFileFlags{}, lookup, "", "yaml", true, outDir))

	logs := buf.String()
	assert.Equal(t, 1, strings.Count(logs, "cluster Platform not used"), "one cluster warning: %s", logs)
	assert.Contains(t, logs, "could not reach the cluster")
	assert.Contains(t, logs, "rendering against the instance's own deps")
	assert.Contains(t, logs, "platform: instance deps (opmodel.dev/catalogs/opm@v4")
	written, err := os.ReadDir(outDir)
	require.NoError(t, err)
	assert.NotEmpty(t, written, "the render completed against the deps")
}
