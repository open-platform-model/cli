package modulereleasecueeval

// ---------------------------------------------------------------------------
// Decision 4: CUE computes metadata.uuid via uid.SHA1 after injection
//
// #ModuleRelease.metadata.uuid is defined as:
//   uuid: #UUIDType & uid.SHA1(OPMNamespace, "\(#moduleMetadata.fqn):\(name):\(namespace)")
//
// The uuid package is CUE stdlib (cuelang.org/go/pkg/uuid) — no registry needed.
// After filling #module, metadata.name, metadata.namespace, and values,
// the uuid field should be a concrete UUID5 string.
//
// These tests prove:
//   - metadata.uuid is concrete after full fill sequence
//   - The UUID is a valid UUID5 string format
//   - The UUID is deterministic: same inputs → same UUID
//   - Different releases produce different UUIDs
//   - The UUID matches what Go's crypto/sha1 would compute (parity check)
// ---------------------------------------------------------------------------

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// OPMNamespace is the UUID5 namespace used by opmodel.dev for identity computation.
	// From catalog/v0/core/common.cue: OPMNamespace: "11bc6112-a6e8-4021-bec9-b3ad246f9466"
	opmNamespace = "11bc6112-a6e8-4021-bec9-b3ad246f9466"
)

// TestUUID_IsConcrete proves that metadata.uuid is a concrete string after
// all required fields are filled.
func TestUUID_IsConcrete(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	result := fillRelease(schema, testModule, "my-release", "production", `{
		replicaCount: 3
		image:        "nginx:1.28"
	}`)
	require.NoError(t, result.Err())

	uuidVal := result.LookupPath(cue.ParsePath("metadata.uuid"))
	require.True(t, uuidVal.Exists())
	require.NoError(t, uuidVal.Err())

	uuid, err := uuidVal.String()
	require.NoError(t, err, "metadata.uuid should be a concrete string")

	t.Logf("Computed metadata.uuid: %s", uuid)
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, uuid,
		"uuid should match UUID format")
}

// TestUUID_IsDeterministic proves that the same inputs always produce the same UUID.
// Two independent fill sequences with identical inputs must yield the same uuid.
func TestUUID_IsDeterministic(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	values := `{replicaCount: 1, image: "nginx:latest"}`

	result1 := fillRelease(schema, testModule, "rel-a", "staging", values)
	require.NoError(t, result1.Err())
	uuid1, err := result1.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)

	result2 := fillRelease(schema, testModule, "rel-a", "staging", values)
	require.NoError(t, result2.Err())
	uuid2, err := result2.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)

	assert.Equal(t, uuid1, uuid2, "same inputs must produce the same UUID (deterministic)")
}

// TestUUID_DiffersByName proves that changing metadata.name produces a different UUID.
func TestUUID_DiffersByName(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	values := `{replicaCount: 1, image: "nginx:latest"}`

	result1 := fillRelease(schema, testModule, "release-one", "default", values)
	require.NoError(t, result1.Err())
	uuid1, err := result1.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)

	result2 := fillRelease(schema, testModule, "release-two", "default", values)
	require.NoError(t, result2.Err())
	uuid2, err := result2.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)

	assert.NotEqual(t, uuid1, uuid2, "different release names must produce different UUIDs")
}

// TestUUID_DiffersByNamespace proves that changing namespace produces a different UUID.
func TestUUID_DiffersByNamespace(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	values := `{replicaCount: 1, image: "nginx:latest"}`

	result1 := fillRelease(schema, testModule, "my-app", "staging", values)
	require.NoError(t, result1.Err())
	uuid1, err := result1.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)

	result2 := fillRelease(schema, testModule, "my-app", "production", values)
	require.NoError(t, result2.Err())
	uuid2, err := result2.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)

	assert.NotEqual(t, uuid1, uuid2, "different namespaces must produce different UUIDs")
}

// TestUUID_MatchesGoComputation proves parity between CUE's uid.SHA1 and a
// Go-side UUID5 computation using the same inputs. This validates that the
// CUE stdlib uuid package implements the same RFC 4122 UUID5 algorithm.
//
// The formula: SHA1(OPMNamespace, "\(fqn):\(name):\(namespace)")
func TestUUID_MatchesGoComputation(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	// Fill the release and get the CUE-computed UUID.
	result := fillRelease(schema, testModule, "my-release", "default", `{
		replicaCount: 1
		image:        "nginx:latest"
	}`)
	require.NoError(t, result.Err())

	cueUUID, err := result.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)

	// Get the FQN that CUE computed for _testModule.
	// _testModule.metadata.fqn is: "test.module.dev/modules@v0#TestModule"
	fqn, err := testModule.LookupPath(cue.ParsePath("metadata.fqn")).String()
	require.NoError(t, err, "testModule.metadata.fqn should be concrete")
	t.Logf("testModule FQN: %s", fqn)

	// Compute expected UUID using Go's sha1.
	input := fmt.Sprintf("%s:%s:%s", fqn, "my-release", "default")
	goUUID := computeUUID5(opmNamespace, input)
	t.Logf("Go-computed UUID5: %s", goUUID)
	t.Logf("CUE-computed UUID: %s", cueUUID)

	assert.Equal(t, goUUID, cueUUID,
		"CUE uid.SHA1 must match Go's UUID5 computation with the same inputs")
}

// computeUUID5 computes a UUID version 5 (SHA1-based) from a namespace UUID string
// and a name string, following RFC 4122.
func computeUUID5(namespaceStr, name string) string {
	// Parse namespace UUID into 16 bytes.
	nsHex := strings.ReplaceAll(namespaceStr, "-", "")
	nsBytes, err := hex.DecodeString(nsHex)
	if err != nil || len(nsBytes) != 16 {
		panic(fmt.Sprintf("invalid namespace UUID: %s", namespaceStr))
	}

	h := sha1.New()
	h.Write(nsBytes)
	h.Write([]byte(name))
	hash := h.Sum(nil)

	// Set version (5) and variant bits per RFC 4122.
	hash[6] = (hash[6] & 0x0f) | 0x50 // version 5
	hash[8] = (hash[8] & 0x3f) | 0x80 // variant bits

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(hash[0:4]),
		binary.BigEndian.Uint16(hash[4:6]),
		binary.BigEndian.Uint16(hash[6:8]),
		binary.BigEndian.Uint16(hash[8:10]),
		hash[10:16],
	)
}
