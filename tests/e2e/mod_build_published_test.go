package e2e

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/open-platform-model/cli/tests/fixtures"
)

// publishedPodinfo returns the podinfo fixture's major-free module path and
// its major, read from the tree's identity package.
func publishedPodinfo(t *testing.T) (path, major string) {
	t.Helper()
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}
	c, err := fixtures.Load("podinfo")
	require.NoError(t, err)
	path, major, ok := strings.Cut(c.ModulePath, "@")
	require.True(t, ok, "fixture module path carries its major: %s", c.ModulePath)
	return path, major
}

// requireYAMLDocuments asserts out is a non-empty stream of YAML documents.
func requireYAMLDocuments(t *testing.T, out string) {
	t.Helper()
	dec := yaml.NewDecoder(strings.NewReader(out))
	n := 0
	for {
		var doc map[string]any
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err, "stdout must be parseable YAML only")
		n++
	}
	assert.Positive(t, n, "expected manifests on stdout")
}

// TestE2E_ModBuild_PublishedModuleWithMajor renders the published podinfo
// fixture by module path, floating within its major.
func TestE2E_ModBuild_PublishedModuleWithMajor(t *testing.T) {
	path, major := publishedPodinfo(t)

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), seedRenderHome(t), 180*time.Second,
		"module", "build", path, "--version", major)
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stderr, "Resolved "+path+" -> "+major)
	assert.Contains(t, stderr, "(newest in "+major+")")
	requireYAMLDocuments(t, stdout)
}

// TestE2E_ModBuild_PublishedModuleWithoutVersion resolves the highest
// core-compatible major, reports the selection on stderr and keeps stdout
// parseable.
func TestE2E_ModBuild_PublishedModuleWithoutVersion(t *testing.T) {
	path, major := publishedPodinfo(t)

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), seedRenderHome(t), 180*time.Second,
		"module", "build", path)
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stderr, "Resolved "+path+" -> "+major)
	assert.Contains(t, stderr, "highest major on core v2")
	requireYAMLDocuments(t, stdout)
}

// TestE2E_ModBuild_PublishedModulePinnedWithValues covers "Published module
// with a pinned version": the exact tag is rendered with the -f values in
// place of the module's debugValues.
func TestE2E_ModBuild_PublishedModulePinnedWithValues(t *testing.T) {
	path, major := publishedPodinfo(t)
	c, err := fixtures.Load("podinfo")
	require.NoError(t, err)

	workDir := t.TempDir()
	values := filepath.Join(workDir, "values.cue")
	require.NoError(t, os.WriteFile(values, []byte(`values: {
	image: {repository: "ghcr.io/stefanprodan/podinfo", tag: "6.7.0", digest: ""}
	replicas: 3
}
`), 0o600))

	stdout, stderr, err := runOPMWithEnv(t, workDir, seedRenderHome(t), 180*time.Second,
		"module", "build", path, "--version", c.Version, "-f", values)
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stderr, "Resolved "+path+" -> "+major+" "+c.Version+" (pinned)")
	assert.Contains(t, stdout, "ghcr.io/stefanprodan/podinfo:6.7.0", "the -f values reach the render")
	assert.NotContains(t, stdout, "podinfo:6.7.1", "debugValues are not used with -f")
	requireYAMLDocuments(t, stdout)
}

// TestE2E_ModBuild_PublishedModuleMajorSuffixRefused refuses a major-suffixed
// module path with the --version spelling, before any registry access.
func TestE2E_ModBuild_PublishedModuleMajorSuffixRefused(t *testing.T) {
	path, major := publishedPodinfo(t)

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), seedRenderHome(t), 60*time.Second,
		"module", "build", path+"@"+major)
	require.Error(t, err, "stdout: %s", stdout)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 2, exitErr.ExitCode(), "stderr: %s", stderr)
	assert.Contains(t, stderr, "must not carry a major")
	assert.Contains(t, stderr, path+" --version "+major)
	assert.Empty(t, stdout)
}
