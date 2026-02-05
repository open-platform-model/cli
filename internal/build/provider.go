package build

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
)

// ProviderLoader loads providers from configuration.
type ProviderLoader struct {
	config *config.OPMConfig
}

// NewProviderLoader creates a new ProviderLoader.
func NewProviderLoader(cfg *config.OPMConfig) *ProviderLoader {
	return &ProviderLoader{config: cfg}
}

// LoadedProvider is the result of loading a provider.
type LoadedProvider struct {
	Name         string
	Version      string
	Transformers []*LoadedTransformer

	// Index for fast lookup by FQN
	byFQN map[string]*LoadedTransformer
}

// LoadedTransformer is a transformer with extracted requirements.
type LoadedTransformer struct {
	Name              string
	FQN               string // Fully qualified name
	RequiredLabels    map[string]string
	RequiredResources []string
	RequiredTraits    []string
	OptionalLabels    map[string]string
	OptionalResources []string
	OptionalTraits    []string
	Value             cue.Value // Full transformer value for execution
}

// Load loads a provider by name from configuration.
func (pl *ProviderLoader) Load(ctx context.Context, name string) (*LoadedProvider, error) {
	if pl.config == nil || pl.config.Providers == nil {
		return nil, fmt.Errorf("no providers configured")
	}

	// Find provider by name
	providerValue, ok := pl.config.Providers[name]
	if !ok {
		available := make([]string, 0, len(pl.config.Providers))
		for k := range pl.config.Providers {
			available = append(available, k)
		}
		return nil, fmt.Errorf("provider %q not found (available: %v)", name, available)
	}

	output.Debug("loading provider", "name", name)

	provider := &LoadedProvider{
		Name:  name,
		byFQN: make(map[string]*LoadedTransformer),
	}

	// Extract version
	if versionVal := providerValue.LookupPath(cue.ParsePath("version")); versionVal.Exists() {
		if str, err := versionVal.String(); err == nil {
			provider.Version = str
		}
	}

	// Extract transformers
	transformersValue := providerValue.LookupPath(cue.ParsePath("transformers"))
	if !transformersValue.Exists() {
		output.Debug("no transformers found in provider", "name", name)
		return provider, nil
	}

	// Iterate over transformers
	iter, err := transformersValue.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating transformers: %w", err)
	}

	for iter.Next() {
		tfName := iter.Label()
		tfValue := iter.Value()

		transformer, err := pl.extractTransformer(name, tfName, tfValue)
		if err != nil {
			return nil, fmt.Errorf("extracting transformer %s: %w", tfName, err)
		}

		provider.Transformers = append(provider.Transformers, transformer)
		provider.byFQN[transformer.FQN] = transformer
	}

	output.Debug("loaded provider",
		"name", name,
		"version", provider.Version,
		"transformers", len(provider.Transformers),
	)

	return provider, nil
}

// extractTransformer extracts a transformer's metadata and requirements.
func (pl *ProviderLoader) extractTransformer(providerName, name string, value cue.Value) (*LoadedTransformer, error) {
	transformer := &LoadedTransformer{
		Name:              name,
		FQN:               buildFQN(providerName, name),
		RequiredLabels:    make(map[string]string),
		RequiredResources: make([]string, 0),
		RequiredTraits:    make([]string, 0),
		OptionalLabels:    make(map[string]string),
		OptionalResources: make([]string, 0),
		OptionalTraits:    make([]string, 0),
		Value:             value,
	}

	// Extract required labels
	if reqLabels := value.LookupPath(cue.ParsePath("requiredLabels")); reqLabels.Exists() {
		iter, err := reqLabels.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					transformer.RequiredLabels[iter.Label()] = str
				}
			}
		}
	}

	// Extract required resources
	if reqRes := value.LookupPath(cue.ParsePath("requiredResources")); reqRes.Exists() {
		iter, err := reqRes.List()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					transformer.RequiredResources = append(transformer.RequiredResources, str)
				}
			}
		}
	}

	// Extract required traits
	if reqTraits := value.LookupPath(cue.ParsePath("requiredTraits")); reqTraits.Exists() {
		iter, err := reqTraits.List()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					transformer.RequiredTraits = append(transformer.RequiredTraits, str)
				}
			}
		}
	}

	// Extract optional labels
	if optLabels := value.LookupPath(cue.ParsePath("optionalLabels")); optLabels.Exists() {
		iter, err := optLabels.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					transformer.OptionalLabels[iter.Label()] = str
				}
			}
		}
	}

	// Extract optional resources
	if optRes := value.LookupPath(cue.ParsePath("optionalResources")); optRes.Exists() {
		iter, err := optRes.List()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					transformer.OptionalResources = append(transformer.OptionalResources, str)
				}
			}
		}
	}

	// Extract optional traits
	if optTraits := value.LookupPath(cue.ParsePath("optionalTraits")); optTraits.Exists() {
		iter, err := optTraits.List()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					transformer.OptionalTraits = append(transformer.OptionalTraits, str)
				}
			}
		}
	}

	return transformer, nil
}

// GetByFQN returns a transformer by its fully qualified name.
func (p *LoadedProvider) GetByFQN(fqn string) (*LoadedTransformer, bool) {
	t, ok := p.byFQN[fqn]
	return t, ok
}

// ToSummaries converts transformers to summaries for error messages.
func (p *LoadedProvider) ToSummaries() []TransformerSummary {
	summaries := make([]TransformerSummary, len(p.Transformers))
	for i, t := range p.Transformers {
		summaries[i] = TransformerSummary{
			FQN:               t.FQN,
			RequiredLabels:    t.RequiredLabels,
			RequiredResources: t.RequiredResources,
			RequiredTraits:    t.RequiredTraits,
		}
	}
	return summaries
}

// buildFQN builds a fully qualified transformer name.
func buildFQN(providerName, transformerName string) string {
	return providerName + "#" + transformerName
}
