package transform

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/output"
)

// LoadedTransformer is a transformer with extracted requirements, used by the Executor.
// It is retained alongside core.Transformer until core-transformer-match-plan-execute
// migrates the executor to operate on core types directly.
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

// LoadProvider loads a provider by name from the given providers map and returns:
//   - a *core.Provider with Transformers populated (for use by Provider.Match())
//   - a []*LoadedTransformer slice (for use by the Executor until the next migration step)
//
// No *cue.Context parameter is needed — all operations are field extractions on
// existing cue.Values (category-A ops that carry their runtime internally).
func LoadProvider(providers map[string]cue.Value, name string) (*core.Provider, []*LoadedTransformer, error) {
	if providers == nil {
		return nil, nil, fmt.Errorf("no providers configured")
	}

	providerValue, ok := providers[name]
	if !ok {
		available := make([]string, 0, len(providers))
		for k := range providers {
			available = append(available, k)
		}
		return nil, nil, fmt.Errorf("provider %q not found (available: %v)", name, available)
	}

	provider := &core.Provider{
		Metadata:     &core.ProviderMetadata{Name: name},
		Transformers: make(map[string]*core.Transformer),
	}
	var loadedTfs []*LoadedTransformer

	// Extract version.
	if versionVal := providerValue.LookupPath(cue.ParsePath("version")); versionVal.Exists() {
		if str, err := versionVal.String(); err == nil {
			provider.Metadata.Version = str
		}
	}

	// Extract transformers.
	transformersValue := providerValue.LookupPath(cue.ParsePath("transformers"))
	if !transformersValue.Exists() {
		output.Debug("no transformers found in provider", "name", name)
		return provider, loadedTfs, nil
	}

	iter, err := transformersValue.Fields()
	if err != nil {
		return nil, nil, fmt.Errorf("iterating transformers: %w", err)
	}

	for iter.Next() {
		tfName := iter.Selector().Unquoted()
		tfValue := iter.Value()
		fqn := BuildFQN(name, tfName)

		coreTf := extractCoreTransformer(name, tfName, fqn, tfValue)
		provider.Transformers[tfName] = coreTf

		loadedTf := extractLoadedTransformer(name, tfName, fqn, tfValue)
		loadedTfs = append(loadedTfs, loadedTf)

		output.Debug("extracted transformer",
			"name", fqn,
			"requiredResources", loadedTf.RequiredResources,
			"requiredTraits", loadedTf.RequiredTraits,
		)
	}

	output.Debug("loaded provider",
		"name", name,
		"version", provider.Metadata.Version,
		"transformers", len(provider.Transformers),
	)

	return provider, loadedTfs, nil
}

// extractCoreTransformer builds a *core.Transformer from a CUE transformer value.
func extractCoreTransformer(providerName, name, fqn string, value cue.Value) *core.Transformer {
	tf := &core.Transformer{
		Metadata: &core.TransformerMetadata{
			Name: name,
			FQN:  fqn,
		},
		RequiredLabels:    make(map[string]string),
		RequiredResources: extractCueValueMap(value, "requiredResources"),
		RequiredTraits:    extractCueValueMap(value, "requiredTraits"),
		OptionalLabels:    make(map[string]string),
		OptionalResources: extractCueValueMap(value, "optionalResources"),
		OptionalTraits:    extractCueValueMap(value, "optionalTraits"),
	}

	extractLabelsField(value, "requiredLabels", tf.RequiredLabels)
	extractLabelsField(value, "optionalLabels", tf.OptionalLabels)

	// Extract the #transform definition value if present.
	if transformVal := value.LookupPath(cue.ParsePath("#transform")); transformVal.Exists() {
		tf.Transform = transformVal
	}

	return tf
}

// extractLoadedTransformer builds a *LoadedTransformer from a CUE transformer value.
// The LoadedTransformer is kept for the Executor until core-transformer-match-plan-execute.
func extractLoadedTransformer(providerName, name, fqn string, value cue.Value) *LoadedTransformer {
	tf := &LoadedTransformer{
		Name:              name,
		FQN:               fqn,
		RequiredLabels:    make(map[string]string),
		RequiredResources: make([]string, 0),
		RequiredTraits:    make([]string, 0),
		OptionalLabels:    make(map[string]string),
		OptionalResources: make([]string, 0),
		OptionalTraits:    make([]string, 0),
		Value:             value,
	}

	extractLabelsField(value, "requiredLabels", tf.RequiredLabels)
	tf.RequiredResources = extractMapKeys(value, "requiredResources")
	tf.RequiredTraits = extractMapKeys(value, "requiredTraits")

	extractLabelsField(value, "optionalLabels", tf.OptionalLabels)
	tf.OptionalResources = extractMapKeys(value, "optionalResources")
	tf.OptionalTraits = extractMapKeys(value, "optionalTraits")

	return tf
}

// extractLabelsField extracts a labels map field from a CUE value into the given map.
func extractLabelsField(value cue.Value, field string, labels map[string]string) {
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

// extractMapKeys extracts the keys from a CUE map field as a string slice.
func extractMapKeys(value cue.Value, field string) []string {
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
		result = append(result, iter.Selector().Unquoted())
	}
	return result
}

// extractCueValueMap extracts a CUE map field as a map[string]cue.Value,
// preserving the full CUE value for each entry (used by core.Transformer fields).
func extractCueValueMap(value cue.Value, field string) map[string]cue.Value {
	result := make(map[string]cue.Value)
	fieldVal := value.LookupPath(cue.ParsePath(field))
	if !fieldVal.Exists() {
		return result
	}
	iter, err := fieldVal.Fields()
	if err != nil {
		return result
	}
	for iter.Next() {
		result[iter.Selector().Unquoted()] = iter.Value()
	}
	return result
}
