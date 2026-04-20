package modulecmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/config"
)

func TestNewModuleCRDCmd(t *testing.T) {
	cmd := NewModuleCRDCmd(&config.GlobalConfig{})

	assert.Equal(t, "crd [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	groupFlag := cmd.Flags().Lookup("group")
	require.NotNil(t, groupFlag)
	assert.Equal(t, "module.opmodel.dev", groupFlag.DefValue)

	outputFlag := cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag)
	assert.Equal(t, "o", outputFlag.Shorthand)
	assert.Equal(t, "yaml", outputFlag.DefValue)
}

// TestModCRD_SimpleModule runs the command against the simple-module fixture
// and verifies the YAML on stdout looks like a CRD. End-to-end coverage of
// flag → load → BuildCRD → output wiring; the schema/name details are
// exercised by pkg/k8sgen tests.
func TestModCRD_SimpleModule(t *testing.T) {
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
	cmd := NewModuleCRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir})

	stdout := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	// Derived from simple-module's metadata.name ("simple-module"), version ("0.1.0"):
	//   kind = SimpleModule, plural = simplemodules, version = v1alpha1.
	assert.Contains(t, stdout, "apiVersion: apiextensions.k8s.io/v1")
	assert.Contains(t, stdout, "kind: CustomResourceDefinition")
	assert.Contains(t, stdout, "name: simplemodules.module.opmodel.dev")
	assert.Contains(t, stdout, "kind: SimpleModule")
	assert.Contains(t, stdout, "listKind: SimpleModuleList")
	assert.Contains(t, stdout, "scope: Namespaced")
	assert.Contains(t, stdout, "name: v1alpha1")
	assert.Contains(t, stdout, "openAPIV3Schema")

	// Provenance labels and annotations link the CRD back to the module.
	assert.Contains(t, stdout, "app.kubernetes.io/managed-by: opm-cli")
	assert.Contains(t, stdout, "module.opmodel.dev/name: simple-module")
	assert.Contains(t, stdout, "module.opmodel.dev/version: 0.1.0")
	// simple-module declares modulePath and fqn but not description or uuid.
	assert.Contains(t, stdout, "module.opmodel.dev/path: example.com/modules")
	assert.Contains(t, stdout, "module.opmodel.dev/fqn: example.com/modules/simple-module:0.1.0")
	assert.NotContains(t, stdout, "module.opmodel.dev/description")
	assert.NotContains(t, stdout, "module.opmodel.dev/uuid")
}

func TestModCRD_CustomGroup(t *testing.T) {
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
	cmd := NewModuleCRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir, "--group", "example.com"})

	stdout := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	assert.Contains(t, stdout, "name: simplemodules.example.com")
	assert.Contains(t, stdout, "group: example.com")
	// The CRD's own group should not fall back to the default. Provenance
	// labels on metadata.labels may still reference module.opmodel.dev/*,
	// so check group specifically rather than the raw string.
	assert.NotContains(t, stdout, "group: opmodel.dev")
}

func TestModCRD_JSONOutput(t *testing.T) {
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
	cmd := NewModuleCRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir, "-o", "json"})

	stdout := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	assert.Contains(t, stdout, `"apiVersion": "apiextensions.k8s.io/v1"`)
	assert.Contains(t, stdout, `"kind": "CustomResourceDefinition"`)
}

func TestModCRD_InvalidPath(t *testing.T) {
	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cfg := &config.GlobalConfig{CueContext: cuecontext.New()}
	cmd := NewModuleCRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"/does/not/exist"})

	err := cmd.Execute()
	require.Error(t, err)
}

func TestModCRD_InvalidOutputFormat(t *testing.T) {
	cfg := &config.GlobalConfig{CueContext: cuecontext.New()}
	cmd := NewModuleCRDCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{".", "-o", "xml"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

// captureStdout replaces os.Stdout with a pipe for the duration of fn and
// returns whatever was written. WriteManifestOutput writes directly to
// os.Stdout, so this is the narrowest way to assert on its output.
// Tests that call this must not run with t.Parallel() — os.Stdout is
// process-global.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	done := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.Bytes()
	}()

	defer func() {
		os.Stdout = orig
	}()

	fn()

	require.NoError(t, w.Close())
	out := <-done
	return string(out)
}
