package loader_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/loader"
)

// buildProviderMap creates a map of provider CUE values from CUE source strings.
func buildProviderMap(ctx *cue.Context, sources map[string]string) map[string]cue.Value {
	result := make(map[string]cue.Value, len(sources))
	for name, src := range sources {
		result[name] = ctx.CompileString(src)
	}
	return result
}

const singleTransformerCUE = `
transformers: {
	deployment: {
		requiredResources: {
			"apps/Deployment": {}
		}
		requiredTraits: {
			"core/replicas": {}
		}
		optionalLabels: {
			env: "production"
		}
		optionalResources: {
			"v1/Service": {}
		}
		optionalTraits: {
			"core/autoscaling": {}
		}
	}
}
`

const noOptionalCUE = `
transformers: {
	service: {
		requiredResources: {
			"v1/Service": {}
		}
	}
}
`

// 3.1 — Named provider found returns *coreprovider.Provider with correct transformer count.
func TestLoadProvider_NamedProviderFound(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	p, err := loader.LoadProvider(ctx, "kubernetes", providers)
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", p.Metadata.Name)
	assert.Len(t, p.Transformers, 1)
}

// 3.2 — Provider name not found returns error listing available providers.
func TestLoadProvider_ProviderNotFound(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	_, err := loader.LoadProvider(ctx, "aws", providers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"aws" not found`)
	assert.Contains(t, err.Error(), "kubernetes")
}

// 3.3 — Auto-select when exactly one provider configured.
func TestLoadProvider_AutoSelect_SingleProvider(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	p, err := loader.LoadProvider(ctx, "", providers)
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", p.Metadata.Name)
}

// 3.4 — Empty name with multiple providers returns error.
func TestLoadProvider_EmptyName_MultipleProviders(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
		"aws":        singleTransformerCUE,
	})

	_, err := loader.LoadProvider(ctx, "", providers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider name must be specified")
}

// 3.5 — Transformer FQN construction: kubernetes#deployment.
func TestLoadProvider_TransformerFQN(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	p, err := loader.LoadProvider(ctx, "kubernetes", providers)
	require.NoError(t, err)
	require.Contains(t, p.Transformers, "deployment")
	assert.Equal(t, "kubernetes#deployment", p.Transformers["deployment"].GetFQN())
}

// 3.6 — Transformer with required and optional criteria populates all fields on *transformer.Transformer.
func TestLoadProvider_TransformerCriteria_AllFields(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	p, err := loader.LoadProvider(ctx, "kubernetes", providers)
	require.NoError(t, err)
	require.Contains(t, p.Transformers, "deployment")

	tf := p.Transformers["deployment"]
	assert.Equal(t, []string{"apps/Deployment"}, tf.GetRequiredResources())
	assert.Equal(t, []string{"core/replicas"}, tf.GetRequiredTraits())
	assert.Equal(t, map[string]string{"env": "production"}, tf.OptionalLabels)
	assert.Contains(t, tf.OptionalResources, "v1/Service")
	assert.Contains(t, tf.OptionalTraits, "core/autoscaling")
}

// 3.7 — Transformer with no optional criteria returns empty slices/maps, no error.
func TestLoadProvider_TransformerNoOptionalCriteria(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": noOptionalCUE,
	})

	p, err := loader.LoadProvider(ctx, "kubernetes", providers)
	require.NoError(t, err)
	require.Contains(t, p.Transformers, "service")

	tf := p.Transformers["service"]
	assert.Empty(t, tf.OptionalLabels)
	assert.Empty(t, tf.OptionalResources)
	assert.Empty(t, tf.OptionalTraits)
}

// 3.8 — Provider with no transformers returns error.
func TestLoadProvider_NoTransformers(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": `version: "1.0.0"`,
	})

	_, err := loader.LoadProvider(ctx, "kubernetes", providers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no transformer definitions")
}

// Invalid transformer definition — error identifies which transformer failed.
func TestLoadProvider_InvalidTransformerDefinition(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": `transformers: { broken: _|_ }`,
	})

	_, err := loader.LoadProvider(ctx, "kubernetes", providers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kubernetes#broken")
}

// 3.9 — Requirements() returns FQN slice matching loaded transformers.
func TestLoadProvider_Requirements(t *testing.T) {
	ctx := cuecontext.New()
	multiTfCUE := `
transformers: {
	deployment: { requiredResources: { "apps/Deployment": {} } }
	service:    { requiredResources: { "v1/Service": {} } }
	configmap:  { requiredResources: { "v1/ConfigMap": {} } }
}
`
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": multiTfCUE,
	})

	p, err := loader.LoadProvider(ctx, "kubernetes", providers)
	require.NoError(t, err)
	assert.Len(t, p.Transformers, 3)

	reqs := p.Requirements()
	assert.Len(t, reqs, 3)
	fqns := make([]string, 0, len(reqs))
	for _, r := range reqs {
		fqns = append(fqns, r.GetFQN())
	}
	assert.Contains(t, fqns, "kubernetes#deployment")
	assert.Contains(t, fqns, "kubernetes#service")
	assert.Contains(t, fqns, "kubernetes#configmap")
}

// CueCtx is set on returned *coreprovider.Provider.
func TestLoadProvider_CueCtxSet(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	p, err := loader.LoadProvider(ctx, "kubernetes", providers)
	require.NoError(t, err)
	assert.NotNil(t, p.CueCtx)
}

// 2.3 — Metadata fields are populated when present in CUE value.
func TestLoadProvider_MetadataExtraction(t *testing.T) {
	ctx := cuecontext.New()
	withMetaCUE := `
apiVersion: "opmodel.dev/providers/kubernetes@v0"
kind: "Provider"
metadata: {
	name:        "kubernetes-provider"
	description: "Kubernetes transformer provider"
	version:     "1.2.3"
	minVersion:  "0.5.0"
	labels: {
		env: "production"
		tier: "infra"
	}
}
transformers: {
	deployment: { requiredResources: { "apps/Deployment": {} } }
}
`
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": withMetaCUE,
	})

	p, err := loader.LoadProvider(ctx, "kubernetes", providers)
	require.NoError(t, err)
	assert.Equal(t, "kubernetes-provider", p.Metadata.Name)
	assert.Equal(t, "Kubernetes transformer provider", p.Metadata.Description)
	assert.Equal(t, "1.2.3", p.Metadata.Version)
	assert.Equal(t, "0.5.0", p.Metadata.MinVersion)
	assert.Equal(t, map[string]string{"env": "production", "tier": "infra"}, p.Metadata.Labels)
	assert.Equal(t, "opmodel.dev/providers/kubernetes@v0", p.APIVersion)
	assert.Equal(t, "Provider", p.Kind)
}

// 2.4 — Config key used as Metadata.Name fallback when metadata.name absent.
func TestLoadProvider_MetadataName_FallbackToConfigKey(t *testing.T) {
	ctx := cuecontext.New()
	providers := buildProviderMap(ctx, map[string]string{
		"kubernetes": singleTransformerCUE, // no metadata block
	})

	p, err := loader.LoadProvider(ctx, "kubernetes", providers)
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", p.Metadata.Name)
}
