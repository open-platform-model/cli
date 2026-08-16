package e2e

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// catalogFixture writes a minimal catalog tree that conforms to the real core
// v2 gates: identity authors the two release-owned fields, the root derives
// its metadata literally, and the one member files per D49 with an authored
// fqn the FQN gate accepts. No member imports core — the gates read shape,
// not types — so the tree publishes hermetically.
func catalogFixture(t *testing.T) string {
	t.Helper()
	files := map[string]string{
		"cue.mod/module.cue": `module: "example.com/catalogs/demo@v1"
language: version: "v0.17.0"
source: kind: "self"
`,
		"identity/identity.cue": `package identity

ModulePath: "example.com/catalogs/demo@v1"
Version:    "1.2.0"
`,
		"catalog.cue": `package democat

metadata: {
	name:       "demo"
	modulePath: "example.com/catalogs/demo@v1"
	version:    "1.2.0"
}
`,
		"resources/v1beta1/thing.cue": `package v1beta1

#ThingResource: {
	kind: "Resource"
	metadata: {
		modulePath:     "example.com/catalogs/demo/resources/v1beta1"
		name:           "thing"
		apiVersion:     "v1beta1"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/resources/thing@v1beta1"
	}
	spec: {
		size: int
	}
}
`,
	}
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

// e2eCUECache hands the opm binary an isolated CUE_CACHE_DIR env entry: the
// shared cache is keyed by path@version alone, and the fixture coordinate
// repeats across tests with different bytes. The extract dir is read-only,
// so cleanup chmods before removing.
func e2eCUECache(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "opm-e2e-cue-cache-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = filepath.WalkDir(dir, func(p string, _ fs.DirEntry, err error) error {
			if err == nil {
				_ = os.Chmod(p, 0o700)
			}
			return nil
		})
		_ = os.RemoveAll(dir)
	})
	return "CUE_CACHE_DIR=" + dir
}

func TestE2E_CatalogRegistryCheck_HelpCarriesAidFraming(t *testing.T) {
	stdout, stderr, err := runOPMPublish(t, t.TempDir(), nil, "catalog", "registry", "check", "--help")
	require.NoError(t, err, "stderr: %s", stderr)

	// D35's sentence, verbatim — the aid-not-guarantee distinction belongs
	// where a catalog author meets it.
	assert.Contains(t, stdout, "This check is an aid, not a guarantee: nothing requires it to have been run, and enforcement exists only at publish — it does not make an unchecked catalog trustworthy; it makes it checkable.")
	assert.Contains(t, stdout, "Exit codes: 0 clean, 2 findings, 3 registry unreachable.")
	assert.Contains(t, stdout, "--compat")
}

func TestE2E_CatalogRegistryCheck_PublishThenCheckClean(t *testing.T) {
	dir := catalogFixture(t)
	env := []string{publishRegistryEnv(t), e2eCUECache(t)}

	stdout, stderr, err := runOPMPublish(t, dir, env, "catalog", "publish")
	skipWithoutCoreSchema(t, stderr)
	require.NoError(t, err, "stdout: %s\nstderr: %s", stdout, stderr)
	assert.Contains(t, stdout, "member gate")
	assert.Contains(t, stdout, "1 members checked, 0 refused")
	assert.Contains(t, stdout, "compat gate")
	assert.Contains(t, stdout, "GO — pushing example.com/catalogs/demo:v1.2.0")

	stdout, stderr, err = runOPMPublish(t, dir, env,
		"catalog", "registry", "check", "example.com/catalogs/demo@v1.2.0", "--compat")
	require.NoError(t, err, "stdout: %s\nstderr: %s", stdout, stderr)
	assert.Contains(t, stdout, "coordinate      example.com/catalogs/demo@v1.2.0")
	assert.Contains(t, stdout, "declaredPath    example.com/catalogs/demo@v1")
	assert.Contains(t, stdout, "resources/v1beta1")
	assert.Contains(t, stdout, "thing")
	assert.Contains(t, stdout, "CLEAN")
}
