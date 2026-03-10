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

	v := ctx.CompileString(`{ transformers: {} }`)
	require.NoError(t, v.Err())

	meta, err := extractProviderMetadata(v, "my-provider")
	require.NoError(t, err)
	assert.Equal(t, "my-provider", meta.Name)
}

func TestExtractProviderMetadata_EmptyNameFallback(t *testing.T) {
	ctx := cuecontext.New()

	v := ctx.CompileString(`{ metadata: { name: "" } }`)
	require.NoError(t, v.Err())

	meta, err := extractProviderMetadata(v, "fallback")
	require.NoError(t, err)
	assert.Equal(t, "fallback", meta.Name)
}

func TestLoadProvider_ExplicitName(t *testing.T) {
	ctx := cuecontext.New()

	k8s := ctx.CompileString(`{ metadata: { name: "kubernetes" } }`)
	require.NoError(t, k8s.Err())

	providers := map[string]cue.Value{"kubernetes": k8s}

	prov, err := LoadProvider("kubernetes", providers)
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", prov.Metadata.Name)
	assert.True(t, prov.Data.Exists())
}

func TestLoadProvider_DefaultsToKubernetes(t *testing.T) {
	ctx := cuecontext.New()

	k8s := ctx.CompileString(`{ metadata: { name: "kubernetes" } }`)
	require.NoError(t, k8s.Err())

	providers := map[string]cue.Value{"kubernetes": k8s}

	// Empty name → defaults to "kubernetes"
	prov, err := LoadProvider("", providers)
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", prov.Metadata.Name)
}

func TestLoadProvider_MultipleProviders_ExplicitRequired(t *testing.T) {
	ctx := cuecontext.New()

	k8s := ctx.CompileString(`{ metadata: { name: "kubernetes" } }`)
	helm := ctx.CompileString(`{ metadata: { name: "helm" } }`)
	require.NoError(t, k8s.Err())
	require.NoError(t, helm.Err())

	providers := map[string]cue.Value{"kubernetes": k8s, "helm": helm}

	// Explicit name works fine with multiple providers
	prov, err := LoadProvider("helm", providers)
	require.NoError(t, err)
	assert.Equal(t, "helm", prov.Metadata.Name)
}

func TestLoadProvider_NotFound(t *testing.T) {
	ctx := cuecontext.New()

	k8s := ctx.CompileString(`{ metadata: { name: "kubernetes" } }`)
	require.NoError(t, k8s.Err())

	providers := map[string]cue.Value{"kubernetes": k8s}

	_, err := LoadProvider("nonexistent", providers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
	assert.Contains(t, err.Error(), "available")
}

func TestLoadProvider_EmptyProviders(t *testing.T) {
	_, err := LoadProvider("kubernetes", map[string]cue.Value{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no providers configured")
}
