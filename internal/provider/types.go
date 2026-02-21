package provider

import (
	"github.com/opmodel/cli/internal/core"
)

// LoadedProvider holds the result of loading a provider from GlobalConfig.
// It contains the provider name and all parsed transformer definitions,
// ready for use in component matching.
type LoadedProvider struct {
	// Name is the provider name as configured in GlobalConfig.
	Name string

	// Transformers is the list of transformer definitions parsed from the
	// provider's CUE value. Each entry is ready for use by core.Provider.Match().
	Transformers []*core.Transformer
}

// Requirements returns the FQN of each transformer, for use in error messages
// (e.g., "no components matched; available transformers: kubernetes#deployment").
func (p *LoadedProvider) Requirements() []string {
	fqns := make([]string, 0, len(p.Transformers))
	for _, t := range p.Transformers {
		fqns = append(fqns, t.GetFQN())
	}
	return fqns
}
