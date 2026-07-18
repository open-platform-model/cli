package inventory

import (
	pkginventory "github.com/open-platform-model/cli/pkg/inventory"
)

// The functions below map between the CLI's pkg/inventory types and the
// ModuleInstance CRD's status.inventory object shape. Conversion targets the
// CRD's OpenAPI field names (group/kind/namespace/name/v/component and
// revision/digest/count/entries) explicitly — never Go struct-tag marshaling —
// because the CRD schema, not the Go tags, anchors cross-actor shape parity
// (enhancement 0006 D2/D31). All integer values use int64, the only integer
// type the unstructured converter accepts.

// entryToWire converts an InventoryEntry into the CRD entry object. Optional
// fields absent from the entry are omitted, matching the CRD's omitempty
// semantics; kind and name (the CRD's required fields) are always present.
func entryToWire(e pkginventory.InventoryEntry) map[string]any {
	m := map[string]any{
		"kind": e.Kind,
		"name": e.Name,
	}
	if e.Group != "" {
		m["group"] = e.Group
	}
	if e.Namespace != "" {
		m["namespace"] = e.Namespace
	}
	if e.Version != "" {
		m["v"] = e.Version
	}
	if e.Component != "" {
		m["component"] = e.Component
	}
	return m
}

// entryFromWire reconstructs an InventoryEntry from a CRD entry object.
func entryFromWire(m map[string]any) pkginventory.InventoryEntry {
	return pkginventory.InventoryEntry{
		Group:     wireString(m, "group"),
		Kind:      wireString(m, "kind"),
		Namespace: wireString(m, "namespace"),
		Name:      wireString(m, "name"),
		Version:   wireString(m, "v"),
		Component: wireString(m, "component"),
	}
}

// inventoryToWire converts an Inventory block into the CRD's status.inventory
// object.
func inventoryToWire(inv pkginventory.Inventory) map[string]any {
	entries := make([]any, 0, len(inv.Entries))
	for _, e := range inv.Entries {
		entries = append(entries, entryToWire(e))
	}
	return map[string]any{
		"revision": int64(inv.Revision),
		"digest":   inv.Digest,
		"count":    int64(inv.Count),
		"entries":  entries,
	}
}

// inventoryFromWire reconstructs an Inventory block from the CRD's
// status.inventory object. A nil or absent object yields an empty inventory
// with a non-nil entries slice (the invariant callers rely on).
func inventoryFromWire(m map[string]any) pkginventory.Inventory {
	inv := pkginventory.Inventory{
		Revision: wireInt(m, "revision"),
		Digest:   wireString(m, "digest"),
		Count:    wireInt(m, "count"),
		Entries:  []pkginventory.InventoryEntry{},
	}
	if raw, ok := m["entries"].([]any); ok {
		for _, item := range raw {
			if em, ok := item.(map[string]any); ok {
				inv.Entries = append(inv.Entries, entryFromWire(em))
			}
		}
	}
	return inv
}

func wireString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func wireInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int64:
		return int(v)
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}
