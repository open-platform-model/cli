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
	requiredResources: (_container.metadata.fqn): _container
	requiredTraits: (_backup.metadata.fqn):       _backup
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

// baseOnlyPlatform: the base catalog defines the provider-fulfilled contract
// and no enabled catalog implements it. Unfulfilled, and still routable.
const baseOnlyPlatform = `package platform

import c "opmodel.dev/core@v2"

c.#Platform
metadata: name: "base-only"
type: "kubernetes"
` + fixtureContracts + `
#registry: (_baseCatalog.metadata.modulePath): #catalog: _baseCatalog
`

// catalogFulfilledPluralityPlatform: two catalogs' transformers require the
// same CATALOG-fulfilled contract. Only a provider-fulfilled contract can be
// over-subscribed, so this platform stays routable.
//
// The mirror transformer declares a required label that k8up's schedule does
// not, which is what keeps the pair DISCRIMINATED: schedule requires the
// container plus the backup trait, and on labels alone mirror would be the
// broader of the two. Without the label this platform is
// undiscriminatedPlatform below, and the "many suppliers, still routable"
// claim would be read on a platform the operator refuses for a different
// reason.
const catalogFulfilledPluralityPlatform = routablePlatform + `
_veleroMirror: c.#ComponentTransformer & {
	metadata: {
		name: "mirror"
		fqn:  "testing.opmodel.dev/catalogs/velero/transformers/mirror@1.4.0"
	}
	requiredResources: (_container.metadata.fqn): _container
	requiredLabels: "testing.opmodel.dev/mirror": "true"
}

_veleroMirrorCatalog: c.#Catalog & {
	metadata: {
		modulePath: "testing.opmodel.dev/catalogs/velero@v1"
		version:    "1.4.0"
	}
	#transformers: (_veleroMirror.metadata.fqn): _veleroMirror
}

#registry: (_veleroMirrorCatalog.metadata.modulePath): #catalog: _veleroMirrorCatalog
`

// undiscriminatedPlatform: catalogFulfilledPluralityPlatform's shape without
// the required label. mirror requires the container alone and schedule
// requires the container plus the backup trait, so mirror's predicate
// contains schedule's: every component schedule matches, mirror matches too,
// and no component's shape tells them apart (enhancement 0015 D5). Still
// routable — the container is catalog-fulfilled, so nothing is
// over-subscribed.
const undiscriminatedPlatform = routablePlatform + `
_veleroMirror: c.#ComponentTransformer & {
	metadata: {
		name: "mirror"
		fqn:  "testing.opmodel.dev/catalogs/velero/transformers/mirror@1.4.0"
	}
	requiredResources: (_container.metadata.fqn): _container
}

_veleroMirrorCatalog: c.#Catalog & {
	metadata: {
		modulePath: "testing.opmodel.dev/catalogs/velero@v1"
		version:    "1.4.0"
	}
	#transformers: (_veleroMirror.metadata.fqn): _veleroMirror
}

#registry: (_veleroMirrorCatalog.metadata.modulePath): #catalog: _veleroMirrorCatalog
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

// overSubscribedAndUndiscriminatedPlatform fails both gates at once, on
// different contracts: velero and k8up compete for the provider-fulfilled
// backup trait, and harbor's mirror contains k8up's schedule over the
// catalog-fulfilled container. velero's backup transformer requires no
// catalog-fulfilled contract, so it is in no comparable pair — the two
// refusals really are about different things.
const overSubscribedAndUndiscriminatedPlatform = overSubscribedPlatform + `
_harborMirror: c.#ComponentTransformer & {
	metadata: {
		name: "mirror"
		fqn:  "testing.opmodel.dev/catalogs/harbor/transformers/mirror@1.0.0"
	}
	requiredResources: (_container.metadata.fqn): _container
}

_harborCatalog: c.#Catalog & {
	metadata: {
		modulePath: "testing.opmodel.dev/catalogs/harbor@v1"
		version:    "1.0.0"
	}
	#transformers: (_harborMirror.metadata.fqn): _harborMirror
}

#registry: (_harborCatalog.metadata.modulePath): #catalog: _harborCatalog
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

// An unfulfilled contract is a report and never a gate (enhancement 0015
// D18): the command names it and still exits zero, because the refusal for an
// unmet demand belongs to the render that demands it.
func TestPlatformCheck_UnfulfilledPlatformExitsZero(t *testing.T) {
	dir := writePlatformDir(t, fixtureModule, baseOnlyPlatform)

	report, _, err := runCheck(t, dir)
	skipIfRegistryUnavailable(t, err)
	require.NoError(t, err, "an unfulfilled contract never changes the exit status")

	assert.Contains(t, report, "unfulfilled contracts: 1")
	assert.Contains(t, report, "testing.opmodel.dev/catalogs/base/traits/backup@v1alpha1 (defined by testing.opmodel.dev/catalogs/base@v1)")
	assert.Contains(t, report, "enhancement 0015 D18")
	assert.Contains(t, report, "fulfilled: no — 1 contract is unfulfilled")
	assert.Contains(t, report, "routable:  yes")
	assert.NotContains(t, report, "over-subscribed contracts")
}

// Over-subscription counts only PROVIDER-fulfilled contracts: any number of
// catalogs may require a catalog-fulfilled one, and the report shows them as
// implementations rather than as competitors.
func TestPlatformCheck_CatalogFulfilledPluralityIsNotOverSubscription(t *testing.T) {
	dir := writePlatformDir(t, fixtureModule, catalogFulfilledPluralityPlatform)

	report, _, err := runCheck(t, dir)
	skipIfRegistryUnavailable(t, err)
	require.NoError(t, err)

	assert.Contains(t, report, "implemented by  testing.opmodel.dev/catalogs/k8up/transformers/schedule@2.0.0, testing.opmodel.dev/catalogs/velero/transformers/mirror@1.4.0",
		"the catalog-fulfilled container admits any number of suppliers")
	assert.NotContains(t, report, "over-subscribed contracts")
	assert.Contains(t, report, "routable:  yes")
	// The suppliers are told apart by mirror's required label, so the
	// platform this assertion is made on is one the operator generates from
	// (enhancement 0015 D5).
	assert.Contains(t, report, "discriminated: yes")
	assert.NotContains(t, report, "comparable transformer pairs")
}

// A comparable pair is what platform-package generation refuses on
// (enhancement 0015 D5), so the pre-flight refuses it too — even though every
// contract here has exactly one supplier and the platform is routable.
func TestPlatformCheck_UndiscriminatedPlatformExitsValidation(t *testing.T) {
	dir := writePlatformDir(t, fixtureModule, undiscriminatedPlatform)

	report, _, err := runCheck(t, dir)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.True(t, exitErr.Printed, "the report already carries the verdict")

	assert.Contains(t, report, "comparable transformer pairs: 1")
	assert.Contains(t, report, "enhancement 0015")
	assert.Contains(t, report, "testing.opmodel.dev/catalogs/velero/transformers/mirror@1.4.0 (broader)")
	assert.Contains(t, report, "and  testing.opmodel.dev/catalogs/k8up/transformers/schedule@2.0.0 (narrower)")
	assert.Contains(t, report, "over  testing.opmodel.dev/catalogs/base/resources/container@v1beta1")
	assert.Contains(t, report, "routable:  yes", "a comparable pair is not over-subscription")
	assert.Contains(t, report, "discriminated: no — 1 pair is comparable")

	// Both counts are named whichever gate fired, so the message never
	// leaves the reader guessing which one is zero.
	assert.Contains(t, err.Error(), "0 over-subscribed contract(s), 1 comparable transformer pair(s)")
	assert.Contains(t, err.Error(), dir)
}

// The two refusals are independent: a platform failing both is reported under
// both headings and exits once.
func TestPlatformCheck_OverSubscribedAndUndiscriminatedExitsOnce(t *testing.T) {
	dir := writePlatformDir(t, fixtureModule, overSubscribedAndUndiscriminatedPlatform)

	report, _, err := runCheck(t, dir)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)

	assert.Contains(t, report, "over-subscribed contracts: 1")
	assert.Contains(t, report, "comparable transformer pairs: 1")
	assert.Contains(t, report, "testing.opmodel.dev/catalogs/harbor/transformers/mirror@1.0.0 (broader)")
	assert.Contains(t, report, "routable:  no — 1 contract is over-subscribed")
	assert.Contains(t, report, "discriminated: no — 1 pair is comparable")
	assert.Contains(t, err.Error(), "1 over-subscribed contract(s), 1 comparable transformer pair(s)")
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

// A report the inventory does not carry is never defaulted: core alpha.9
// derives the inventory but not its comparable-predicate report, and a
// missing `discriminated` defaulted to true would read as a pass on exactly
// the platforms this command now refuses.
func TestPlatformCheck_CoreWithoutTheComparableReportNamesTheRelease(t *testing.T) {
	dir := writePlatformDir(t, `module: "testing.opmodel.dev/platforms/check-fixture@v0"
language: version: "v0.17.0"
deps: "opmodel.dev/core@v2": v: "v2.0.0-alpha.9"
`, undiscriminatedPlatform)

	report, _, err := runCheck(t, dir)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Contains(t, err.Error(), `"comparable"`)
	assert.Contains(t, err.Error(), "2.0.0-alpha.10")
	assert.Contains(t, err.Error(), dir)
	assert.Empty(t, report, "no partial report is printed")
}
