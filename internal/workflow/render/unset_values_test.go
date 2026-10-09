package render

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/modregistrytest"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/instinit"
)

// unsetValuesModule returns the files of the simple-module fixture with its
// #config replaced: note is required and read by a component, other is
// required and read by none, replicas has a default. Its debugValues set
// nothing.
func unsetValuesModule(t *testing.T) map[string][]byte {
	t.Helper()
	fixture := os.DirFS(filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module"))
	files := map[string][]byte{}
	require.NoError(t, fs.WalkDir(fixture, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fixture, p)
		files[p] = data
		return err
	}))
	head, _, found := strings.Cut(string(files["module.cue"]), "// Configuration schema with defaults.")
	require.True(t, found, "the fixture's #config comment moved")
	files["module.cue"] = []byte(head + `#config: {
	replicas: *1 | int
	note:     string
	other:    int
}

debugValues: {}

#components: foo: metadata: {name: "foo", annotations: note: #config.note}
`)
	return files
}

// captureFindings runs fn and returns what it wrote to the process's
// standard error, where the grouped findings of a refused render go.
func captureFindings(t *testing.T, fn func()) string {
	t.Helper()
	stderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = stderr })
	read := make(chan string)
	go func() {
		data, _ := io.ReadAll(r)
		read <- string(data)
	}()
	fn()
	os.Stderr = stderr
	require.NoError(t, w.Close())
	return <-read
}

// assertUnsetValuesNamed holds the report for the module of
// unsetValuesModule: exit 2, two issues, one finding for each unset required
// value at values.<field>, none at the component field that reads note and
// none for the defaulted value.
func assertUnsetValuesNamed(t *testing.T, err error, log, findings string) {
	t.Helper()
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Contains(t, log, "render failed: 2 issues")
	assert.Contains(t, findings, "incomplete value string\n  values.note\n")
	assert.Contains(t, findings, "incomplete value int\n  values.other\n")
	assert.NotContains(t, findings, "components", "the place that reads the value is not a finding")
	assert.NotContains(t, findings, "values.replicas", "a defaulted value is not a finding")
}

// TestFromModule_UnsetRequiredValuesAreNamed holds what a module render
// (opm module build, opm module apply) prints when the module's debugValues
// leave required #config values unset.
func TestFromModule_UnsetRequiredValuesAreNamed(t *testing.T) {
	const registry = "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
	if _, err := config.NewKernel(registry).SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}
	dir := filepath.Join(t.TempDir(), "unset")
	for name, data := range unsetValuesModule(t) {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, data, 0o600))
	}
	logBuf := captureRenderLog(t)

	var err error
	findings := captureFindings(t, func() {
		_, err = FromModule(context.Background(), ModuleOpts{
			ModulePath: dir,
			Config:     &config.GlobalConfig{ConfigPath: filepath.Join(t.TempDir(), "config.cue"), Registry: registry},
			K8sConfig:  &config.ResolvedKubernetesConfig{},
		})
	})

	assertUnsetValuesNamed(t, err, logBuf.String(), findings)
}

// TestFromInstanceFile_UnsetRequiredValuesAreNamed holds the same report for
// an instance package (opm instance build, vet, apply, diff): the package
// `opm instance init` writes for the module, published to a local registry,
// with a values.cue that sets nothing.
func TestFromInstanceFile_UnsetRequiredValuesAreNamed(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed tests")
	}
	const modPath, modVersion = "example.com/modules/simple_module@v0", "v0.1.0"
	served := fstest.MapFS{}
	for name, data := range unsetValuesModule(t) {
		served["example.com_modules_simple_module_"+modVersion+"/"+name] = &fstest.MapFile{Data: data}
	}
	reg, err := modregistrytest.New(served, "")
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	registry := "example.com=" + reg.Host() + "+insecure,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
	if _, err := config.NewKernel(registry).SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}

	mf, err := modfile.Parse(served["example.com_modules_simple_module_"+modVersion+"/cue.mod/module.cue"].Data, "module.cue")
	require.NoError(t, err)
	mod, err := module.NewVersion(modPath, modVersion)
	require.NoError(t, err)
	core, err := module.NewVersion("opmodel.dev/core@v2", mf.Deps["opmodel.dev/core@v2"].Version)
	require.NoError(t, err)
	files, err := instinit.Render(instinit.Input{
		Name:              "hello",
		Namespace:         "default",
		PackageModulePath: instinit.DefaultPackageModulePath("hello"),
		Module:            mod,
		Core:              core,
	})
	require.NoError(t, err)
	dir := filepath.Join(t.TempDir(), "hello")
	require.NoError(t, instinit.Write(context.Background(), dir, files, registry))
	logBuf := captureRenderLog(t)

	findings := captureFindings(t, func() {
		_, err = FromInstanceFile(context.Background(), InstanceFileOpts{
			InstanceFilePath: dir,
			Config:           &config.GlobalConfig{ConfigPath: filepath.Join(t.TempDir(), "config.cue"), Registry: registry},
			K8sConfig:        &config.ResolvedKubernetesConfig{},
		})
	})

	assertUnsetValuesNamed(t, err, logBuf.String(), findings)
}
