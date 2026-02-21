package release

import "cuelang.org/go/cue"

// collectAllCUEErrors runs Validate() on the full value tree.
// Returns a combined error containing all discovered errors, or nil if clean.
// Used by Builder.Build() as a structural loading guard (step 4c and step 5).
func collectAllCUEErrors(v cue.Value) error {
	return v.Validate()
}
