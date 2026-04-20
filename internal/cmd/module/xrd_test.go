package modulecmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/config"
)

func TestNewModuleXRDCmd(t *testing.T) {
	cmd := NewModuleXRDCmd(&config.GlobalConfig{})

	assert.Equal(t, "xrd [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	groupFlag := cmd.Flags().Lookup("group")
	require.NotNil(t, groupFlag)
	assert.Equal(t, "module.opmodel.dev", groupFlag.DefValue)

	scopeFlag := cmd.Flags().Lookup("scope")
	require.NotNil(t, scopeFlag)
	assert.Equal(t, "Namespaced", scopeFlag.DefValue)

	outputFlag := cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag)
	assert.Equal(t, "o", outputFlag.Shorthand)
	assert.Equal(t, "yaml", outputFlag.DefValue)

	compFunctionFlag := cmd.Flags().Lookup("comp-function")
	require.NotNil(t, compFunctionFlag)
	assert.Equal(t, "function-opm", compFunctionFlag.DefValue)

	compStepFlag := cmd.Flags().Lookup("comp-step")
	require.NotNil(t, compStepFlag)
	assert.Equal(t, "render-opm-module", compStepFlag.DefValue)

	compInputAPIFlag := cmd.Flags().Lookup("comp-input-api")
	require.NotNil(t, compInputAPIFlag)
	assert.Equal(t, "template.fn.crossplane.io/v1beta1", compInputAPIFlag.DefValue)
}

// TestModXRD_SimpleModule runs the command against the simple-module fixture
// and verifies both the XRD and the matching Composition are emitted as a
// multi-document YAML stream. End-to-end coverage of flag → load → BuildXRD
// + BuildComposition → output wiring; the schema/name details are exercised
// by pkg/k8sgen tests.
func TestModXRD_SimpleModule(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("Test fixture not found:", fixtureDir)
	}

	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_REGISTRY")

	cfg := &config.GlobalConfig{CueContext: cuecontext.New()}
	cmd := NewModuleXRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir})

	stdout := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	// Derived from simple-module's metadata.name ("simple-module"), version ("0.1.0"):
	//   kind = SimpleModule, plural = simplemodules, version = v1alpha1.
	assert.Contains(t, stdout, "apiVersion: apiextensions.crossplane.io/v2")
	assert.Contains(t, stdout, "kind: CompositeResourceDefinition")
	assert.Contains(t, stdout, "name: simplemodules.module.opmodel.dev")
	assert.Contains(t, stdout, "kind: SimpleModule")
	assert.Contains(t, stdout, "listKind: SimpleModuleList")
	assert.Contains(t, stdout, "scope: Namespaced")
	assert.Contains(t, stdout, "name: v1alpha1")
	assert.Contains(t, stdout, "referenceable: true")
	assert.Contains(t, stdout, "openAPIV3Schema")

	// Composition half of the stream.
	assert.Contains(t, stdout, "apiVersion: apiextensions.crossplane.io/v1")
	assert.Contains(t, stdout, "kind: Composition")
	assert.Contains(t, stdout, "compositeTypeRef:")
	assert.Contains(t, stdout, "apiVersion: module.opmodel.dev/v1alpha1")
	assert.Contains(t, stdout, "mode: Pipeline")
	assert.Contains(t, stdout, "step: render-opm-module")
	assert.Contains(t, stdout, "name: function-opm")
	assert.Contains(t, stdout, "path: example.com/modules/simple-module")
	assert.Contains(t, stdout, "version: 0.1.0")

	// Multi-document YAML: two `---` separators (leading separator + between docs).
	assert.GreaterOrEqual(t, strings.Count(stdout, "\n---\n"), 1,
		"output must contain a YAML document separator between the XRD and the Composition")

	// Provenance labels and annotations link both manifests back to the module.
	assert.Contains(t, stdout, "app.kubernetes.io/managed-by: opm-cli")
	assert.Contains(t, stdout, "module.opmodel.dev/name: simple-module")
	assert.Contains(t, stdout, "module.opmodel.dev/version: 0.1.0")
	assert.Contains(t, stdout, "module.opmodel.dev/path: example.com/modules")
	assert.Contains(t, stdout, "module.opmodel.dev/fqn: example.com/modules/simple-module:0.1.0")
	assert.NotContains(t, stdout, "module.opmodel.dev/description")
	assert.NotContains(t, stdout, "module.opmodel.dev/uuid")
}

func TestModXRD_CustomGroup(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("Test fixture not found:", fixtureDir)
	}

	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_REGISTRY")

	cfg := &config.GlobalConfig{CueContext: cuecontext.New()}
	cmd := NewModuleXRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir, "--group", "example.com"})

	stdout := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	// The group flag changes both the XRD metadata.name and the
	// Composition's compositeTypeRef.apiVersion — verify both.
	assert.Contains(t, stdout, "name: simplemodules.example.com")
	assert.Contains(t, stdout, "group: example.com")
	assert.Contains(t, stdout, "apiVersion: example.com/v1alpha1")
}

func TestModXRD_ScopeCluster(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("Test fixture not found:", fixtureDir)
	}

	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_REGISTRY")

	cfg := &config.GlobalConfig{CueContext: cuecontext.New()}
	cmd := NewModuleXRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir, "--scope", "Cluster"})

	stdout := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	assert.Contains(t, stdout, "scope: Cluster")
	assert.NotContains(t, stdout, "scope: Namespaced")
}

func TestModXRD_InvalidScope(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("Test fixture not found:", fixtureDir)
	}

	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_REGISTRY")

	cfg := &config.GlobalConfig{CueContext: cuecontext.New()}
	cmd := NewModuleXRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir, "--scope", "Bogus"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid XRD scope")
}

// TestModXRD_CompositionFlagOverrides exercises the --comp-function /
// --comp-step / --comp-input-api flags end-to-end; the pkg/k8sgen layer
// owns the actual substitution, but a command-layer smoke test catches any
// flag-wiring regressions.
func TestModXRD_CompositionFlagOverrides(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("Test fixture not found:", fixtureDir)
	}

	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_REGISTRY")

	cfg := &config.GlobalConfig{CueContext: cuecontext.New()}
	cmd := NewModuleXRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		fixtureDir,
		"--comp-function", "function-opm-fork",
		"--comp-step", "render-custom",
		"--comp-input-api", "opm.fn.crossplane.io/v1alpha1",
	})

	stdout := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	assert.Contains(t, stdout, "name: function-opm-fork")
	assert.Contains(t, stdout, "step: render-custom")
	assert.Contains(t, stdout, "apiVersion: opm.fn.crossplane.io/v1alpha1")
	// The defaults should no longer appear as ancillary text.
	assert.NotContains(t, stdout, "name: function-opm\n")
	assert.NotContains(t, stdout, "step: render-opm-module")
}

func TestModXRD_JSONOutput(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("Test fixture not found:", fixtureDir)
	}

	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_REGISTRY")

	cfg := &config.GlobalConfig{CueContext: cuecontext.New()}
	cmd := NewModuleXRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir, "-o", "json"})

	stdout := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	// Both manifests must appear in the JSON stream.
	assert.Contains(t, stdout, `"apiVersion": "apiextensions.crossplane.io/v2"`)
	assert.Contains(t, stdout, `"kind": "CompositeResourceDefinition"`)
	assert.Contains(t, stdout, `"apiVersion": "apiextensions.crossplane.io/v1"`)
	assert.Contains(t, stdout, `"kind": "Composition"`)
}

func TestModXRD_InvalidPath(t *testing.T) {
	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cfg := &config.GlobalConfig{CueContext: cuecontext.New()}
	cmd := NewModuleXRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"/does/not/exist"})

	err := cmd.Execute()
	require.Error(t, err)
}

func TestModXRD_InvalidOutputFormat(t *testing.T) {
	cfg := &config.GlobalConfig{CueContext: cuecontext.New()}
	cmd := NewModuleXRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{".", "-o", "xml"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}
