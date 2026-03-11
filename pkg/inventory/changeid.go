package inventory

import (
	"crypto/sha1" //nolint:gosec // SHA1 is used for non-cryptographic change ID generation only
	"fmt"
	"time"
)

func ComputeChangeID(modulePath, moduleVersion, values, manifestDigest string) string {
	h := sha1.New() //nolint:gosec // not used for security, only for change fingerprinting
	h.Write([]byte(modulePath))
	h.Write([]byte(moduleVersion))
	h.Write([]byte(values))
	h.Write([]byte(manifestDigest))
	sum := h.Sum(nil)
	return fmt.Sprintf("change-sha1-%08x", sum[:4])
}

func UpdateIndex(index []string, changeID string) []string {
	filtered := make([]string, 0, len(index))
	for _, id := range index {
		if id != changeID {
			filtered = append(filtered, id)
		}
	}
	return append([]string{changeID}, filtered...)
}

func PruneHistory(secret *InventorySecret, maxHistory int) {
	if maxHistory <= 0 || len(secret.Index) <= maxHistory {
		return
	}

	toRemove := secret.Index[maxHistory:]
	for _, id := range toRemove {
		delete(secret.Changes, id)
	}
	secret.Index = secret.Index[:maxHistory]
}

func PrepareChange(source ChangeSource, values, manifestDigest string, entries []InventoryEntry) (string, *ChangeEntry) {
	changeID := ComputeChangeID(source.Path, source.Version, values, manifestDigest)

	entry := &ChangeEntry{
		Source:         source,
		Values:         values,
		ManifestDigest: manifestDigest,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Inventory: InventoryList{
			Entries: entries,
		},
	}

	return changeID, entry
}
