package inventory

import (
	"crypto/sha1" //nolint:gosec // SHA1 is used for non-cryptographic change ID generation only
	"fmt"
	"time"
)

// ComputeChangeID computes a deterministic change ID from four inputs:
// module path, module version, resolved values string, and manifest digest.
//
// Format: "change-sha1-<8hex>" (first 8 hex chars of SHA1).
//
// All four inputs are included to ensure:
//   - Module upgrades produce new change IDs even with identical rendered output
//   - Explicit value changes are recorded as distinct changes
//   - Content changes always produce different IDs
//
// For local modules, moduleVersion is the empty string.
func ComputeChangeID(modulePath, moduleVersion, values, manifestDigest string) string {
	h := sha1.New() //nolint:gosec // not used for security, only for change fingerprinting
	h.Write([]byte(modulePath))
	h.Write([]byte(moduleVersion))
	h.Write([]byte(values))
	h.Write([]byte(manifestDigest))
	sum := h.Sum(nil)
	return fmt.Sprintf("change-sha1-%08x", sum[:4])
}

// UpdateIndex adds changeID to the front of the index.
// If the changeID already exists, it is removed from its current position
// and prepended — implementing "move to front" for idempotent re-applies.
// The original slice is not modified; a new slice is returned.
func UpdateIndex(index []string, changeID string) []string {
	// Remove existing occurrence (if any)
	filtered := make([]string, 0, len(index))
	for _, id := range index {
		if id != changeID {
			filtered = append(filtered, id)
		}
	}
	// Prepend the new/moved ID
	return append([]string{changeID}, filtered...)
}

// PruneHistory removes the oldest change entries from the inventory when
// the index exceeds maxHistory. Entries are removed from the tail of the index
// (oldest first). Both the index and the Changes map are updated in place.
func PruneHistory(secret *InventorySecret, maxHistory int) {
	if maxHistory <= 0 || len(secret.Index) <= maxHistory {
		return
	}

	// Entries to remove: the tail beyond maxHistory
	toRemove := secret.Index[maxHistory:]
	for _, id := range toRemove {
		delete(secret.Changes, id)
	}
	secret.Index = secret.Index[:maxHistory]
}

// PrepareChange computes a change ID and builds a ChangeEntry for the current apply.
// The timestamp is set to the current UTC time in RFC 3339 format.
// Returns (changeID, changeEntry).
func PrepareChange(module ModuleRef, values, manifestDigest string, entries []InventoryEntry) (string, *ChangeEntry) {
	changeID := ComputeChangeID(module.Path, module.Version, values, manifestDigest)

	entry := &ChangeEntry{
		Module:         module,
		Values:         values,
		ManifestDigest: manifestDigest,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Inventory: InventoryList{
			Entries: entries,
		},
	}

	return changeID, entry
}
