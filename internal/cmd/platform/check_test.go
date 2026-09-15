package platformcmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
)

// The fixture platforms below declare their catalogs inline rather than
// importing published ones: the inventory is derived from the registry
// entries' contract maps, so a platform depending on core alone exercises
// every arm of the report without pinning a catalog release.
const fixtureModule = `module: "testing.opmodel.dev/platforms/check-fixture@v0"
language: version: "v0.17.0"
deps: "opmodel.dev/core@v2": v: "` + config.DefaultCorePin + `"
`

// fixtureContracts is the shared prelude: one catalog-fulfilled resource, one
// provider-fulfilled trait, and the base catalog that defines both.
//
//nolint:misspell // "fulfilment" is core's field name (#Resource/#Trait.fulfilment); the CUE must spell it exactly
const fixtureContracts = `
_container: c.#Resource & {
	metadata: {
		name:       "container"
		apiVersion: "v1beta1"
		fqn:        "testing.opmodel.dev/catalogs/base/resources/container@v1beta1"
	}
	spec: container: image: string
}

_backup: c.#Trait & {
	metadata: {
		name:       "backup"
		apiVersion: "v1alpha1"
		fqn:        "testing.opmodel.dev/catalogs/base/traits/backup@v1alpha1"
	}
	fulfilment: "provider"
	optional:   bool | *false
	spec: backup: schedule: string
	appliesTo: [_container]
}

_baseCatalog: c.#Catalog & {
	metadata: {
		modulePath: "testing.opmodel.dev/catalogs/base@v1"
		version:    "1.0.0"
	}
	#resources: (_container.metadata.fqn): _container
	#traits: (_backup.metadata.fqn):       _backup
}
`

// routablePlatform: one provider catalog implements the provider-fulfilled
// contract, so nothing is unfulfilled and nothing over-subscribed.
const routablePlatform = `package platform

import c "opmodel.dev/core@v2"

c.#Platform
metadata: name: "routable"
type: "kubernetes"
` + fixtureContracts + `
_k8upSchedule: c.#ComponentTransformer & {
	metadata: {
		name: "schedule"
		fqn:  "testing.opmodel.dev/catalogs/k8up/transformers/schedule@2.0.0"
	}
	requiredTraits: (_backup.metadata.fqn): _backup
}

_k8upCatalog: c.#Catalog & {
	metadata: {
		modulePath: "testing.opmodel.dev/catalogs/k8up@v2"
		version:    "2.0.0"
	}
	#transformers: (_k8upSchedule.metadata.fqn): _k8upSchedule
}

#registry: {
	(_baseCatalog.metadata.modulePath): #catalog: _baseCatalog
	(_k8upCatalog.metadata.modulePath): #catalog: _k8upCatalog
}
`

// overSubscribedPlatform: two provider catalogs compete for one
// provider-fulfilled contract.
const overSubscribedPlatform = routablePlatform + `
_veleroBackup: c.#ComponentTransformer & {
	metadata: {
		name: "backup"
		fqn:  "testing.opmodel.dev/catalogs/velero/transformers/backup@1.4.0"
	}
	requiredTraits: (_backup.metadata.fqn): _backup
}

_veleroCatalog: c.#Catalog & {
	metadata: {
		modulePath: "testing.opmodel.dev/catalogs/velero@v1"
		version:    "1.4.0"
	}
	#transformers: (_veleroBackup.metadata.fqn): _veleroBackup
}

#registry: (_veleroCatalog.metadata.modulePath): #catalog: _veleroCatalog
`

// writePlatformDir writes a platform module holding platformCUE and returns
// its directory.
func writePlatformDir(t *testing.T, modFile, platformCUE string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(modFile), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte(platformCUE), 0o600))
	return dir
}

// runCheck drives `opm platform check` in process against dir and returns the
// report it printed on stdout, the diagnostics it printed on the log and
// details streams, and the error it returned.
func runCheck(t *testing.T, dir string) (report, diagnostics string, err error) {
	t.Helper()
	cmd := NewPlatformCheckCmd(&config.GlobalConfig{Registry: config.DefaultRegistry})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{dir})

	// The output package writes to the process streams directly, not to the
	// cobra command's buffers: the report goes to stdout, the grouped
	// diagnostic to the log writer and the detail block to stderr.
	var logBuf bytes.Buffer
	output.SetLogWriter(&logBuf)
	t.Cleanup(func() { output.SetupLogging(output.LogConfig{}) })

	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, perr := os.Pipe()
	require.NoError(t, perr)
	rErr, wErr, perr := os.Pipe()
	require.NoError(t, perr)
	os.Stdout, os.Stderr = wOut, wErr

	err = cmd.Execute()

	require.NoError(t, wOut.Close())
	require.NoError(t, wErr.Close())
	os.Stdout, os.Stderr = oldOut, oldErr
	outRaw, rerr := io.ReadAll(rOut)
	require.NoError(t, rerr)
	require.NoError(t, rOut.Close())
	errRaw, rerr := io.ReadAll(rErr)
	require.NoError(t, rerr)
	require.NoError(t, rErr.Close())
	return string(outRaw), logBuf.String() + string(errRaw), err
}

// skipIfRegistryUnavailable skips when err looks like the registry could not
// be reached (the repo's posture for registry-backed unit tests).
func skipIfRegistryUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	for _, needle := range []string{"dial tcp", "no such host", "connection refused", "i/o timeout", "context deadline exceeded", "TLS handshake", "network is unreachable"} {
		if strings.Contains(msg, needle) {
			t.Skipf("registry unavailable: %v", err)
		}
	}
}

func TestPlatformCheck_RoutablePlatformExitsZero(t *testing.T) {
	dir := writePlatformDir(t, fixtureModule, routablePlatform)

	report, _, err := runCheck(t, dir)
	skipIfRegistryUnavailable(t, err)
	require.NoError(t, err)

	assert.Contains(t, report, "platform: "+dir+" (argument)")
	assert.Contains(t, report, "defined contracts: 2")
	assert.Contains(t, report, "testing.opmodel.dev/catalogs/base/traits/backup@v1alpha1")
	assert.Contains(t, report, "defined by      testing.opmodel.dev/catalogs/base@v1")
	assert.Contains(t, report, "implemented by  testing.opmodel.dev/catalogs/k8up/transformers/schedule@2.0.0")
	assert.Contains(t, report, "fulfilled: yes")
	assert.Contains(t, report, "routable:  yes")
	assert.NotContains(t, report, "over-subscribed contracts")
}

func TestPlatformCheck_OverSubscribedPlatformExitsValidation(t *testing.T) {
	dir := writePlatformDir(t, fixtureModule, overSubscribedPlatform)

	report, _, err := runCheck(t, dir)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.True(t, exitErr.Printed, "the report already carries the verdict")

	assert.Contains(t, report, "over-subscribed contracts: 1")
	assert.Contains(t, report, "testing.opmodel.dev/catalogs/base/traits/backup@v1alpha1 (defined by testing.opmodel.dev/catalogs/base@v1)")
	// Both competing catalogs are named.
	assert.Contains(t, report, "testing.opmodel.dev/catalogs/k8up/transformers/schedule@2.0.0")
	assert.Contains(t, report, "testing.opmodel.dev/catalogs/velero/transformers/backup@1.4.0")
	assert.Contains(t, report, "routable:  no — 1 contract is over-subscribed")
}

func TestPlatformCheck_NotAPlatformModuleFailsBeforeAnyBuild(t *testing.T) {
	// A directory with no cue.mod/module.cue: resolution refuses it, so
	// nothing is fetched and nothing is built.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte("package platform\n"), 0o600))

	report, _, err := runCheck(t, dir)
	require.Error(t, err)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitNotFound, exitErr.Code)
	assert.Contains(t, err.Error(), dir)
	assert.Contains(t, err.Error(), "is not a platform module")
	assert.Contains(t, err.Error(), "cue.mod/module.cue not found")
	assert.Empty(t, report, "no report is printed for a platform that was never built")
}

func TestPlatformCheck_PlatformThatDoesNotBuildReportsItsDiagnostic(t *testing.T) {
	// metadata.name is a string on #Platform; an int conflicts, and the
	// conflict carries a source position, so it prints grouped.
	dir := writePlatformDir(t, fixtureModule, `package platform

import c "opmodel.dev/core@v2"

c.#Platform
metadata: name: 42
type: "kubernetes"
`)

	report, diagnostics, err := runCheck(t, dir)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Contains(t, diagnostics, "platform module does not build", "the failure is framed before the diagnostic")
	assert.Contains(t, diagnostics, "conflicting values 42 and string")
	assert.Contains(t, diagnostics, "platform.cue:6:17", "the grouped diagnostic names the source position")
	assert.Contains(t, err.Error(), "metadata.name")
	assert.Empty(t, report, "an empty inventory is never reported in place of a build failure")
}

func TestPlatformCheck_CoreWithoutTheInventoryNamesTheRelease(t *testing.T) {
	// A platform built against a core release predating the contract
	// inventory derives no #contracts; the command says so and names the
	// release that introduced it, rather than printing an empty report.
	dir := writePlatformDir(t, `module: "testing.opmodel.dev/platforms/check-fixture@v0"
language: version: "v0.17.0"
deps: "opmodel.dev/core@v2": v: "v2.0.0-alpha.6"
`, `package platform

import c "opmodel.dev/core@v2"

c.#Platform
metadata: name: "pre-inventory"
type: "kubernetes"
`)

	report, _, err := runCheck(t, dir)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Contains(t, err.Error(), "#contracts")
	assert.Contains(t, err.Error(), "2.0.0-alpha.9")
	assert.Contains(t, err.Error(), dir)
	assert.Empty(t, report)
}
