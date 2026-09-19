package platform

import (
	"fmt"
	"sort"
	"strings"

	libplatform "github.com/open-platform-model/library/opm/platform"
)

// Report is what `opm platform check` prints: the contract inventory core
// derives for a platform (0015:D1, D2, D5, D18), plus the
// provenance of the platform it was read from.
//
// Every field is a report. A platform that is not fulfilled, not routable or
// not discriminated is still a healthy value here; [Report.Routable] and
// [Report.Discriminated] are the two that decide the command's exit status,
// because platform-package generation is the step that refuses on both.
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

	// Comparable lists every pair of enabled transformers whose match
	// predicates are comparable over a shared catalog-fulfilled contract
	// (0015:D5). The library's row type is carried directly:
	// the rows are data this report only prints, so a CLI copy would exist
	// only to be converted into.
	Comparable []libplatform.ComparablePredicates

	// fulfilled, routable and discriminated are the inventory's own
	// verdicts, kept unexported so the three lists and their verdicts
	// cannot drift apart in a caller-built Report: NewReport is the only
	// constructor.
	fulfilled     bool
	routable      bool
	discriminated bool
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
		Comparable:     inv.Comparable,
		fulfilled:      inv.Fulfilled,
		routable:       inv.Routable,
		discriminated:  inv.Discriminated,
	}
}

// Routable reports whether any provider-fulfilled contract is over-subscribed.
// It is one of the two values that decide `opm platform check`'s exit status,
// [Report.Discriminated] being the other: 0015:D18 makes an
// unfulfilled contract a report and never a gate, so a report that is not
// fulfilled still exits zero.
func (r Report) Routable() bool { return r.routable }

// Discriminated reports whether every pair of enabled transformers is told
// apart by some component's shape. It is false exactly when [Report.Comparable]
// carries a row, and it is the second value deciding the command's exit
// status: platform-package generation refuses an undiscriminated platform
// exactly as it refuses an over-subscribed one (0015:D5).
func (r Report) Discriminated() bool { return r.discriminated }

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
		b.WriteString("discriminated: yes (vacuously — no contract is defined)\n")
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
			"  not fail this check: a platform may define a contract ahead of the\n" +
			"  provider that implements it, and an unmet demand is\n" +
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

	if len(r.Comparable) > 0 {
		fmt.Fprintf(&b, "\ncomparable transformer pairs: %d\n", len(r.Comparable))
		b.WriteString("  Enabled transformers whose match predicates are comparable over a shared\n" +
			"  catalog-fulfilled contract: every component the narrower one matches is\n" +
			"  also matched by the broader one, so both would render. A platform\n" +
			"  package cannot be generated from this platform until one\n" +
			"  catalog is disabled or the transformers are discriminated by a required\n" +
			"  label value or a required trait.\n")
		for _, row := range sortedRows(r.Comparable) {
			fmt.Fprintf(&b, "  %s (broader)\n", row.Broader)
			fmt.Fprintf(&b, "    and  %s (narrower)\n", row.Narrower)
			fmt.Fprintf(&b, "    over  %s\n", strings.Join(sortedCopy(row.Contracts), ", "))
		}
	}

	b.WriteString("\n" + verdictLine("fulfilled", r.fulfilled, len(r.Unfulfilled), "contract", "unfulfilled"))
	b.WriteString(verdictLine("routable", r.routable, len(r.OverSubscribed), "contract", "over-subscribed"))
	b.WriteString(verdictLine("discriminated", r.discriminated, len(r.Comparable), "pair", "comparable"))
	return strings.TrimRight(b.String(), "\n")
}

// verdictLine words one of the inventory's three booleans, naming the list
// that produced it when the answer is no. noun is that list's unit
// ("contract", "pair"); condition is what its members are. The label is
// padded to the width of the first two, so the longer "discriminated:" runs
// past the pad and prints with a single space.
func verdictLine(label string, ok bool, n int, noun, condition string) string {
	if ok {
		return fmt.Sprintf("%-10s yes\n", label+":")
	}
	verb := "is"
	if n != 1 {
		verb, noun = "are", noun+"s"
	}
	return fmt.Sprintf("%-10s no — %d %s %s %s\n", label+":", n, noun, verb, condition)
}

// sortedRows copies the comparable rows and orders them by broader then
// narrower: the inventory carries them in the build's comprehension order,
// and the report is read by people diffing two runs.
func sortedRows(rows []libplatform.ComparablePredicates) []libplatform.ComparablePredicates {
	out := append([]libplatform.ComparablePredicates(nil), rows...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Broader != out[j].Broader {
			return out[i].Broader < out[j].Broader
		}
		return out[i].Narrower < out[j].Narrower
	})
	return out
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
