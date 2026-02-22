package loader

import (
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/output"
)

// LoadProvider loads a provider by name from the given providers map and returns
// a fully-parsed [*core.Provider] with all metadata fields, transformer definitions,
// and CueCtx set.
//
// If name is empty and the map contains exactly one provider, that provider is
// selected automatically. If name is empty and there are multiple providers, an
// error is returned.
func LoadProvider(cueCtx *cue.Context, name string, providers map[string]cue.Value) (*core.Provider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}

	// Auto-select when name is omitted and there is exactly one provider.
	if name == "" {
		if len(providers) == 1 {
			for k := range providers {
				name = k
			}
		} else {
			available := sortedProviderKeys(providers)
			return nil, fmt.Errorf("provider name must be specified (available: %v)", available)
		}
	}

	providerValue, ok := providers[name]
	if !ok {
		available := sortedProviderKeys(providers)
		return nil, fmt.Errorf("provider %q not found (available: %v)", name, available)
	}

	transformers, err := parseTransformers(name, providerValue)
	if err != nil {
		return nil, err
	}

	if len(transformers) == 0 {
		return nil, fmt.Errorf("provider %q has no transformer definitions", name)
	}

	transformerMap := make(map[string]*core.Transformer, len(transformers))
	for _, tf := range transformers {
		transformerMap[tf.Metadata.Name] = tf
	}

	p := &core.Provider{
		CueCtx:       cueCtx,
		Transformers: transformerMap,
	}
	extractProviderMetadata(providerValue, name, p)

	output.Debug("loaded provider",
		"name", p.Metadata.Name,
		"version", p.Metadata.Version,
		"transformers", len(transformerMap),
	)

	return p, nil
}

// extractProviderMetadata fills all fields of p.Metadata and p.ApiVersion/Kind
// from the provider CUE value. configKeyName is used as fallback for Metadata.Name
// when metadata.name is absent from the CUE value.
func extractProviderMetadata(v cue.Value, configKeyName string, p *core.Provider) { //nolint:gocyclo // linear field extraction; each branch is a distinct metadata field
	meta := &core.ProviderMetadata{Name: configKeyName}

	if f := v.LookupPath(cue.ParsePath("metadata.name")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.Name = str
		}
	}
	if f := v.LookupPath(cue.ParsePath("metadata.description")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.Description = str
		}
	}
	if f := v.LookupPath(cue.ParsePath("metadata.version")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.Version = str
		}
	}
	if f := v.LookupPath(cue.ParsePath("metadata.minVersion")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.MinVersion = str
		}
	}
	if labelsVal := v.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		labels := make(map[string]string)
		if iter, err := labelsVal.Fields(); err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					labels[iter.Selector().Unquoted()] = str
				}
			}
		}
		if len(labels) > 0 {
			meta.Labels = labels
		}
	}
	if f := v.LookupPath(cue.ParsePath("apiVersion")); f.Exists() {
		if str, err := f.String(); err == nil {
			p.ApiVersion = str
		}
	}
	if f := v.LookupPath(cue.ParsePath("kind")); f.Exists() {
		if str, err := f.String(); err == nil {
			p.Kind = str
		}
	}

	p.Metadata = meta
}

// parseTransformers iterates the transformers field of a provider CUE value
// and returns a slice of parsed [*core.Transformer] values.
func parseTransformers(providerName string, providerValue cue.Value) ([]*core.Transformer, error) {
	transformersValue := providerValue.LookupPath(cue.ParsePath("transformers"))
	if !transformersValue.Exists() {
		return nil, nil
	}

	iter, err := transformersValue.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating transformers for provider %q: %w", providerName, err)
	}

	var transformers []*core.Transformer
	for iter.Next() {
		tfName := iter.Selector().Unquoted()
		tfValue := iter.Value()
		fqn := buildFQN(providerName, tfName)

		tf, err := extractTransformer(tfName, fqn, tfValue)
		if err != nil {
			return nil, fmt.Errorf("parsing transformer %q: %w", fqn, err)
		}

		output.Debug("extracted transformer",
			"fqn", fqn,
			"requiredResources", tf.GetRequiredResources(),
			"requiredTraits", tf.GetRequiredTraits(),
		)

		transformers = append(transformers, tf)
	}

	return transformers, nil
}

// extractTransformer builds a [*core.Transformer] from a CUE transformer value.
func extractTransformer(name, fqn string, value cue.Value) (*core.Transformer, error) {
	if err := value.Err(); err != nil {
		return nil, fmt.Errorf("CUE evaluation error: %w", err)
	}

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

	if transformVal := value.LookupPath(cue.ParsePath("#transform")); transformVal.Exists() {
		tf.Transform = transformVal
	}

	return tf, nil
}

// buildFQN returns a fully qualified transformer name in the form
// "<provider>#<transformer>", e.g. "kubernetes#deployment".
func buildFQN(providerName, transformerName string) string {
	return providerName + "#" + transformerName
}

// sortedProviderKeys returns the keys of a providers map in sorted order.
func sortedProviderKeys(m map[string]cue.Value) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// extractLabelsField extracts a labels map field from a CUE value into dst.
func extractLabelsField(value cue.Value, field string, dst map[string]string) {
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
			dst[iter.Selector().Unquoted()] = str
		}
	}
}

// extractCueValueMap extracts a CUE map field as a map[string]cue.Value,
// preserving the full CUE value for each entry.
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
