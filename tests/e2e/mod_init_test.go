// Package e2e provides end-to-end tests for the OPM CLI.
package e2e

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"cuelabs.dev/go/oci/ociregistry/ocimem"
	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var opmBinary string
var homeDir string

func TestMain(m *testing.M) {
	// Build the binary once for all tests
	tmpDir, err := os.MkdirTemp("", "opm-e2e-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}

	opmBinary = filepath.Join(tmpDir, "opm")
	homeDir = tmpDir

	// Build the binary
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	cmd := exec.CommandContext(ctx, "go", "build", "-o", opmBinary, "../../cmd/opm")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cancel()
		os.RemoveAll(tmpDir)
		panic("failed to build opm binary: " + err.Error())
	}
	cancel() // Call cancel explicitly before os.Exit

	// Write a stub config.cue directly — scalar data only, matching the
	// post-D39 schema (no providers, no imports, no cue.mod).
	opmDir := filepath.Join(homeDir, ".opm")
	if err := os.MkdirAll(opmDir, 0o755); err != nil {
		os.RemoveAll(tmpDir)
		panic("failed to create .opm dir: " + err.Error())
	}

	dummyConfig := `package config
config: {
	registry: "localhost:5000"
	kubernetes: {
		context:    "kind-opm-dev"
		kubeconfig: "~/.kube/config"
		namespace:  "default"
	}
}`
	if err := os.WriteFile(filepath.Join(opmDir, "config.cue"), []byte(dummyConfig), 0o644); err != nil {
		os.RemoveAll(tmpDir)
		panic("failed to write dummy config.cue: " + err.Error())
	}

	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

// runOPM runs the opm binary with the given arguments and returns output
func runOPM(t *testing.T, workDir string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, opmBinary, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "HOME="+homeDir)

	stdoutBytes, err := cmd.Output()
	var stderrBytes []byte
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderrBytes = exitErr.Stderr
	}

	return string(stdoutBytes), string(stderrBytes), err
}

// templatesRegistryEnv starts an in-process registry and returns the
// OPM_REGISTRY value routing the reserved templates segment and the
// scaffold's own example.com domain at it, while the core schema and the
// templates' catalog/core deps still resolve from GHCR — the hermetic
// inversion: the shortcut's expansion resolves against a registry this test
// owns.
func templatesRegistryEnv(t *testing.T) string {
	t.Helper()
	reg, err := modregistrytest.NewServer(ocimem.NewWithConfig(&ocimem.Config{ImmutableTags: true}), nil)
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return "OPM_REGISTRY=opmodel.dev/templates=" + reg.Host() + "+insecure" +
		",example.com=" + reg.Host() + "+insecure" +
		",opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
}

// repoTemplateDir resolves a template tree in this repository.
func repoTemplateDir(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "templates", name))
	require.NoError(t, err)
	return abs
}

// TestE2E_ModInit_ThenVet is the hermetic scaffold round-trip: publish the
// repo's real standard template into an in-process registry, init from it
// via the bare-word shortcut, and require the scaffold to pass vet and a
// publish dry-run (GO) with the template's identity appearing nowhere in it.
func TestE2E_ModInit_ThenVet(t *testing.T) {
	env := []string{templatesRegistryEnv(t)}
	tmpDir := t.TempDir()

	// Publish the template through the real pipeline — the same act the
	// release CI performs.
	stdout, stderr, err := runOPMPublish(t, tmpDir, env, "module", "publish", repoTemplateDir(t, "standard"))
	skipWithoutCoreSchema(t, stderr)
	require.NoError(t, err, "publishing the template failed\nstdout: %s\nstderr: %s", stdout, stderr)

	// Scaffold from it by shortcut. The bare word expands into the reserved
	// segment, which the env routes at the in-process registry.
	stdout, stderr, err = runOPMPublish(t, tmpDir, env, "mod", "init", "example.com/modules/my_app@v0", "standard")
	require.NoError(t, err, "init failed\nstdout: %s\nstderr: %s", stdout, stderr)
	assert.Contains(t, stdout, "opm module vet", "init points at vet")

	moduleDir := filepath.Join(tmpDir, "my_app")
	require.DirExists(t, moduleDir)

	// The template's identity never leaks into the scaffold: no file carries
	// its path, and the package clauses bind the new leaf.
	var files []string
	require.NoError(t, filepath.Walk(moduleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		files = append(files, path)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "opmodel.dev/templates",
			"template identity leaked into %s", path)
		return nil
	}))
	require.NotEmpty(t, files)

	moduleCue, err := os.ReadFile(filepath.Join(moduleDir, "module.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(moduleCue), "package my_app")
	componentsCue, err := os.ReadFile(filepath.Join(moduleDir, "components.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(componentsCue), "package my_app")

	idCue, err := os.ReadFile(filepath.Join(moduleDir, "identity", "identity.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(idCue), `ModulePath: "example.com/modules/my_app@v0"`)
	assert.Contains(t, string(idCue), `Version: "0.1.0"`, "version is set to the initial version as a plain literal")
	assert.NotContains(t, string(idCue), `*"`, "no default arm: the kernel's loader gate needs a concrete literal")
	assert.NotContains(t, string(idCue), "#VersionType", "no local #VersionType: core's #IdentityPackage constrains Version")

	// The scaffold passes vet unmodified…
	stdout, stderr, err = runOPMPublish(t, moduleDir, env, "module", "vet")
	require.NoError(t, err, "vet failed\nstdout: %s\nstderr: %s", stdout, stderr)

	// …and every publish gate short of the push: dry-run reports GO.
	stdout, stderr, err = runOPMPublish(t, moduleDir, env, "module", "publish", "--dry-run")
	require.NoError(t, err, "dry-run failed\nstdout: %s\nstderr: %s", stdout, stderr)
	assert.Contains(t, stdout, "GO — pushing example.com/modules/my_app:v0.1.0")

	// …and the kernel's loader accepts it: the scaffold's identity is a plain
	// literal, so metadata.version is concrete at the shape gate. A build that
	// fails for any reason other than the gate (registry, platform) is skipped,
	// but a gate refusal naming metadata.version is a hard failure.
	buildHome := makeBuildHome(t)
	defer os.RemoveAll(buildHome)
	stdout, stderr, err = runOPMWithEnv(t, moduleDir, buildHome, 120*time.Second, "module", "build", moduleDir, "--name", "my-app")
	assert.NotContains(t, stderr, `"metadata.version"`, "scaffold refused at the loader shape gate")
	if err != nil {
		t.Skipf("opm module build failed past the shape gate (likely registry/platform unavailable): err=%v stderr=%s", err, stderr)
	}
	assert.NotEmpty(t, stdout, "expected manifest output on stdout")
}

// TestE2E_ModInit_TypoFailsInsideTheSegment pins the no-fallback contract: a
// bare word that names no published template refuses naming the expanded
// path — inside the reserved segment, never elsewhere.
func TestE2E_ModInit_TypoFailsInsideTheSegment(t *testing.T) {
	env := []string{templatesRegistryEnv(t)}
	tmpDir := t.TempDir()

	stdout, stderr, err := runOPMPublish(t, tmpDir, env, "mod", "init", "example.com/modules/my_app@v0", "standrad")
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode(), "stdout: %s\nstderr: %s", stdout, stderr)
	assert.Contains(t, stderr, "opmodel.dev/templates/standrad", "the refusal names the expansion")
	assert.NoDirExists(t, filepath.Join(tmpDir, "my_app"))
}

// TestE2E_ModInit_OfflineRefusesHonestly pins the offline scenario: with the
// reserved segment routed at an unreachable registry and nothing cached, init
// refuses naming the expanded template path and the registry it tried —
// exit 3, no directory created, no embedded fallback.
func TestE2E_ModInit_OfflineRefusesHonestly(t *testing.T) {
	// 127.0.0.1:1 answers nothing; the mapping makes the failure hermetic.
	env := []string{"OPM_REGISTRY=opmodel.dev/templates=127.0.0.1:1+insecure,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"}
	tmpDir := t.TempDir()

	stdout, stderr, err := runOPMPublish(t, tmpDir, env, "mod", "init", "example.com/modules/my_app@v0", "standard")
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 3, exitErr.ExitCode(), "stdout: %s\nstderr: %s", stdout, stderr)
	assert.Contains(t, stderr, "opmodel.dev/templates/standard", "the refusal names the expansion")
	assert.Contains(t, stderr, "127.0.0.1:1", "the refusal names the registry tried")
	assert.NoDirExists(t, filepath.Join(tmpDir, "my_app"))
}

// TestE2E_ModInit_FromClonesAnyPublishedModule pins --from as a first-class
// clone source: publish a derivation-complete donor into the in-process
// registry, clone it to a new identity, and require the donor's identity to
// appear nowhere in the result.
func TestE2E_ModInit_FromClonesAnyPublishedModule(t *testing.T) {
	env := []string{templatesRegistryEnv(t)}
	tmpDir := t.TempDir()

	donor := publishModuleFixture(t, func(files map[string]string) {
		// The donor derives its metadata from its identity package (D12) —
		// what makes a clone follow the rewritten identity. kind is required
		// by the registry loader's shape gate.
		files["module.cue"] = `package demo

import (
	"strings"

	id "example.com/modules/demo/identity"
)

kind: "Module"

metadata: {
	_segments: strings.Split(strings.SplitN(id.ModulePath, "@", 2)[0], "/")
	name:       _segments[len(_segments)-1]
	modulePath: id.ModulePath
	version:    "\(id.Version)"
}
`
	})

	stdout, stderr, err := runOPMPublish(t, donor, env, "module", "publish")
	skipWithoutCoreSchema(t, stderr)
	require.NoError(t, err, "publishing the donor failed\nstdout: %s\nstderr: %s", stdout, stderr)

	stdout, stderr, err = runOPMPublish(t, tmpDir, env, "mod", "init",
		"example.com/modules/clone@v0", "--from", "example.com/modules/demo@v1")
	require.NoError(t, err, "clone failed\nstdout: %s\nstderr: %s", stdout, stderr)

	cloneDir := filepath.Join(tmpDir, "clone")
	require.DirExists(t, cloneDir)

	require.NoError(t, filepath.Walk(cloneDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "modules/demo", "donor identity leaked into %s", path)
		return nil
	}))

	moduleCue, err := os.ReadFile(filepath.Join(cloneDir, "module.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(moduleCue), "package clone")
	idCue, err := os.ReadFile(filepath.Join(cloneDir, "identity", "identity.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(idCue), `ModulePath: "example.com/modules/clone@v0"`)
	assert.Contains(t, string(idCue), `"0.1.0"`)
}

// TestE2E_ModInit_NonDerivingDonorRefuses pins the mismatch half: a donor
// whose metadata carries literals instead of the D12 derivation cannot be
// re-identified — init refuses naming the defect and leaves nothing behind.
func TestE2E_ModInit_NonDerivingDonorRefuses(t *testing.T) {
	env := []string{templatesRegistryEnv(t)}
	tmpDir := t.TempDir()

	// Distinct coordinates from the deriving donor above: CUE's module cache
	// is shared across the suite's single temp HOME and keyed by
	// path@version, so republishing different bytes under the same
	// coordinates would serve the sibling test's cached module (the warm-
	// cache trap). kind is still needed for the acquire's shape gate.
	donor := publishModuleFixture(t, func(files map[string]string) {
		files["cue.mod/module.cue"] = `module: "example.com/modules/litdonor@v1"
language: version: "v0.17.0"
source: kind: "self"
`
		files["identity/identity.cue"] = `package identity

ModulePath: "example.com/modules/litdonor@v1"
Version:    "1.2.0"
`
		files["module.cue"] = `package litdonor

kind: "Module"

metadata: {
	name:       "litdonor"
	modulePath: "example.com/modules/litdonor@v1"
	version:    "1.2.0"
}
`
	})

	stdout, stderr, err := runOPMPublish(t, donor, env, "module", "publish")
	skipWithoutCoreSchema(t, stderr)
	require.NoError(t, err, "publishing the donor failed\nstdout: %s\nstderr: %s", stdout, stderr)

	stdout, stderr, err = runOPMPublish(t, tmpDir, env, "mod", "init",
		"example.com/modules/clone@v0", "--from", "example.com/modules/litdonor@v1")
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode(), "stdout: %s\nstderr: %s", stdout, stderr)
	assert.Contains(t, stderr, "does not derive metadata", "stdout: %s\nstderr: %s", stdout, stderr)
	assert.NoDirExists(t, filepath.Join(tmpDir, "clone"), "a refused clone leaves nothing behind")
}

func TestE2E_ModInit_ExistingNonModuleDirRefuses(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "my_app"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "my_app", "notes.txt"), []byte("hi"), 0o644))

	_, _, err := runOPM(t, tmpDir, "mod", "init", "example.com/modules/my_app@v0")
	require.Error(t, err)
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		assert.Equal(t, 2, exitErr.ExitCode(), "expected exit code 2 for validation error")
	}
}

func TestE2E_TemplateList(t *testing.T) {
	stdout, stderr, err := runOPM(t, t.TempDir(), "module", "template", "list")
	require.NoError(t, err, "stderr: %s", stderr)
	for _, name := range []string{"minimal", "standard", "advanced"} {
		assert.Contains(t, stdout, name)
	}
}

func TestE2E_Version(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "e2e-version-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	stdout, stderr, err := runOPM(t, tmpDir, "version")
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Contains(t, stdout, "opm version")
	assert.Contains(t, stdout, "CUE SDK")
}

func TestE2E_Help(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "e2e-help-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	stdout, stderr, err := runOPM(t, tmpDir, "--help")
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Contains(t, stdout, "mod")
	assert.Contains(t, stdout, "config")
	assert.Contains(t, stdout, "version")
}
