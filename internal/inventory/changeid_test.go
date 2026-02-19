package inventory

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ComputeChangeID ---

func TestComputeChangeID_Format(t *testing.T) {
	id := ComputeChangeID("opmodel.dev/modules/jellyfin", "1.0.0", `{port: 8096}`, "sha256:abc123")
	assert.True(t, strings.HasPrefix(id, "change-sha1-"), "change ID should start with 'change-sha1-'")
	// "change-sha1-" is 12 chars, then 8 hex chars
	assert.Len(t, id, 12+8, "change ID should be change-sha1- + 8 hex chars")
}

func TestComputeChangeID_Deterministic(t *testing.T) {
	id1 := ComputeChangeID("opmodel.dev/modules/jellyfin", "1.0.0", `{port: 8096}`, "sha256:abc123def456")
	id2 := ComputeChangeID("opmodel.dev/modules/jellyfin", "1.0.0", `{port: 8096}`, "sha256:abc123def456")
	assert.Equal(t, id1, id2, "same inputs should produce same change ID")
}

func TestComputeChangeID_VersionBumpProducesDifferentID(t *testing.T) {
	id1 := ComputeChangeID("opmodel.dev/modules/jellyfin", "1.0.0", `{port: 8096}`, "sha256:abc123")
	id2 := ComputeChangeID("opmodel.dev/modules/jellyfin", "1.1.0", `{port: 8096}`, "sha256:abc123")
	assert.NotEqual(t, id1, id2, "version bump should produce different change ID")
}

func TestComputeChangeID_ValuesChangeProducesDifferentID(t *testing.T) {
	id1 := ComputeChangeID("path", "1.0.0", `{port: 8096}`, "sha256:abc123")
	id2 := ComputeChangeID("path", "1.0.0", `{port: 9000}`, "sha256:abc123")
	assert.NotEqual(t, id1, id2, "values change should produce different change ID")
}

func TestComputeChangeID_DigestChangeProducesDifferentID(t *testing.T) {
	id1 := ComputeChangeID("path", "1.0.0", `{}`, "sha256:aaa111")
	id2 := ComputeChangeID("path", "1.0.0", `{}`, "sha256:bbb222")
	assert.NotEqual(t, id1, id2, "digest change should produce different change ID")
}

func TestComputeChangeID_LocalModuleUsesEmptyVersion(t *testing.T) {
	// Local modules pass empty version string — this should be stable
	id := ComputeChangeID("./my-local-module", "", `{port: 8096}`, "sha256:abc123")
	assert.True(t, strings.HasPrefix(id, "change-sha1-"))

	// Stable across calls
	id2 := ComputeChangeID("./my-local-module", "", `{port: 8096}`, "sha256:abc123")
	assert.Equal(t, id, id2)
}

// --- UpdateIndex ---

func TestUpdateIndex_NewID_PrependedToFront(t *testing.T) {
	index := []string{"change-sha1-bbb22222"}
	result := UpdateIndex(index, "change-sha1-aaa11111")
	assert.Equal(t, []string{"change-sha1-aaa11111", "change-sha1-bbb22222"}, result)
}

func TestUpdateIndex_ReapplyMovesToFront(t *testing.T) {
	index := []string{"change-sha1-aaa11111", "change-sha1-bbb22222"}
	result := UpdateIndex(index, "change-sha1-bbb22222")
	assert.Equal(t, []string{"change-sha1-bbb22222", "change-sha1-aaa11111"}, result,
		"existing ID should move to front, not be duplicated")
}

func TestUpdateIndex_EmptyIndex(t *testing.T) {
	result := UpdateIndex([]string{}, "change-sha1-aaa11111")
	assert.Equal(t, []string{"change-sha1-aaa11111"}, result)
}

func TestUpdateIndex_DoesNotMutateInput(t *testing.T) {
	input := []string{"change-sha1-aaa", "change-sha1-bbb"}
	original := make([]string, len(input))
	copy(original, input)

	UpdateIndex(input, "change-sha1-ccc")
	assert.Equal(t, original, input, "UpdateIndex should not mutate the input slice")
}

func TestUpdateIndex_NoDuplicate_OnReapply(t *testing.T) {
	index := []string{"change-sha1-aaa", "change-sha1-bbb", "change-sha1-ccc"}
	result := UpdateIndex(index, "change-sha1-bbb")
	// bbb moved to front, no duplicate
	assert.Equal(t, []string{"change-sha1-bbb", "change-sha1-aaa", "change-sha1-ccc"}, result)
	assert.Len(t, result, 3, "no duplicate entries")
}

// --- PruneHistory ---

func TestPruneHistory_RemovesOldestEntries(t *testing.T) {
	secret := &InventorySecret{
		Index: []string{"id-1", "id-2", "id-3", "id-4"},
		Changes: map[string]*ChangeEntry{
			"id-1": {Timestamp: "newest"},
			"id-2": {},
			"id-3": {},
			"id-4": {Timestamp: "oldest"},
		},
	}

	PruneHistory(secret, 2)

	assert.Equal(t, []string{"id-1", "id-2"}, secret.Index)
	assert.Len(t, secret.Changes, 2)
	assert.Contains(t, secret.Changes, "id-1")
	assert.Contains(t, secret.Changes, "id-2")
	assert.NotContains(t, secret.Changes, "id-3")
	assert.NotContains(t, secret.Changes, "id-4")
}

func TestPruneHistory_NoOpWhenUnderLimit(t *testing.T) {
	secret := &InventorySecret{
		Index: []string{"id-1", "id-2"},
		Changes: map[string]*ChangeEntry{
			"id-1": {},
			"id-2": {},
		},
	}

	PruneHistory(secret, 10)
	assert.Len(t, secret.Index, 2)
	assert.Len(t, secret.Changes, 2)
}

func TestPruneHistory_ExactlyAtLimit(t *testing.T) {
	secret := &InventorySecret{
		Index:   []string{"id-1", "id-2", "id-3"},
		Changes: map[string]*ChangeEntry{"id-1": {}, "id-2": {}, "id-3": {}},
	}
	PruneHistory(secret, 3)
	assert.Len(t, secret.Index, 3, "at limit: no pruning")
}

func TestPruneHistory_ZeroMaxHistoryNoOp(t *testing.T) {
	secret := &InventorySecret{
		Index:   []string{"id-1"},
		Changes: map[string]*ChangeEntry{"id-1": {}},
	}
	PruneHistory(secret, 0)
	// maxHistory=0 is treated as no-op (guard against invalid config)
	assert.Len(t, secret.Index, 1)
}

// --- PrepareChange ---

func TestPrepareChange_ProducesValidEntry(t *testing.T) {
	source := ChangeSource{
		Path:        "opmodel.dev/modules/jellyfin",
		Version:     "1.0.0",
		ReleaseName: "jellyfin",
	}
	entries := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "media", Name: "jellyfin", Version: "v1", Component: "app"},
	}

	changeID, entry := PrepareChange(source, `{port: 8096}`, "sha256:abc123", entries)

	require.NotEmpty(t, changeID)
	assert.True(t, strings.HasPrefix(changeID, "change-sha1-"))
	assert.Equal(t, source, entry.Source)
	assert.Equal(t, `{port: 8096}`, entry.Values)
	assert.Equal(t, "sha256:abc123", entry.ManifestDigest)
	assert.NotEmpty(t, entry.Timestamp)
	assert.Equal(t, entries, entry.Inventory.Entries)
}

func TestPrepareChange_LocalModule(t *testing.T) {
	source := ChangeSource{
		Path:        "./my-module",
		ReleaseName: "my-module",
		Local:       true,
		// Version intentionally empty for local modules
	}

	changeID, entry := PrepareChange(source, `{}`, "sha256:digest", nil)

	assert.True(t, strings.HasPrefix(changeID, "change-sha1-"))
	assert.True(t, entry.Source.Local)
	assert.Empty(t, entry.Source.Version)
}

func TestPrepareChange_IDMatchesComputeChangeID(t *testing.T) {
	source := ChangeSource{Path: "path", Version: "v1", ReleaseName: "test"}
	values := `{replicas: 3}`
	digest := "sha256:abc"

	changeID, _ := PrepareChange(source, values, digest, nil)
	expectedID := ComputeChangeID(source.Path, source.Version, values, digest)

	assert.Equal(t, expectedID, changeID)
}
