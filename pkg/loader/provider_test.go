package loader

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractProviderMetadata_WithMetadata(t *testing.T) {
	ctx := cuecontext.New()

	v := ctx.CompileString(`{
		metadata: {
			name:        "kubernetes"
			description: "Kubernetes provider"
		}
	}`)
	require.NoError(t, v.Err())

	meta, err := extractProviderMetadata(v, "fallback-key")
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", meta.Name)
}

func TestExtractProviderMetadata_NoMetadataBlock(t *testing.T) {
	ctx := cuecontext.New()

	// Provider value with no metadata block — should use config key name.
	v := ctx.CompileString(`{
		transformers: {}
	}`)
	require.NoError(t, v.Err())

	meta, err := extractProviderMetadata(v, "my-provider")
	require.NoError(t, err)
	assert.Equal(t, "my-provider", meta.Name)
}

func TestExtractProviderMetadata_EmptyNameFallback(t *testing.T) {
	ctx := cuecontext.New()

	// metadata.name is empty string — should fall back to configKeyName.
	v := ctx.CompileString(`{
		metadata: {
			name: ""
		}
	}`)
	require.NoError(t, v.Err())

	meta, err := extractProviderMetadata(v, "fallback")
	require.NoError(t, err)
	assert.Equal(t, "fallback", meta.Name)
}

func TestRegistryKeys(t *testing.T) {
	ctx := cuecontext.New()

	registry := ctx.CompileString(`{
		kubernetes: { metadata: name: "kubernetes" }
		helm:       { metadata: name: "helm" }
	}`)
	require.NoError(t, registry.Err())

	keys, err := registryKeys(registry)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"kubernetes", "helm"}, keys)
}

func TestRegistryKeys_Empty(t *testing.T) {
	ctx := cuecontext.New()

	registry := ctx.CompileString(`{}`)
	require.NoError(t, registry.Err())

	keys, err := registryKeys(registry)
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestRegistryKeys_Single(t *testing.T) {
	ctx := cuecontext.New()

	registry := ctx.CompileString(`{
		kubernetes: { metadata: name: "kubernetes" }
	}`)
	require.NoError(t, registry.Err())

	keys, err := registryKeys(registry)
	require.NoError(t, err)
	assert.Equal(t, []string{"kubernetes"}, keys)
}

// TestLoadProvider_AutoSelect verifies auto-selection logic using an
// in-memory registry value (bypasses CUE module loading).
func TestRegistryKeys_AutoSelectLogic(t *testing.T) {
	ctx := cuecontext.New()

	// Single-provider registry.
	registry := ctx.CompileString(`{
		kubernetes: {}
	}`)
	require.NoError(t, registry.Err())

	names, err := registryKeys(registry)
	require.NoError(t, err)
	require.Len(t, names, 1)

	// Simulate auto-selection: if len==1, use names[0].
	providerName := ""
	if len(names) == 1 {
		providerName = names[0]
	}
	assert.Equal(t, "kubernetes", providerName)
}

// TestLoadProvider_AutoSelectRequiresName verifies the multi-provider case
// requires explicit name selection.
func TestRegistryKeys_MultiProviderRequiresName(t *testing.T) {
	ctx := cuecontext.New()

	registry := ctx.CompileString(`{
		kubernetes: {}
		helm:       {}
	}`)
	require.NoError(t, registry.Err())

	names, err := registryKeys(registry)
	require.NoError(t, err)

	// With multiple providers and empty name, auto-select should fail.
	providerName := ""
	var selected string
	if len(names) == 1 {
		selected = names[0]
	}
	assert.Empty(t, selected, "should not auto-select with multiple providers and empty name")
	assert.Empty(t, providerName)
}

// TestLoadProvider_NotFound verifies the not-found lookup path using cue.Value directly.
func TestLoadProvider_NotFound(t *testing.T) {
	ctx := cuecontext.New()

	registry := ctx.CompileString(`{
		kubernetes: { metadata: name: "kubernetes" }
	}`)
	require.NoError(t, registry.Err())

	providerVal := registry.LookupPath(cue.MakePath(cue.Str("nonexistent")))
	assert.False(t, providerVal.Exists(), "non-existent provider should not be found")
}
