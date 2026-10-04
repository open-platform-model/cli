package operatorcmd

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/modref"
	oplib "github.com/open-platform-model/cli/internal/operator"
	"github.com/open-platform-model/cli/internal/operator/operatortest"
	"github.com/open-platform-model/cli/internal/output"
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
		"values", "reset-values",
	} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "expected --%s flag", flag)
	}

	assert.Equal(t, "f", cmd.Flags().Lookup("values").Shorthand)
	assert.Equal(t, "", cmd.Flags().Lookup("version").DefValue, "--version defaults to the pinned module version")

	for _, flag := range []string{"catalog-prerelease", "skip-platform", "reset-values"} {
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
		{name: "values with reset", flags: installFlags{values: []string{"v.cue"}, resetValues: true}},
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

	for _, c := range []struct {
		flags installFlags
		want  string
	}{
		{installFlags{crdsOnly: true, catalogPrerelease: true}, "--catalog-prerelease"},
		{installFlags{skipPlatform: true, catalogPrerelease: true}, "--catalog-prerelease"},
		{installFlags{crdsOnly: true, values: []string{"values.cue"}}, "--reset-values have no effect with --crds-only"},
		{installFlags{crdsOnly: true, resetValues: true}, "--reset-values have no effect with --crds-only"},
	} {
		err := runOperatorInstall(context.Background(), cfg, &cmdutil.K8sFlags{
			Kubeconfig: filepath.Join(t.TempDir(), "nonexistent-kubeconfig"),
		}, c.flags)

		var exitErr *opmexit.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
		assert.Contains(t, err.Error(), c.want)
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

// captureLogs routes the log stream into a buffer for the test's duration.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)
	t.Cleanup(func() { output.SetupLogging(output.LogConfig{}) })
	return &buf
}

// mirrorRegistry serves the pinned operator module version and 0.2.0 from
// an in-memory registry that is the only registry the mapping names.
func mirrorRegistry(t *testing.T) string {
	t.Helper()
	return operatortest.Registry(t,
		operatortest.Version{Module: oplib.PinnedModuleVersion, Operator: strings.TrimPrefix(oplib.PinnedOperatorVersion, "v")},
		operatortest.Version{Module: "0.2.0", Operator: "1.0.0-beta.8"},
	)
}

// installWithoutCluster runs install with --skip-platform against a
// kubeconfig that does not exist, so it stops at the first cluster step.
func installWithoutCluster(t *testing.T, registry string, flags installFlags) error {
	t.Helper()
	flags.skipPlatform = true
	return runOperatorInstall(context.Background(), &config.GlobalConfig{Registry: registry}, &cmdutil.K8sFlags{
		Kubeconfig: filepath.Join(t.TempDir(), "nonexistent-kubeconfig"),
	}, flags)
}

// "Old-style operator tag": refused before any cluster call, saying what
// --version takes now.
func TestRunOperatorInstall_OldOperatorTagIsRefusedBeforeTheCluster(t *testing.T) {
	reg := mirrorRegistry(t)
	for _, old := range []string{"v1.0.0-beta.5", "1.0.0-beta.5"} {
		err := installWithoutCluster(t, reg, installFlags{version: old})
		var exitErr *opmexit.ExitError
		require.ErrorAs(t, err, &exitErr, old)
		assert.Equal(t, opmexit.ExitValidationError, exitErr.Code, old)
		assert.Contains(t, err.Error(), "--version now takes an operator module version", old)
		assert.NotContains(t, err.Error(), "kubeconfig", old)
	}
}

// "Mirror only" and "Default install uses the pin", up to the cluster: the
// module and the operator version it deploys resolve through a mapping that
// names only the mirror, and the output names both versions.
func TestRunOperatorInstall_ResolvesThroughAMirrorOnly(t *testing.T) {
	logs := captureLogs(t)
	reg := mirrorRegistry(t)

	err := installWithoutCluster(t, reg, installFlags{})
	require.Error(t, err, "the missing kubeconfig stops it after resolution")
	assert.Contains(t, logs.String(), "operator module opmodel.dev/modules/opm_operator "+oplib.PinnedModuleVersion+" (pinned; deploys opm-operator "+oplib.PinnedOperatorVersion+")")

	logs.Reset()
	_ = installWithoutCluster(t, reg, installFlags{version: "0.2.0"})
	assert.Contains(t, logs.String(), "opm_operator 0.2.0 (--version 0.2.0; deploys opm-operator v1.0.0-beta.8)")
}

// An unreachable registry exits 3 and names the mirror fix.
func TestRunOperatorInstall_UnreachableRegistry(t *testing.T) {
	err := installWithoutCluster(t, "opmodel.dev=127.0.0.1:1+insecure", installFlags{})
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitConnectivityError, exitErr.Code)
	assert.Contains(t, err.Error(), "--registry or OPM_REGISTRY")
	assert.Contains(t, err.Error(), oplib.OperatorModulePath)
}

func TestInstallErrorMapping(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{&oplib.TargetError{ModuleVersion: "v0.4.0", Rule: "newer"}, opmexit.ExitValidationError},
		{&oplib.VersionError{ModuleVersion: "v0.4.0", Reason: "none"}, opmexit.ExitValidationError},
		{&oplib.GuardError{Err: errors.New("exists")}, opmexit.ExitValidationError},
		{&oplib.OwnedRecordError{}, opmexit.ExitValidationError},
		{&modref.RefusalError{Refusal: publish.Refusal{Headline: "no such version"}}, opmexit.ExitValidationError},
		{&publish.ConnectivityError{Op: "listing", Err: errors.New("refused")}, opmexit.ExitConnectivityError},
		{&oplib.RolloutError{Err: errors.New("timed out")}, opmexit.ExitGeneralError},
		{&opmexit.ExitError{Code: opmexit.ExitPermissionDenied, Err: errors.New("denied")}, opmexit.ExitPermissionDenied},
	}
	for _, c := range cases {
		var exitErr *opmexit.ExitError
		require.ErrorAs(t, installError(c.err), &exitErr, c.err.Error())
		assert.Equal(t, c.code, exitErr.Code, c.err.Error())
	}
}

// The operator's instance renders under its fixed name and namespace,
// from the merged values only, against the module's own pins.
func TestModuleRenderOpts(t *testing.T) {
	cfg := &config.GlobalConfig{}
	k8s := &config.ResolvedKubernetesConfig{}
	res := &modref.Resolution{Path: oplib.OperatorModulePath, Version: "v0.1.0"}
	values := kernel.Source{Origin: "merged values"}

	opts := moduleRenderOpts(cfg, k8s, res, values)

	assert.Same(t, res, opts.Published)
	assert.Equal(t, oplib.OperatorInstanceName, opts.Name)
	assert.Equal(t, oplib.OperatorNamespace, opts.Namespace)
	assert.True(t, opts.DepsOnly, "the render never reads a Platform")
	require.Len(t, opts.Values, 1)
	assert.Equal(t, "merged values", opts.Values[0].Origin)
	assert.Empty(t, opts.ValuesFiles)
	assert.Empty(t, opts.PlatformFlag)
	assert.Nil(t, opts.ClusterPlatform)
	assert.Same(t, cfg, opts.Config)
	assert.Same(t, k8s, opts.K8sConfig)
}
