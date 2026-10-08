package modulecmd

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
)

func TestNewModuleVetCmd(t *testing.T) {
	cmd := NewModuleVetCmd(&config.GlobalConfig{})

	assert.Equal(t, "vet [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	// Args validation is set to MaximumNArgs(1) but not directly testable
}

func TestNewModuleVetCmd_NoLocalVerboseFlag(t *testing.T) {
	cmd := NewModuleVetCmd(&config.GlobalConfig{})

	// Verify that --verbose is NOT a local flag on this command.
	// It should come from the root persistent flag instead.
	localFlag := cmd.Flags().Lookup("verbose")
	assert.Nil(t, localFlag, "--verbose should not be a local flag (should use root persistent flag)")
}

// TestModVet_ValidModule exercises the module vet path with the simple-module
// fixture (core@v2 line, no instance.cue, empty debugValues, no catalog, no
// components): the identity and #config checks pass, and the render against
// a platform generated from the module's own deps yields zero objects — the
// verdict build reaches for the same input. The fixture's module name carries
// an underscore, which the default instance name hyphenates. The fixture imports
// opmodel.dev/core@v2, so it resolves only when a registry (or a warm CUE
// cache) is available; without one the core schema load fails with a
// connectivity or no-registry error before the vet check is reached, and the test skips
// rather than false-failing — matching the repo's other registry-backed tests.
func TestModVet_ValidModule(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("Test fixture not found:", fixtureDir)
	}

	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	var logs bytes.Buffer
	output.SetLogWriter(&logs)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

	configPath := filepath.Join(tmpHome, ".opm", "config.cue")
	cfg := &config.GlobalConfig{ConfigPath: configPath}
	cmd := NewModuleVetCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir})

	err := cmd.Execute()
	var exitErr *opmexit.ExitError
	var noRegistry *config.NoRegistryError
	if errors.As(err, &noRegistry) || (errors.As(err, &exitErr) && exitErr.Code == opmexit.ExitConnectivityError) {
		t.Skipf("core@v2 not resolvable (registry/cache unavailable?): %v", err)
	}
	require.NoError(t, err, "identity, #config and a zero-object render all pass")
	assert.Contains(t, logs.String(), "Module config valid")
	assert.Contains(t, logs.String(), "platform: module deps (no catalogs; generated module "+config.PlatformCacheDir(configPath))
	assert.Contains(t, logs.String(), "Module valid (0 resources)")
	assert.Contains(t, logs.String(), `"simple-module-debug"`, "the default instance name hyphenates the module name")
}

func TestNewModuleVetCmd_PlatformFlagNamesTheModuleDeps(t *testing.T) {
	cmd := NewModuleVetCmd(&config.GlobalConfig{})
	assert.Equal(t, "Render against this platform module directory instead of the module's own deps",
		cmd.Flags().Lookup("platform").Usage)
	for _, name := range []string{"values", "namespace", "name", "instance-name", "platform"} {
		assert.NotNil(t, cmd.Flags().Lookup(name), "--%s is registered", name)
	}
	assert.Contains(t, cmd.Long, "--platform <dir>")
}

// --name is the spelling module build and module apply use; --instance-name
// stays as its deprecated alias.
func TestNewModuleVetCmd_NameFlagAndDeprecatedAlias(t *testing.T) {
	cmd := NewModuleVetCmd(&config.GlobalConfig{})

	name := cmd.Flags().Lookup("name")
	require.NotNil(t, name, "--name is registered")
	assert.Empty(t, name.Deprecated)
	assert.Equal(t, NewModuleBuildCmd(&config.GlobalConfig{}).Flags().Lookup("name").Usage, name.Usage,
		"vet and build describe --name the same way")

	old := cmd.Flags().Lookup("instance-name")
	require.NotNil(t, old, "the old spelling still parses")
	assert.Equal(t, "use --name", old.Deprecated)
	assert.True(t, old.Hidden, "a deprecated flag is not in the help")

	var warnings bytes.Buffer
	cmd.SetOut(&warnings)
	require.NoError(t, cmd.ParseFlags([]string{"--instance-name", "web"}))
	assert.Equal(t, "Flag --instance-name has been deprecated, use --name\n", warnings.String())
}

func TestModVet_NameAndInstanceNameTogetherIsAUsageError(t *testing.T) {
	cmd := NewModuleVetCmd(&config.GlobalConfig{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "absent"), "--name", "a", "--instance-name", "b"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "[instance-name name] were all set")
	var exitErr *opmexit.ExitError
	assert.False(t, errors.As(err, &exitErr), "a usage error exits 1 through main's non-ExitError path")
}

// Both spellings name the synthesized instance. Registry-backed like
// TestModVet_ValidModule, and skipped the same way.
func TestModVet_NameAndItsAliasSetTheInstanceName(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module")
	for _, flag := range []string{"--name", "--instance-name"} {
		t.Run(flag, func(t *testing.T) {
			tmpHome, cleanup := setupTestConfig(t)
			defer cleanup()
			t.Setenv("HOME", tmpHome)

			var logs bytes.Buffer
			output.SetLogWriter(&logs)
			t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

			cmd := NewModuleVetCmd(&config.GlobalConfig{ConfigPath: filepath.Join(tmpHome, ".opm", "config.cue")})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{fixtureDir, flag, "web"})

			err := cmd.Execute()
			var exitErr *opmexit.ExitError
			var noRegistry *config.NoRegistryError
			if errors.As(err, &noRegistry) || (errors.As(err, &exitErr) && exitErr.Code == opmexit.ExitConnectivityError) {
				t.Skipf("core not resolvable (registry or cache unavailable?): %v", err)
			}
			require.NoError(t, err)
			assert.Contains(t, logs.String(), `"web"`)
			assert.NotContains(t, logs.String(), "simple-module-debug")
		})
	}
}

func TestModVet_RejectsInstancePackage(t *testing.T) {
	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_REGISTRY")

	instanceDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(instanceDir, "instance.cue"), []byte(`package jellyfin

kind: "ModuleInstance"
metadata: name: "jf"
`), 0o600))

	cfg := &config.GlobalConfig{}
	cmd := NewModuleVetCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{instanceDir})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is an instance package, not a module")
	assert.Contains(t, err.Error(), "opm instance")
}

func TestModVet_CUEValidationError(t *testing.T) {
	t.Skip("Requires valid OPM module fixture with CUE errors — skipping for now")
	// This test requires a module that triggers CUE validation errors during render.
	// The test module needs proper imports and structure which is complex to set up inline.
	// Integration tests with real fixtures would be better for this case.
}

// TestModVet_ValuesDetailLogic checks the display detail string vetValuesDetail
// assembles for the "Values satisfy #config" vet check line.
func TestModVet_ValuesDetailLogic(t *testing.T) {
	tests := []struct {
		name           string
		valuesFlags    []string
		expectedDetail string
	}{
		{
			name:           "no values flags uses debugValues",
			valuesFlags:    nil,
			expectedDetail: "debugValues",
		},
		{
			name:           "single external values file",
			valuesFlags:    []string{"/path/to/prod-values.cue"},
			expectedDetail: "prod-values.cue",
		},
		{
			name:           "multiple external values files",
			valuesFlags:    []string{"/path/to/base.cue", "/another/path/prod.cue"},
			expectedDetail: "base.cue, prod.cue",
		},
		{
			name:           "values files with absolute paths show only basename",
			valuesFlags:    []string{"/very/long/path/to/config/values.cue"},
			expectedDetail: "values.cue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedDetail, vetValuesDetail(tt.valuesFlags))
		})
	}
}

// setupTestConfig creates a minimal test config in a temp directory.
func setupTestConfig(t *testing.T) (tmpHome string, cleanup func()) {
	tmpHome, err := os.MkdirTemp("", "mod-vet-config-*")
	require.NoError(t, err)

	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))

	cueModDir := filepath.Join(opmDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o700))

	// Minimal config
	simpleConfig := `package config

config: {
	providers: {
		"default": {
			registry: "opmodel.dev"
		}
	}
}
`
	require.NoError(t, os.WriteFile(filepath.Join(opmDir, "config.cue"), []byte(simpleConfig), 0o600))

	// Module file
	moduleContent := `module: "test.local/config@v0"

language: {
	version: "v0.15.0"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(cueModDir, "module.cue"), []byte(moduleContent), 0o600))

	cleanup = func() {
		os.RemoveAll(tmpHome)
	}

	return tmpHome, cleanup
}

// unusableCUECache points CUE_CACHE_DIR at a regular file, so a module fetch
// fails before it asks any registry: the failing schema load needs no network.
func unusableCUECache(t *testing.T) {
	t.Helper()
	notADir := filepath.Join(t.TempDir(), "cache")
	require.NoError(t, os.WriteFile(notADir, nil, 0o600))
	t.Setenv("CUE_CACHE_DIR", notADir)
}

// With no registry configured anywhere, a core schema that cannot be loaded
// is reported as the missing configuration (exit 2, opm config init), not as
// a connectivity failure.
func TestModVet_NoRegistryConfigured(t *testing.T) {
	unusableCUECache(t)
	t.Setenv("OPM_REGISTRY", "")
	t.Setenv("CUE_REGISTRY", "")

	configPath := filepath.Join(t.TempDir(), "config.cue")
	cmd := NewModuleVetCmd(&config.GlobalConfig{ConfigPath: configPath})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{t.TempDir()})

	err := cmd.Execute()
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code, "%v", err)
	assert.Contains(t, err.Error(), "no registry is configured: loading core schema: ")
	assert.Contains(t, err.Error(), "the registry field of "+configPath)
	assert.Contains(t, err.Error(), "opm config init")
}

// A run with only CUE_REGISTRY set has a registry: the same failure keeps its
// connectivity exit code and is not called a missing configuration.
func TestModVet_OnlyCUERegistryIsConfigured(t *testing.T) {
	unusableCUECache(t)
	t.Setenv("OPM_REGISTRY", "")
	t.Setenv("CUE_REGISTRY", cuemodtest.UnreachableRegistry)

	cmd := NewModuleVetCmd(&config.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{t.TempDir()})

	err := cmd.Execute()
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitConnectivityError, exitErr.Code, "%v", err)
	assert.NotContains(t, err.Error(), "no registry is configured")
	assert.Contains(t, err.Error(), "loading core schema: ")
}

// The core schema fetch is classified as publish classifies it: only no
// response is unreachable, another failure answer is a failed registry
// operation, and a refused credential exits 4 and points to the login.
func TestModVet_SchemaFetchFailureIsNamed(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module")
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		want     string
		code     int
		login    bool
	}{
		{"refused connection", func(*testing.T) string { return cuemodtest.UnreachableRegistry }, "registry unreachable: loading core schema: ", opmexit.ExitConnectivityError, false},
		{"401", func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusUnauthorized) }, "registry refused the credentials (authentication or permission): loading core schema: ", opmexit.ExitPermissionDenied, true},
		// Known gap, pinned: CUE's registry client reports a 403 answer to
		// the schema's tag lookup as "not found", so the refusal never
		// reaches the classification and the fetch reads as a failed
		// registry operation. The same holds for publish.
		{"403", func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusForbidden) }, "registry operation failed: loading core schema: ", opmexit.ExitConnectivityError, false},
		{"503", func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusServiceUnavailable) }, "registry operation failed: loading core schema: ", opmexit.ExitConnectivityError, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cuemodtest.ColdCache(t)
			registry := tc.registry(t)

			cmd := NewModuleVetCmd(&config.GlobalConfig{Registry: registry})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{fixtureDir})
			err := cmd.Execute()

			var exitErr *opmexit.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, tc.code, exitErr.Code, "%v", err)
			assert.Contains(t, err.Error(), tc.want)
			assert.Equal(t, tc.login, strings.HasSuffix(err.Error(), "opm registry login "+registry), "%v", err)
		})
	}
}

func TestNewModuleVetCmd_HelpStatesTheExitCodes(t *testing.T) {
	long := NewModuleVetCmd(&config.GlobalConfig{}).Long
	assert.Contains(t, long, "Exit codes: 0 valid, 1 usage error")
	assert.Contains(t, long, "4 the registry refused the credentials.")
}
