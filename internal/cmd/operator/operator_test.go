package operatorcmd

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/internal/publish"
)

func TestNewOperatorCmd(t *testing.T) {
	cmd := NewOperatorCmd(&config.GlobalConfig{})

	assert.Equal(t, "operator", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.ElementsMatch(t, []string{"install", "uninstall"}, names)
}

func TestNewOperatorInstallCmd(t *testing.T) {
	cmd := NewOperatorInstallCmd(&config.GlobalConfig{})

	assert.Equal(t, "install", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	for _, flag := range []string{
		"crds-only", "rbac", "user", "group", "version", "timeout",
		"kubeconfig", "context", "catalog-prerelease", "skip-platform",
	} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "expected --%s flag", flag)
	}

	for _, flag := range []string{"catalog-prerelease", "skip-platform"} {
		assert.Equal(t, "false", cmd.Flags().Lookup(flag).DefValue,
			"--%s must default off", flag)
	}
}

func TestNewOperatorUninstallCmd(t *testing.T) {
	cmd := NewOperatorUninstallCmd(&config.GlobalConfig{})

	assert.Equal(t, "uninstall", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	for _, flag := range []string{"remove-finalizers", "kubeconfig", "context"} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "expected --%s flag", flag)
	}
}

func TestInstallFlagsSeedsPlatform(t *testing.T) {
	assert.True(t, installFlags{}.seedsPlatform(), "a bare install seeds the Platform")
	assert.False(t, installFlags{crdsOnly: true}.seedsPlatform())
	assert.False(t, installFlags{skipPlatform: true}.seedsPlatform())
}

func TestInstallFlagsValidate(t *testing.T) {
	tests := []struct {
		name    string
		flags   installFlags
		wantErr bool
	}{
		{name: "bare install", flags: installFlags{}},
		{name: "prerelease on a seeding install", flags: installFlags{catalogPrerelease: true}},
		{name: "crds-only alone", flags: installFlags{crdsOnly: true}},
		{name: "skip-platform alone", flags: installFlags{skipPlatform: true}},
		{
			name:    "prerelease with crds-only",
			flags:   installFlags{crdsOnly: true, catalogPrerelease: true},
			wantErr: true,
		},
		{
			name:    "prerelease with skip-platform",
			flags:   installFlags{skipPlatform: true, catalogPrerelease: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.flags.validate()
			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), "--catalog-prerelease")
		})
	}
}

// The invalid combinations must be rejected before any registry lookup or
// cluster call. runOperatorInstall is given a config with an unusable
// registry and an unreachable kubeconfig, so reaching either would surface a
// different error than the flag-validation one asserted here.
func TestRunOperatorInstallRejectsFlagsBeforeAnyIO(t *testing.T) {
	cfg := &config.GlobalConfig{Registry: "!!not a registry!!"}

	for _, flags := range []installFlags{
		{crdsOnly: true, catalogPrerelease: true},
		{skipPlatform: true, catalogPrerelease: true},
	} {
		err := runOperatorInstall(context.Background(), cfg, &cmdutil.K8sFlags{
			Kubeconfig: filepath.Join(t.TempDir(), "nonexistent-kubeconfig"),
		}, flags)

		var exitErr *opmexit.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
		assert.Contains(t, err.Error(), "--catalog-prerelease")
	}
}

// A refusal from catalog resolution prints through the refusal funnel and
// exits 2; an unreachable registry exits 3 because nothing was judged.
func TestCatalogResolveErrorMapping(t *testing.T) {
	refusal := &platform.RefusalError{Refusal: publish.Refusal{
		Headline:    "opmodel.dev/catalogs/opm@v2 has no published release",
		Evidence:    [][]string{{"catalog", platform.DefaultCatalogPath}},
		Consequence: "A cluster Platform pins exactly one published catalog build.",
		Action:      "opm operator install --catalog-prerelease",
	}}
	var exitErr *opmexit.ExitError

	require.ErrorAs(t, catalogResolveError(refusal), &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.True(t, exitErr.Printed, "a refusal is printed by the funnel, not re-printed by the runner")

	conn := &publish.ConnectivityError{Op: "listing published versions", Err: errors.New("connection refused")}
	require.ErrorAs(t, catalogResolveError(conn), &exitErr)
	assert.Equal(t, opmexit.ExitConnectivityError, exitErr.Code)

	require.ErrorAs(t, catalogResolveError(errors.New("boom")), &exitErr)
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
}
