package e2e

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// seedRenderHome returns a hermetic HOME whose ~/.opm holds exactly what
// `opm config init` writes (config.cue against the public registry mapping
// and the local default platform module), so render-bearing e2e tests never
// depend on the developer's real ~/.opm. Cleaned up with the test.
//
// The CUE module cache is not test state: the child process is pointed at
// the invoking user's cache (CUE_CACHE_DIR) so a cold temp HOME does not
// refetch every module and the read-only extracted tree never lands under
// the temp directory. Should anything still land there, the cleanup makes
// the tree writable before t.TempDir removes it.
func seedRenderHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Cleanup(func() { makeWritable(home) })
	if userCache, err := os.UserCacheDir(); err == nil && os.Getenv("CUE_CACHE_DIR") == "" {
		t.Setenv("CUE_CACHE_DIR", filepath.Join(userCache, "cue"))
	}
	opmDir := filepath.Join(home, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))
	configPath := filepath.Join(opmDir, "config.cue")
	require.NoError(t, os.WriteFile(configPath, []byte(config.DefaultConfigTemplate), 0o600))
	require.NoError(t, config.WritePlatformModule(config.PlatformDir(configPath)))
	return home
}

// makeWritable chmods every entry under dir writable so a read-only tree
// (CUE extracts module cache files read-only) can be removed.
func makeWritable(dir string) {
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Best effort: an unreadable entry is skipped, the walk continues.
			return nil //nolint:nilerr // walk continues past entries it cannot read
		}
		if d.IsDir() {
			_ = os.Chmod(path, 0o755)
		} else {
			_ = os.Chmod(path, 0o644)
		}
		return nil
	})
}

// olderCatalogPin is a published build of the abstraction catalog that is
// older than the build the examples module requires (examples/cue.mod pins
// v4.0.1 or newer), so a platform pinning it exhibits catalog version skew.
const olderCatalogPin = "v4.0.0"

// seedSkewPlatform writes a platform module identical to the seeded default
// except that the abstraction catalog is pinned at olderCatalogPin, and
// returns its directory.
func seedSkewPlatform(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, config.WritePlatformModule(dir))
	modFile := filepath.Join(dir, filepath.FromSlash(config.PlatformModuleFileName))
	content, err := os.ReadFile(modFile)
	require.NoError(t, err)
	skewed := strings.Replace(string(content), config.DefaultCatalogPins[0], olderCatalogPin, 1)
	require.NotEqual(t, string(content), skewed, "the seeded module must pin the abstraction catalog")
	require.NoError(t, os.WriteFile(modFile, []byte(skewed), 0o600))
	return dir
}

// examplesCatalogPin reads the abstraction catalog build the examples module
// requires, so the skew assertions name the version the kernel reports
// rather than a literal that drifts on the next fixture bump.
func examplesCatalogPin(t *testing.T, repoRoot string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repoRoot, "examples", "cue.mod", "module.cue"))
	require.NoError(t, err)
	m := regexp.MustCompile(`"` + regexp.QuoteMeta(config.DefaultCatalogPaths[0]) + `":\s*\{\s*v:\s*"([^"]+)"`).FindStringSubmatch(string(content))
	require.Len(t, m, 2, "examples/cue.mod/module.cue must pin %s", config.DefaultCatalogPaths[0])
	return m[1]
}

// podinfoExample returns the repo root and the podinfo example instance file.
func podinfoExample(t *testing.T) (repoRoot, instanceFile string) {
	t.Helper()
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	instanceFile = filepath.Join(repoRoot, "examples", "instances", "podinfo", "instance.cue")
	if _, statErr := os.Stat(instanceFile); statErr != nil {
		t.Skipf("examples/instances/podinfo not available: %v", statErr)
	}
	return repoRoot, instanceFile
}

func exitCode(t *testing.T, err error) int {
	t.Helper()
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr), "expected an exit error, got %v", err)
	return exitErr.ExitCode()
}

// TestE2E_InstanceBuild_LayersValuesFile covers the kernel-render scenario
// "Instance file with extra values": a -f file is layered onto the instance
// package as a values source, the render imports the layered package and the
// rendered objects reflect the override, while the package's own values.cue
// still applies.
func TestE2E_InstanceBuild_LayersValuesFile(t *testing.T) {
	_, instanceFile := podinfoExample(t)
	home := seedRenderHome(t)

	workDir := t.TempDir()
	override := filepath.Join(workDir, "override.cue")
	require.NoError(t, os.WriteFile(override, []byte("package podinfo\nvalues: image: tag: \"6.7.0\"\n"), 0o600))

	stdout, stderr, err := runOPMWithEnv(t, workDir, home, 180*time.Second,
		"instance", "build", instanceFile, "-f", override)
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Contains(t, stdout, "ghcr.io/stefanprodan/podinfo:6.7.0", "the -f override must reach the rendered Deployment")
	assert.NotContains(t, stdout, "podinfo:6.7.1", "the module default must be overridden")
	assert.Contains(t, stdout, "replicas: 2", "the package's own values.cue still applies beside the override")
	assert.Contains(t, stderr, "(local default)", "provenance names the local default platform module")
}

// TestE2E_InstanceBuild_SkewWarnsByDefault covers "Render warnings reach the
// user": with no skewPolicy configured, a module requiring a newer catalog
// build than the platform pins renders against the platform's build and the
// kernel's skew warning names the path and both versions.
func TestE2E_InstanceBuild_SkewWarnsByDefault(t *testing.T) {
	repoRoot, instanceFile := podinfoExample(t)
	home := seedRenderHome(t)
	platformDir := seedSkewPlatform(t)
	required := examplesCatalogPin(t, repoRoot)

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), home, 180*time.Second,
		"instance", "build", instanceFile, "--platform", platformDir)
	require.NoError(t, err, "stderr: %s", stderr)

	assert.NotEmpty(t, stdout, "the render proceeds under the warn policy")
	assert.Contains(t, stderr, `version skew on "`+config.DefaultCatalogPaths[0]+`"`)
	assert.Contains(t, stderr, "module requires "+required)
	assert.Contains(t, stderr, "platform carries "+olderCatalogPin)
	assert.Contains(t, stderr, "rendering against the platform's build")
	assert.Contains(t, stderr, "(--platform)", "provenance names the flag-provided directory")
	assert.NotContains(t, stderr, "skew policy: refuse")
}

// TestE2E_InstanceBuild_SkewRefusedByConfig covers "Skew refusal is a
// validation failure" and the config-types scenario "Refuse configured":
// with skewPolicy "refuse" in config.cue, the render is refused before
// evaluation as a validation error naming the path and both versions.
func TestE2E_InstanceBuild_SkewRefusedByConfig(t *testing.T) {
	repoRoot, instanceFile := podinfoExample(t)
	home := seedRenderHome(t)
	platformDir := seedSkewPlatform(t)
	required := examplesCatalogPin(t, repoRoot)

	configPath := filepath.Join(home, ".opm", "config.cue")
	refusing := strings.Replace(config.DefaultConfigTemplate, `skewPolicy: "warn"`, `skewPolicy: "refuse"`, 1)
	require.NotEqual(t, config.DefaultConfigTemplate, refusing, "the config template must carry the skewPolicy key")
	require.NoError(t, os.WriteFile(configPath, []byte(refusing), 0o600))

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), home, 180*time.Second,
		"instance", "build", instanceFile, "--platform", platformDir)
	require.Error(t, err, "stdout: %s", stdout)

	assert.Equal(t, opmexit.ExitValidationError, exitCode(t, err), "a skew refusal is a validation failure")
	assert.Empty(t, stdout, "nothing is rendered when the render is refused before evaluation")
	assert.Contains(t, stderr, "skew policy: refuse (config)", "provenance reports the refuse policy and its source")
	assert.Contains(t, stderr, "render failed")
	assert.Contains(t, stderr, `version skew on "`+config.DefaultCatalogPaths[0]+`"`)
	assert.Contains(t, stderr, "module requires "+required)
	assert.Contains(t, stderr, "platform carries "+olderCatalogPin)
}
