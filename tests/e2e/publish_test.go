package e2e

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cuelabs.dev/go/oci/ociregistry/ocimem"
	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// publishModuleFixture writes a minimal, import-free module tree that
// conforms to core's #IdentityPackage, mutated per test.
func publishModuleFixture(t *testing.T, mutate func(map[string]string)) string {
	t.Helper()
	files := map[string]string{
		"cue.mod/module.cue": `module: "example.com/modules/demo@v1"
language: version: "v0.17.0"
source: kind: "self"
`,
		"identity/identity.cue": `package identity

ModulePath: "example.com/modules/demo@v1"
Version:    "1.2.0"
`,
		"module.cue": `package demo

metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v1"
	version:    "1.2.0"
}
`,
	}
	if mutate != nil {
		mutate(files)
	}
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

// publishRegistryEnv starts an in-process registry for the fixture domain and
// returns the OPM_REGISTRY value routing example.com at it while the core
// schema still resolves from GHCR.
func publishRegistryEnv(t *testing.T) string {
	t.Helper()
	reg, err := modregistrytest.NewServer(ocimem.NewWithConfig(&ocimem.Config{ImmutableTags: true}), nil)
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return "OPM_REGISTRY=opmodel.dev=ghcr.io/open-platform-model,example.com=" + reg.Host() + "+insecure,registry.cue.works"
}

// runOPMPublish runs the opm binary with extra environment entries, capturing
// stdout and stderr independently regardless of exit status.
func runOPMPublish(t *testing.T, workDir string, extraEnv []string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, opmBinary, args...)
	cmd.Dir = workDir
	cmd.Env = append(append(os.Environ(), "HOME="+homeDir), extraEnv...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

// skipWithoutCoreSchema skips when the core v2 schema cannot be resolved
// (offline, no warm cache) — matching the repo's other registry-backed tests.
func skipWithoutCoreSchema(t *testing.T, stderr string) {
	t.Helper()
	if strings.Contains(stderr, "loading core schema") {
		t.Skipf("core v2 schema unavailable (registry/cache): %s", stderr)
	}
}

func TestE2E_ModulePublish_DryRunGO(t *testing.T) {
	dir := publishModuleFixture(t, nil)
	env := []string{publishRegistryEnv(t)}

	stdout, stderr, err := runOPMPublish(t, dir, env, "module", "publish", "--dry-run")
	skipWithoutCoreSchema(t, stderr)
	require.NoError(t, err, "stdout: %s\nstderr: %s", stdout, stderr)

	// The plan is the dry run: every coordinate derived from the artifact.
	assert.Contains(t, stdout, "declaredPath    example.com/modules/demo@v1")
	assert.Contains(t, stdout, "registryRepo    example.com/modules/demo")
	assert.Contains(t, stdout, "tag             v1.2.0")
	assert.Contains(t, stdout, "GO — pushing example.com/modules/demo:v1.2.0")
}

func TestE2E_ModulePublish_DryRunRefused(t *testing.T) {
	dir := publishModuleFixture(t, func(files map[string]string) {
		// Two defects in one tree: cue.mod disagrees with identity (msg 2)
		// and a local override is present (msg 4b). One invocation must
		// report both.
		files["cue.mod/module.cue"] = `module: "example.com/modules/other@v1"
language: version: "v0.17.0"
source: kind: "self"
deps: "example.com/dep@v1": v: "v1.4.0"
`
		files["cue.mod/local-module.cue"] = `deps: "example.com/dep@v1": replaceWith: "../dep"
`
	})
	env := []string{publishRegistryEnv(t)}

	stdout, stderr, err := runOPMPublish(t, dir, env, "module", "publish", "--dry-run")
	skipWithoutCoreSchema(t, stderr)

	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode(), "stdout: %s\nstderr: %s", stdout, stderr)

	assert.Contains(t, stdout, "REFUSED — 2 refusals")

	// Msg 2's aligned two-value columns: both paths, both declaring files.
	assert.Contains(t, stderr, "disagrees with itself about where it lives")
	assert.Contains(t, stderr, "declared  example.com/modules/demo@v1   identity/identity.cue:")
	assert.Contains(t, stderr, "cue.mod   example.com/modules/other@v1  cue.mod/module.cue:")

	// Msg 4b names the waiver flag and the superseding registry version.
	assert.Contains(t, stderr, "local-module.cue is present")
	assert.Contains(t, stderr, "--skip-override-check")
	assert.Contains(t, stderr, "would resolve to v1.4.0 when published")
}

func TestE2E_ModulePublish_PushAndAlreadyPublished(t *testing.T) {
	dir := publishModuleFixture(t, nil)
	env := []string{publishRegistryEnv(t)}

	// First publish pushes for real (to the in-process registry).
	stdout, stderr, err := runOPMPublish(t, dir, env, "module", "publish")
	skipWithoutCoreSchema(t, stderr)
	require.NoError(t, err, "stdout: %s\nstderr: %s", stdout, stderr)

	// Second publish refuses: a published tag names fixed bytes permanently.
	stdout, stderr, err = runOPMPublish(t, dir, env, "module", "publish")
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode(), "stdout: %s\nstderr: %s", stdout, stderr)
	assert.Contains(t, stderr, "already holds v1.2.0")
	assert.Contains(t, stderr, "opm module version set 1.2.1")
}
