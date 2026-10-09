package controllercmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/library/opm/k8s/ownership"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	ctrllib "github.com/open-platform-model/cli/internal/controller"
	"github.com/open-platform-model/cli/internal/controller/controllertest"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/modref"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/internal/publish"
)

func TestNewControllerCmd(t *testing.T) {
	cmd := NewControllerCmd(&config.GlobalConfig{})

	assert.Equal(t, "controller", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.ElementsMatch(t, []string{"install", "uninstall"}, names)
}

func TestNewControllerInstallCmd(t *testing.T) {
	cmd := NewControllerInstallCmd(&config.GlobalConfig{})

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

func TestNewControllerUninstallCmd(t *testing.T) {
	cmd := NewControllerUninstallCmd(&config.GlobalConfig{})

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
// cluster call. runControllerInstall is given a config with an unusable
// registry and an unreachable kubeconfig, so reaching either would surface a
// different error than the flag-validation one asserted here.
func TestRunControllerInstallRejectsFlagsBeforeAnyIO(t *testing.T) {
	cfg := &config.GlobalConfig{Registry: "!!not a registry!!"}

	for _, c := range []struct {
		flags installFlags
		want  string
	}{
		{installFlags{crdsOnly: true, catalogPrerelease: true}, "--catalog-prerelease"},
		{installFlags{skipPlatform: true, catalogPrerelease: true}, "--catalog-prerelease"},
		{installFlags{crdsOnly: true, values: []string{"values.cue"}}, "--reset-values have no effect with --crds-only"},
		{installFlags{crdsOnly: true, resetValues: true}, "--reset-values have no effect with --crds-only"},
		{installFlags{crdsOnly: true, version: "0.2.0"}, "drop --version"},
	} {
		err := runControllerInstall(context.Background(), cfg, &cmdutil.K8sFlags{
			Kubeconfig: filepath.Join(t.TempDir(), "nonexistent-kubeconfig"),
		}, c.flags)

		var exitErr *opmexit.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
		assert.Contains(t, err.Error(), c.want)
	}
}

// --crds-only touches no registry: with a registry mapping that cannot be
// parsed and the module registry wired to fail the test, it goes straight to
// the cluster and fails only at the missing kubeconfig.
func TestRunControllerInstallCRDsOnlySkipsTheRegistry(t *testing.T) {
	prev := newModuleRegistry
	newModuleRegistry = func(string) (ctrllib.ModuleRegistry, error) {
		t.Fatal("--crds-only built a module registry client")
		return nil, nil
	}
	t.Cleanup(func() { newModuleRegistry = prev })
	cfg := &config.GlobalConfig{Registry: "!!not a registry!!"}

	err := runControllerInstall(context.Background(), cfg, &cmdutil.K8sFlags{
		Kubeconfig: filepath.Join(t.TempDir(), "nonexistent-kubeconfig"),
	}, installFlags{crdsOnly: true, timeout: time.Second})

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "registry")
	assert.Contains(t, err.Error(), "nonexistent-kubeconfig")
}

// A refusal from catalog resolution prints through the refusal funnel and
// exits 2; an unreachable registry exits 3 because nothing was judged.
func TestCatalogResolveErrorMapping(t *testing.T) {
	refusal := &platform.RefusalError{Refusal: publish.Refusal{
		Headline:    "opmodel.dev/catalogs/opm@v2 has no published release",
		Evidence:    [][]string{{"catalog", platform.DefaultCatalogPath}},
		Consequence: "A cluster Platform pins exactly one published catalog build.",
		Action:      "opm controller install --catalog-prerelease",
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

// mirrorRegistry serves the pinned controller module version and 0.2.0 from
// an in-memory registry that is the only registry the mapping names.
func mirrorRegistry(t *testing.T) string {
	t.Helper()
	return controllertest.Registry(t,
		controllertest.Version{Module: ctrllib.PinnedModuleVersion, Controller: strings.TrimPrefix(ctrllib.PinnedControllerVersion, "v")},
		controllertest.Version{Module: "0.2.0", Controller: "1.0.0-beta.8"},
	)
}

// installWithoutCluster runs install with --skip-platform against a
// kubeconfig that does not exist, so it stops at the first cluster step.
func installWithoutCluster(t *testing.T, registry string, flags installFlags) error {
	t.Helper()
	flags.skipPlatform = true
	return runControllerInstall(context.Background(), &config.GlobalConfig{Registry: registry}, &cmdutil.K8sFlags{
		Kubeconfig: filepath.Join(t.TempDir(), "nonexistent-kubeconfig"),
	}, flags)
}

// "Unserved version": a selector no module version matches, an opm-controller
// release tag included, is refused before any cluster call with no message
// of its own.
func TestRunControllerInstall_UnservedVersionIsRefusedBeforeTheCluster(t *testing.T) {
	reg := mirrorRegistry(t)
	for _, v := range []string{"v1.0.0-beta.5", "1.0.0-beta.5"} {
		err := installWithoutCluster(t, reg, installFlags{version: v})
		var exitErr *opmexit.ExitError
		require.ErrorAs(t, err, &exitErr, v)
		assert.Equal(t, opmexit.ExitValidationError, exitErr.Code, v)
		assert.NotContains(t, err.Error(), "--version now takes", v)
		assert.NotContains(t, err.Error(), "kubeconfig", v)
	}
}

// "Mirror only" and "Default install uses the pin", up to the cluster: the
// module and the controller version it deploys resolve through a mapping that
// names only the mirror, and the output names both versions.
func TestRunControllerInstall_ResolvesThroughAMirrorOnly(t *testing.T) {
	logs := captureLogs(t)
	reg := mirrorRegistry(t)

	err := installWithoutCluster(t, reg, installFlags{})
	require.Error(t, err, "the missing kubeconfig stops it after resolution")
	assert.Contains(t, logs.String(), "controller module opmodel.dev/modules/opm_controller "+ctrllib.PinnedModuleVersion+" (pinned; deploys opm-controller "+ctrllib.PinnedControllerVersion+")")

	logs.Reset()
	_ = installWithoutCluster(t, reg, installFlags{version: "0.2.0"})
	assert.Contains(t, logs.String(), "opm_controller 0.2.0 (--version 0.2.0; deploys opm-controller v1.0.0-beta.8)")
}

// An unreachable registry exits 3 and names the mirror fix.
func TestRunControllerInstall_UnreachableRegistry(t *testing.T) {
	err := installWithoutCluster(t, "opmodel.dev=127.0.0.1:1+insecure", installFlags{})
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitConnectivityError, exitErr.Code)
	assert.Contains(t, err.Error(), "--registry or OPM_REGISTRY")
	assert.Contains(t, err.Error(), ctrllib.ControllerModulePath)
}

func TestInstallErrorMapping(t *testing.T) {
	unreadable := apierrors.NewForbidden(schema.GroupResource{Group: "rbac.authorization.k8s.io", Resource: "clusterroles"}, "x", errors.New("no get"))
	cases := []struct {
		err  error
		code int
	}{
		{&ctrllib.TargetError{ModuleVersion: "v0.4.0", Rule: "newer"}, opmexit.ExitValidationError},
		{&ctrllib.VersionError{ModuleVersion: "v0.4.0", Reason: "none"}, opmexit.ExitValidationError},
		{&ctrllib.GuardError{Err: errors.New("exists")}, opmexit.ExitValidationError},
		// The guard refuses an object another instance owns or adopts.
		{&ctrllib.GuardError{Err: &inventory.GuardRefusalError{Refused: []inventory.Refused{{Reason: ownership.RefuseAdoptedElsewhere, Message: "Deployment/x was adopted"}}}}, opmexit.ExitValidationError},
		// An object install cannot read. The guard's refusal keeps its code
		// whatever API error it wraps; the same read failing in an earlier
		// check exits by the API error.
		{&ctrllib.GuardError{Err: fmt.Errorf("cannot check whether ClusterRole/x already exists: %w", unreadable)}, opmexit.ExitValidationError},
		{fmt.Errorf("checking ClusterRole/x before apply: %w", unreadable), opmexit.ExitPermissionDenied},
		{&ctrllib.OwnedRecordError{}, opmexit.ExitValidationError},
		{&modref.RefusalError{Refusal: publish.Refusal{Headline: "no such version"}}, opmexit.ExitValidationError},
		{&publish.ConnectivityError{Op: "listing", Err: errors.New("refused")}, opmexit.ExitConnectivityError},
		{&ctrllib.RolloutError{Err: errors.New("timed out")}, opmexit.ExitGeneralError},
		{&opmexit.ExitError{Code: opmexit.ExitPermissionDenied, Err: errors.New("denied")}, opmexit.ExitPermissionDenied},
		// The rerun hint keeps the code of the error it wraps.
		{withRerunHint(&ctrllib.InstallResult{CRDs: 4}, &ctrllib.RolloutError{Err: errors.New("timed out")}), opmexit.ExitGeneralError},
	}
	for _, c := range cases {
		var exitErr *opmexit.ExitError
		require.ErrorAs(t, installError(c.err), &exitErr, c.err.Error())
		assert.Equal(t, c.code, exitErr.Code, c.err.Error())
	}
}

// A failure after the CRD step says install is safe to re-run; one before
// it does not.
func TestWithRerunHint(t *testing.T) {
	const hint = "(install is idempotent, safe to re-run)"
	failed := errors.New("apply failed")

	err := withRerunHint(&ctrllib.InstallResult{CRDs: 4}, failed)
	assert.ErrorIs(t, err, failed)
	assert.Contains(t, err.Error(), hint)

	assert.Same(t, failed, withRerunHint(nil, failed), "no result")
	assert.Same(t, failed, withRerunHint(&ctrllib.InstallResult{}, failed), "the CRD step did not run")
}
