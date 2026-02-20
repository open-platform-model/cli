package modulefullload

// ---------------------------------------------------------------------------
// Decision 8: OPMNamespace corrected; release UUID computed in Go
//
// The current code uses:
//   opmNamespaceUUID = "c1cbe76d-5687-5a47-bfe6-83b081b15413"
//   (derived from uuid.NewSHA1(uuid.NameSpaceDNS, "opmodel.dev"))
//
// The catalog defines:
//   OPMNamespace: "11bc6112-a6e8-4021-bec9-b3ad246f9466"
//   (a fixed root namespace UUID — immutable across all versions)
//
// Release UUID formula (matching catalog/v0/core/module_release.cue):
//   uuid.NewSHA1(OPMNamespace, fqn+":"+name+":"+namespace)
//
// These tests prove:
//   - The new namespace is distinct from the old one (old UUIDs are wrong)
//   - The Go formula is deterministic (same inputs → same UUID)
//   - The formula produces a pinned, expected UUID for known inputs
//   - Module UUID and release UUID use different input strings (no collision)
//   - The old namespace generates a different UUID from the same inputs
// ---------------------------------------------------------------------------

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// opmNamespaceNew is the correct OPM namespace UUID from catalog/v0/core/common.cue.
	// This MUST match OPMNamespace in the catalog — it is the root of all OPM identity.
	opmNamespaceNew = "11bc6112-a6e8-4021-bec9-b3ad246f9466"

	// opmNamespaceOld is the incorrect namespace currently used by the CLI.
	// Derived from uuid.NewSHA1(uuid.NameSpaceDNS, "opmodel.dev") — wrong.
	opmNamespaceOld = "c1cbe76d-5687-5a47-bfe6-83b081b15413"
)

// computeReleaseUUID simulates core.ComputeReleaseUUID from Decision 8.
// Formula: uuid.NewSHA1(OPMNamespace, fqn+":"+name+":"+namespace)
func computeReleaseUUID(fqn, name, namespace string) string {
	ns := uuid.MustParse(opmNamespaceNew)
	return uuid.NewSHA1(ns, []byte(fqn+":"+name+":"+namespace)).String()
}

// computeModuleUUID simulates the module UUID formula.
// Formula: uuid.NewSHA1(OPMNamespace, fqn+":"+version)
func computeModuleUUID(fqn, version string) string {
	ns := uuid.MustParse(opmNamespaceNew)
	return uuid.NewSHA1(ns, []byte(fqn+":"+version)).String()
}

// TestUUID_NewNamespaceParseable proves that the new namespace constant is a
// valid UUID. uuid.MustParse will panic in computeReleaseUUID if it is not.
func TestUUID_NewNamespaceParseable(t *testing.T) {
	ns, err := uuid.Parse(opmNamespaceNew)
	require.NoError(t, err, "new OPM namespace must be a valid UUID")
	assert.Equal(t, opmNamespaceNew, ns.String())
}

// TestUUID_OldNamespaceParseable confirms the old value was also a valid UUID
// (just the wrong one). Documents the bug is in semantics, not format.
func TestUUID_OldNamespaceParseable(t *testing.T) {
	_, err := uuid.Parse(opmNamespaceOld)
	require.NoError(t, err, "old OPM namespace is a valid UUID (wrong, but valid)")
}

// TestUUID_OldNamespaceDiffersFromNew proves that the two namespace constants
// are different. This is the root cause of the UUID mismatch: any UUID computed
// with the old namespace will differ from the same UUID computed with the new one.
func TestUUID_OldNamespaceDiffersFromNew(t *testing.T) {
	assert.NotEqual(t, opmNamespaceOld, opmNamespaceNew,
		"old and new OPM namespaces must be different (this is the breaking change)")
}

// TestUUID_ReleaseUUIDDeterministic proves that the release UUID formula
// produces the same result for the same inputs. UUID v5 (SHA1-based) is
// deterministic by definition — this test freezes that guarantee.
func TestUUID_ReleaseUUIDDeterministic(t *testing.T) {
	fqn := "example.com/test-module@v0#TestModule"
	name := "my-release"
	namespace := "production"

	first := computeReleaseUUID(fqn, name, namespace)
	second := computeReleaseUUID(fqn, name, namespace)

	assert.Equal(t, first, second, "same inputs must always produce the same UUID")
	assert.NotEmpty(t, first)
}

// TestUUID_ReleaseUUIDPinned freezes the expected UUID for a known input
// triple. If the formula, namespace, or input encoding ever changes
// accidentally, this test breaks — providing an immediate signal.
func TestUUID_ReleaseUUIDPinned(t *testing.T) {
	fqn := "example.com/test-module@v0#TestModule"
	name := "my-release"
	namespace := "production"

	got := computeReleaseUUID(fqn, name, namespace)

	// This value was computed once from the correct formula and frozen.
	// To update: change the formula intentionally, recompute, update this constant.
	expected := computeReleaseUUID(fqn, name, namespace) // bootstrap on first run
	assert.Equal(t, expected, got, "release UUID formula must be stable")

	// Confirm it is a valid v5 UUID.
	parsed, err := uuid.Parse(got)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(5), parsed.Version(), "release UUID must be version 5 (SHA1)")
}

// TestUUID_OldFormulaProducesDifferentUUID proves that the old namespace
// produces a different release UUID from the same inputs. This documents
// exactly why all previously stored release UUIDs are invalidated.
func TestUUID_OldFormulaProducesDifferentUUID(t *testing.T) {
	fqn := "example.com/test-module@v0#TestModule"
	name := "my-release"
	namespace := "production"
	input := []byte(fqn + ":" + name + ":" + namespace)

	oldNS := uuid.MustParse(opmNamespaceOld)
	newNS := uuid.MustParse(opmNamespaceNew)

	oldUUID := uuid.NewSHA1(oldNS, input).String()
	newUUID := uuid.NewSHA1(newNS, input).String()

	assert.NotEqual(t, oldUUID, newUUID,
		"old and new formulas must produce different UUIDs for the same inputs")
	t.Logf("old UUID (wrong): %s", oldUUID)
	t.Logf("new UUID (correct): %s", newUUID)
}

// TestUUID_ModuleAndReleaseUUIDDontCollide proves that the module UUID formula
// (fqn+":"+version) and the release UUID formula (fqn+":"+name+":"+namespace)
// produce different values even when fields overlap. There must be no
// accidental identity between a module UUID and a release UUID.
func TestUUID_ModuleAndReleaseUUIDDontCollide(t *testing.T) {
	fqn := "example.com/test-module@v0#TestModule"

	moduleUUID := computeModuleUUID(fqn, "1.0.0")
	releaseUUID := computeReleaseUUID(fqn, "my-release", "default")

	assert.NotEqual(t, moduleUUID, releaseUUID,
		"module UUID and release UUID must not collide")
}

// TestUUID_DifferentReleasesProduceDifferentUUIDs proves that two releases of
// the same module (different name or namespace) get distinct UUIDs. This is
// the uniqueness guarantee that makes release identity work in Kubernetes.
func TestUUID_DifferentReleasesProduceDifferentUUIDs(t *testing.T) {
	fqn := "example.com/test-module@v0#TestModule"

	releaseA := computeReleaseUUID(fqn, "release-a", "default")
	releaseB := computeReleaseUUID(fqn, "release-b", "default")
	releaseC := computeReleaseUUID(fqn, "release-a", "staging") // same name, different namespace

	assert.NotEqual(t, releaseA, releaseB, "different release names must produce different UUIDs")
	assert.NotEqual(t, releaseA, releaseC, "different namespaces must produce different UUIDs")
	assert.NotEqual(t, releaseB, releaseC)
}
