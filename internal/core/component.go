package core

import "cuelang.org/go/cue"

// Component is a component with extracted metadata.
// Components are extracted by the release builder during the build phase.
type Component struct {
	Name        string
	Labels      map[string]string    // Effective labels (merged from resources/traits)
	Annotations map[string]string    // Annotations from metadata.annotations
	Resources   map[string]cue.Value // FQN -> resource value
	Traits      map[string]cue.Value // FQN -> trait value
	Value       cue.Value            // Full component value
}
