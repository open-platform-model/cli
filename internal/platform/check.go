package platform

import (
	"fmt"
	"sort"
	"strings"

	libplatform "github.com/open-platform-model/library/opm/platform"
)

// Report is what `opm platform check` prints: the contract inventory core
// derives for a platform (enhancement 0015 D1, D2, D18), plus the provenance
// of the platform it was read from.
//
// Every field is a report. A platform that is not fulfilled or not routable
// is still a healthy value here; only [Report.Routable] decides the command's
// exit status, because platform-package generation is the step that refuses
// on it.
type Report struct {
	// Resolution is where the checked platform came from, printed as the
	// report's first line so a report can never be read as describing a
	// different platform.
	Resolution Resolution

	// Defined maps each contract FQN an enabled catalog lists to that
	// catalog's registry key (the inventory's definedBy, read off the
	// build — never parsed out of the FQN).
	Defined map[string]string

	// RequiredBy maps each defined contract FQN to the implementation FQNs
	// of the enabled transformers that require it. A contract nothing
	// requires maps to an empty list.
	RequiredBy map[string][]string

	// Unfulfilled lists the provider-fulfilled contracts no enabled
	// transformer implements.
	Unfulfilled []string

	// OverSubscribed lists the provider-fulfilled contracts required by
	// transformers from more than one catalog.
	OverSubscribed []string

	// fulfilled and routable are the inventory's own verdicts, kept
	// unexported so the two lists and their verdicts cannot drift apart in
	// a caller-built Report: NewReport is the only constructor.
	fulfilled bool
	routable  bool
}

// NewReport builds the report for a platform resolved as res from the
// inventory read off its build.
func NewReport(res Resolution, inv *libplatform.ContractInventory) Report {
	return Report{
		Resolution:     res,
		Defined:        inv.DefinedBy,
		RequiredBy:     inv.RequiredBy,
		Unfulfilled:    inv.Unfulfilled,
		OverSubscribed: inv.OverSubscribed,
		fulfilled:      inv.Fulfilled,
		routable:       inv.Routable,
	}
}

// Routable reports whether a platform package may be generated from this
// platform. It is the only value that decides `opm platform check`'s exit
// status: enhancement 0015 D18 makes an unfulfilled contract a report and
// never a gate, so a report that is not fulfilled still exits zero.
func (r Report) Routable() bool { return r.routable }

// Render returns the report text.
func (r Report) Render() string {
	var b strings.Builder
	b.WriteString(r.Resolution.Describe() + "\n")

	if len(r.Defined) == 0 {
		b.WriteString("\nthe enabled catalogs define no contracts — nothing was verified here.\n" +
			"A platform whose catalogs populate no contract maps produces an empty\n" +
			"inventory, which is not the same answer as a platform whose contracts all\n" +
			"check out.\n")
		b.WriteString("\nfulfilled: yes (vacuously — no contract is defined)\n")
		b.WriteString("routable:  yes (vacuously — no contract is defined)\n")
		return strings.TrimRight(b.String(), "\n")
	}

	fmt.Fprintf(&b, "\ndefined contracts: %d\n", len(r.Defined))
	for _, fqn := range sortedKeys(r.Defined) {
		fmt.Fprintf(&b, "  %s\n", fqn)
		fmt.Fprintf(&b, "    defined by      %s\n", r.Defined[fqn])
		fmt.Fprintf(&b, "    implemented by  %s\n", joinOrNothing(r.RequiredBy[fqn]))
	}

	if len(r.Unfulfilled) > 0 {
		fmt.Fprintf(&b, "\nunfulfilled contracts: %d\n", len(r.Unfulfilled))
		b.WriteString("  Provider-fulfilled contracts no enabled transformer implements. These do\n" +
			"  not fail this check (enhancement 0015 D18): a platform may define a\n" +
			"  contract ahead of the provider that implements it, and an unmet demand is\n" +
			"  refused by the render that demands it.\n")
		for _, fqn := range sortedCopy(r.Unfulfilled) {
			fmt.Fprintf(&b, "  %s%s\n", fqn, definedByClause(r.Defined[fqn]))
		}
	}

	if len(r.OverSubscribed) > 0 {
		fmt.Fprintf(&b, "\nover-subscribed contracts: %d\n", len(r.OverSubscribed))
		b.WriteString("  Provider-fulfilled contracts required by transformers from more than one\n" +
			"  catalog. A platform package cannot be generated from this platform until\n" +
			"  one of the competing catalogs is disabled.\n")
		for _, fqn := range sortedCopy(r.OverSubscribed) {
			fmt.Fprintf(&b, "  %s%s\n", fqn, definedByClause(r.Defined[fqn]))
			fmt.Fprintf(&b, "    required by  %s\n", joinOrNothing(r.RequiredBy[fqn]))
		}
	}

	b.WriteString("\n" + verdictLine("fulfilled", r.fulfilled, len(r.Unfulfilled), "unfulfilled"))
	b.WriteString(verdictLine("routable", r.routable, len(r.OverSubscribed), "over-subscribed"))
	return strings.TrimRight(b.String(), "\n")
}

// verdictLine words one of the inventory's two booleans, naming the list that
// produced it when the answer is no.
func verdictLine(label string, ok bool, n int, noun string) string {
	if ok {
		return fmt.Sprintf("%-10s yes\n", label+":")
	}
	verb := "is"
	contracts := "contract"
	if n != 1 {
		verb, contracts = "are", "contracts"
	}
	return fmt.Sprintf("%-10s no — %d %s %s %s\n", label+":", n, contracts, verb, noun)
}

// definedByClause names the defining catalog when the contract carries one.
func definedByClause(catalog string) string {
	if catalog == "" {
		return ""
	}
	return " (defined by " + catalog + ")"
}

// joinOrNothing joins the implementation FQNs, or says nothing implements it.
func joinOrNothing(impls []string) string {
	if len(impls) == 0 {
		return "nothing on this platform"
	}
	return strings.Join(sortedCopy(impls), ", ")
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}
