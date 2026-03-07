// Package loader provides functions to load providers and module releases
// from CUE module directories. These are the entry points for embedding
// the OPM factory engine in other tools.
package loader

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/experiments/factory/pkg/provider"
)

// LoadProvider loads a named provider from the factory's provider registry.
//
// cueModuleDir must be the absolute path to the factory v1alpha1 CUE module root
// (the directory containing cue.mod/). This is the same directory that is passed
// to engine.NewModuleRenderer.
//
// providerName selects the provider from the #Registry map (e.g. "kubernetes").
// If providerName is empty and the registry contains exactly one provider, that
// provider is selected automatically.
func LoadProvider(cueCtx *cue.Context, cueModuleDir string, providerName string) (*provider.Provider, error) {
	// Load the providers package, which defines #Registry.
	cfg := &load.Config{Dir: cueModuleDir}
	instances := load.Instances([]string{"./providers"}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("providers package not found in %s", cueModuleDir)
	}
	if instances[0].Err != nil {
		return nil, fmt.Errorf("loading providers package: %w", instances[0].Err)
	}

	providersPkg := cueCtx.BuildInstance(instances[0])
	if err := providersPkg.Err(); err != nil {
		return nil, fmt.Errorf("building providers package: %w", err)
	}

	// Look up the #Registry definition.
	registry := providersPkg.LookupPath(cue.MakePath(cue.Def("Registry")))
	if err := registry.Err(); err != nil {
		return nil, fmt.Errorf("looking up #Registry: %w", err)
	}

	// Auto-select when providerName is empty and there is exactly one entry.
	if providerName == "" {
		names, err := registryKeys(registry)
		if err != nil {
			return nil, err
		}
		if len(names) == 1 {
			providerName = names[0]
		} else {
			return nil, fmt.Errorf("provider name required (available: %v)", names)
		}
	}

	// Look up the specific provider entry.
	providerVal := registry.LookupPath(cue.MakePath(cue.Str(providerName)))
	if !providerVal.Exists() {
		names, _ := registryKeys(registry)
		return nil, fmt.Errorf("provider %q not found in #Registry (available: %v)", providerName, names)
	}
	if err := providerVal.Err(); err != nil {
		return nil, fmt.Errorf("evaluating provider %q: %w", providerName, err)
	}

	// Extract display metadata — we read only the minimal fields needed for logging
	// and provider selection. All transformer and schema data stays in Raw.
	meta, err := extractProviderMetadata(providerVal, providerName)
	if err != nil {
		return nil, fmt.Errorf("extracting provider metadata: %w", err)
	}

	return &provider.Provider{
		Metadata: meta,
		Data:     providerVal,
	}, nil
}

// extractProviderMetadata decodes the provider metadata struct directly using
// Decode(), falling back to configKeyName for metadata.name when the field is absent.
func extractProviderMetadata(v cue.Value, configKeyName string) (*provider.ProviderMetadata, error) {
	meta := &provider.ProviderMetadata{Name: configKeyName}

	metaVal := v.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		// Provider has no metadata block — use config key name as the name.
		return meta, nil
	}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding provider metadata: %w", err)
	}
	// Preserve the fallback: if CUE metadata.name decoded as empty, use configKeyName.
	if meta.Name == "" {
		meta.Name = configKeyName
	}
	return meta, nil
}

// registryKeys returns the sorted keys of a #Registry struct.
func registryKeys(registry cue.Value) ([]string, error) {
	iter, err := registry.Fields()
	if err != nil {
		return nil, fmt.Errorf("reading #Registry fields: %w", err)
	}
	var names []string
	for iter.Next() {
		names = append(names, iter.Selector().Unquoted())
	}
	return names, nil
}
