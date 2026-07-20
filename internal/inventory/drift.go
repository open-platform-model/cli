package inventory

import (
	"fmt"
	"sort"
	"strings"
)

// DescribeEntrySetDrift compares two inventory entry lists as sets and returns
// a human-readable description of the difference, or "" when the sets are
// equal. Order is irrelevant — entries are identity tuples, and the two actors
// have no reason to agree on ordering.
//
// This is the observable behind enhancement 0006 D40's inventory-stable
// criterion: a handoff must not change which resources an instance owns. It
// deliberately compares identity only, not content — D7.4's pre-flip digest
// gate already established content parity, and the operator's relabel changes
// content on every resource by design.
func DescribeEntrySetDrift(before, after []InventoryEntry) string {
	added := entriesMissingFrom(after, before)
	removed := entriesMissingFrom(before, after)

	if len(added) == 0 && len(removed) == 0 {
		return ""
	}

	var parts []string
	if len(removed) > 0 {
		parts = append(parts, fmt.Sprintf("%d resource(s) no longer tracked (%s)", len(removed), strings.Join(removed, ", ")))
	}
	if len(added) > 0 {
		parts = append(parts, fmt.Sprintf("%d newly tracked resource(s) (%s)", len(added), strings.Join(added, ", ")))
	}
	return strings.Join(parts, "; ")
}

// entriesMissingFrom returns descriptions of the entries in candidates that
// have no identity match in reference, sorted for deterministic output.
func entriesMissingFrom(candidates, reference []InventoryEntry) []string {
	var missing []string
	for _, c := range candidates {
		found := false
		for _, r := range reference {
			if IdentityEqual(c, r) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, DescribeEntry(c))
		}
	}
	sort.Strings(missing)
	return missing
}

// DescribeEntry renders an inventory entry as "Kind/namespace/name".
func DescribeEntry(e InventoryEntry) string {
	if e.Namespace == "" {
		return e.Kind + "/" + e.Name
	}
	return e.Kind + "/" + e.Namespace + "/" + e.Name
}
