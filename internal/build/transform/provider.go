package transform

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

	// byFQN is an index for fast lookup.
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

	providerValue, ok := pl.config.Providers[name]
	if !ok {
		available := make([]string, 0, len(pl.config.Providers))
		for k := range pl.config.Providers {
			available = append(available, k)
		}
		return nil, fmt.Errorf("provider %q not found (available: %v)", name, available)
	}

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

	iter, err := transformersValue.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating transformers: %w", err)
	}

	for iter.Next() {
		tfName := iter.Selector().Unquoted()
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
//
//nolint:unparam // error return allows for future validation
func (pl *ProviderLoader) extractTransformer(providerName, name string, value cue.Value) (*LoadedTransformer, error) {
	transformer := &LoadedTransformer{
		Name:              name,
		FQN:               BuildFQN(providerName, name),
		RequiredLabels:    make(map[string]string),
		RequiredResources: make([]string, 0),
		RequiredTraits:    make([]string, 0),
		OptionalLabels:    make(map[string]string),
		OptionalResources: make([]string, 0),
		OptionalTraits:    make([]string, 0),
		Value:             value,
	}

	pl.extractLabelsField(value, "requiredLabels", transformer.RequiredLabels)
	transformer.RequiredResources = pl.extractMapKeys(value, "requiredResources")
	transformer.RequiredTraits = pl.extractMapKeys(value, "requiredTraits")

	pl.extractLabelsField(value, "optionalLabels", transformer.OptionalLabels)
	transformer.OptionalResources = pl.extractMapKeys(value, "optionalResources")
	transformer.OptionalTraits = pl.extractMapKeys(value, "optionalTraits")

	output.Debug("extracted transformer",
		"name", transformer.FQN,
		"requiredResources", transformer.RequiredResources,
		"requiredTraits", transformer.RequiredTraits,
	)

	return transformer, nil
}

// extractLabelsField extracts a labels map field from a CUE value.
func (pl *ProviderLoader) extractLabelsField(value cue.Value, field string, labels map[string]string) {
	fieldVal := value.LookupPath(cue.ParsePath(field))
	if !fieldVal.Exists() {
		return
	}
	iter, err := fieldVal.Fields()
	if err != nil {
		return
	}
	for iter.Next() {
		if str, err := iter.Value().String(); err == nil {
			labels[iter.Selector().Unquoted()] = str
		}
	}
}

// extractMapKeys extracts the keys from a map field as a string slice.
func (pl *ProviderLoader) extractMapKeys(value cue.Value, field string) []string {
	result := make([]string, 0)
	fieldVal := value.LookupPath(cue.ParsePath(field))
	if !fieldVal.Exists() {
		return result
	}

	iter, err := fieldVal.Fields()
	if err != nil {
		return result
	}
	for iter.Next() {
		key := iter.Selector().Unquoted()
		result = append(result, key)
	}
	return result
}

// Requirements returns the provider's transformers as a []TransformerRequirements slice.
// This replaces ToSummaries — no data is copied; LoadedTransformer satisfies the interface directly.
func (p *LoadedProvider) Requirements() []TransformerRequirements {
	result := make([]TransformerRequirements, len(p.Transformers))
	for i, t := range p.Transformers {
		result[i] = t
	}
	return result
}

// GetFQN returns the transformer's fully qualified name.
func (t *LoadedTransformer) GetFQN() string { return t.FQN }

// GetRequiredLabels returns the transformer's required labels.
func (t *LoadedTransformer) GetRequiredLabels() map[string]string { return t.RequiredLabels }

// GetRequiredResources returns the transformer's required resources.
func (t *LoadedTransformer) GetRequiredResources() []string { return t.RequiredResources }

// GetRequiredTraits returns the transformer's required traits.
func (t *LoadedTransformer) GetRequiredTraits() []string { return t.RequiredTraits }

// BuildFQN builds a fully qualified transformer name.
func BuildFQN(providerName, transformerName string) string {
	return providerName + "#" + transformerName
}
