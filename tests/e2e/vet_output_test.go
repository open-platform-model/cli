package e2e

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_InstanceVet_Output(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	testdataDir := filepath.Join(cwd, "testdata", "vet-errors")

	tmpDir, err := os.MkdirTemp("", "e2e-rel-vet-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	_, stderr, err := runOPM(t, tmpDir, "instance", "vet",
		filepath.Join(testdataDir, "instance", "instance.cue"),
		"-f", filepath.Join(testdataDir, "instance", "values.cue"))

	// Assert exit code 2
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode())

	// Assert grouped shape exists
	assert.Contains(t, stderr, "render failed: 2 issues")
	assert.Contains(t, stderr, "field not allowed")
	assert.Contains(t, stderr, "values.test")
	assert.Contains(t, stderr, "conflicting values")
	assert.Contains(t, stderr, "values.media.test")

	// Anti-regression: Assert flattened shape does NOT exist
	// (Checking for the literal dash prefix that cue/errors prints when flattened)
	assert.NotContains(t, stderr, "ERRO render failed: - ")
}

func TestE2E_ModuleVet_Output(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	testdataDir := filepath.Join(cwd, "testdata", "vet-errors")

	tmpDir, err := os.MkdirTemp("", "e2e-mod-vet-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	_, stderr, err := runOPM(t, tmpDir, "mod", "vet",
		filepath.Join(testdataDir, "module"),
		"-f", filepath.Join(testdataDir, "instance", "values.cue"))

	// Assert exit code 2
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode())

	// Assert grouped shape exists
	assert.Contains(t, stderr, "values do not satisfy #config: 2 issues")
	assert.Contains(t, stderr, "field not allowed")
	assert.Contains(t, stderr, "values.test")
	assert.Contains(t, stderr, "conflicting values")

	// Anti-regression: Assert flattened shape does NOT exist
	assert.NotContains(t, stderr, "ERRO values do not satisfy #config: - ")

	// Cheap failure first: a #config violation stops vet before any
	// platform is resolved or generated.
	assert.NotContains(t, stderr, "platform:")
}

// TestE2E_ModuleVet_RendersAgainstModuleDeps vets a module with components
// and a catalog pin from a HOME holding no platform/: after the #config
// check, vet renders against a platform generated from the module's own deps
// and reports each rendered object and the summary, without printing
// manifests.
func TestE2E_ModuleVet_RendersAgainstModuleDeps(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	modPath := filepath.Join(repoRoot, "tests", "fixtures", "modules", "podinfo")
	if _, statErr := os.Stat(modPath); statErr != nil {
		t.Skipf("tests/fixtures/modules/podinfo not available: %v", statErr)
	}

	customHome := seedRenderHome(t)

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), customHome, 180*time.Second, "module", "vet", modPath, "--instance-name", "e2e-podinfo")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stderr, "Module config valid")
	assert.Contains(t, stderr, "platform: module deps (opmodel.dev/catalogs/opm@v4 v4.0.1; generated module "+filepath.Join(customHome, ".opm", "cache", "platforms"))
	assert.Contains(t, stderr, "Deployment")
	assert.Regexp(t, `Module valid \([1-9][0-9]* resources\)`, stderr)
	assert.NotContains(t, stderr, "version skew")
	assert.Empty(t, stdout, "vet prints no manifests")
}

// TestE2E_ModuleVet_OpenDebugValues asserts that a module whose debugValues
// is left open (`_`) against a #config field with no default is refused as a
// #config violation through the standard grouped block: the kernel's
// concreteness check on the merged value names the incomplete field at its
// schema position. No per-input "not concrete" pre-check runs any more.
func TestE2E_ModuleVet_OpenDebugValues(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	testdataDir := filepath.Join(cwd, "testdata", "vet-errors")

	tmpDir, err := os.MkdirTemp("", "e2e-mod-vet-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	_, stderr, err := runOPM(t, tmpDir, "mod", "vet",
		filepath.Join(testdataDir, "open-debug-values"))

	// Assert exit code 2
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode())

	// Assert the standard grouped block names the incomplete field
	assert.Contains(t, stderr, "values do not satisfy #config: 1 issue")
	assert.Contains(t, stderr, "incomplete value int")
	assert.Contains(t, stderr, "values.replicas")
	assert.Contains(t, stderr, "> module.cue:")

	// Anti-regression: the retired per-input pre-check's wording is gone
	assert.NotContains(t, stderr, "not concrete")
}

// TestE2E_ModuleVet_OpenDebugValuesRefusedAtSynthesis vets a module whose
// #config gives every field a default and whose debugValues is left open:
// the #config check passes, then the render synthesizes the instance and
// refuses the open values, the verdict `module build` reaches for the same
// input.
func TestE2E_ModuleVet_OpenDebugValuesRefusedAtSynthesis(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	modDir := filepath.Join(t.TempDir(), "open-defaults")
	require.NoError(t, os.CopyFS(modDir, os.DirFS(filepath.Join(repoRoot, "tests", "fixtures", "valid", "simple-module"))))
	modFile := filepath.Join(modDir, "module.cue")
	src, err := os.ReadFile(modFile)
	require.NoError(t, err)
	open := strings.Replace(string(src), "debugValues: {}", "", 1)
	require.NotEqual(t, string(src), open, "the fixture declares debugValues: {}")
	require.NoError(t, os.WriteFile(modFile, []byte(open), 0o600))

	_, stderr, err := runOPMWithEnv(t, t.TempDir(), seedRenderHome(t), 180*time.Second, "module", "vet", modDir)

	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode())
	assert.Contains(t, stderr, "Module config valid")
	assert.Contains(t, stderr, "incomplete value")
	assert.NotContains(t, stderr, "Module valid (")
}

// TestE2E_ModuleVet_ValuesFilesWithoutConfig asserts that values files
// supplied to a module that declares no #config are refused, the verdict
// build reaches for the same input, instead of a vacuous "Values satisfy
// #config" pass.
func TestE2E_ModuleVet_ValuesFilesWithoutConfig(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	testdataDir := filepath.Join(cwd, "testdata", "vet-errors")

	tmpDir, err := os.MkdirTemp("", "e2e-mod-vet-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	_, stderr, err := runOPM(t, tmpDir, "mod", "vet",
		filepath.Join(testdataDir, "no-config"),
		"-f", filepath.Join(testdataDir, "instance", "values.cue"))

	// Assert exit code 2
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode())

	assert.Contains(t, stderr, "module does not define #config")
	assert.NotContains(t, stderr, "Values satisfy #config")
}
