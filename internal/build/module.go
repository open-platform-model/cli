package build

import (
	"cuelang.org/go/cue"
)

// LoadedComponent is a component with extracted metadata.
// Components are extracted by ReleaseBuilder during the build phase.
type LoadedComponent struct {
	Name        string
	Labels      map[string]string    // Effective labels (merged from resources/traits)
	Annotations map[string]string    // Annotations from metadata.annotations
	Resources   map[string]cue.Value // FQN -> resource value
	Traits      map[string]cue.Value // FQN -> trait value
	Value       cue.Value            // Full component value
}
