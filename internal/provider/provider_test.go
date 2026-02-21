package provider_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/provider"
)

// newCtx returns a fresh CUE context for each test.
func newCtx() *cue.Context {
	return cuecontext.New()
}

// buildProviders creates a map of provider CUE values from CUE source strings.
func buildProviders(ctx *cue.Context, sources map[string]string) map[string]cue.Value {
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

// 3.1 — Named provider found returns LoadedProvider with correct transformer count.
func TestLoad_NamedProviderFound(t *testing.T) {
	ctx := newCtx()
	providers := buildProviders(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	lp, err := provider.Load(ctx, "kubernetes", providers)
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", lp.Name)
	assert.Len(t, lp.Transformers, 1)
}

// 3.2 — Provider name not found returns error listing available providers.
func TestLoad_ProviderNotFound(t *testing.T) {
	ctx := newCtx()
	providers := buildProviders(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	_, err := provider.Load(ctx, "aws", providers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"aws" not found`)
	assert.Contains(t, err.Error(), "kubernetes")
}

// 3.3 — Auto-select when exactly one provider configured.
func TestLoad_AutoSelect_SingleProvider(t *testing.T) {
	ctx := newCtx()
	providers := buildProviders(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	lp, err := provider.Load(ctx, "", providers)
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", lp.Name)
}

// 3.4 — Empty name with multiple providers returns error.
func TestLoad_EmptyName_MultipleProviders(t *testing.T) {
	ctx := newCtx()
	providers := buildProviders(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
		"aws":        singleTransformerCUE,
	})

	_, err := provider.Load(ctx, "", providers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider name must be specified")
}

// 3.5 — Transformer FQN construction: kubernetes#deployment.
func TestLoad_TransformerFQN(t *testing.T) {
	ctx := newCtx()
	providers := buildProviders(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	lp, err := provider.Load(ctx, "kubernetes", providers)
	require.NoError(t, err)
	require.Len(t, lp.Transformers, 1)
	assert.Equal(t, "kubernetes#deployment", lp.Transformers[0].GetFQN())
}

// 3.6 — Transformer with required and optional criteria populates all fields on *core.Transformer.
func TestLoad_TransformerCriteria_AllFields(t *testing.T) {
	ctx := newCtx()
	providers := buildProviders(ctx, map[string]string{
		"kubernetes": singleTransformerCUE,
	})

	lp, err := provider.Load(ctx, "kubernetes", providers)
	require.NoError(t, err)
	require.Len(t, lp.Transformers, 1)

	tf := lp.Transformers[0]
	assert.Equal(t, []string{"apps/Deployment"}, tf.GetRequiredResources())
	assert.Equal(t, []string{"core/replicas"}, tf.GetRequiredTraits())
	assert.Equal(t, map[string]string{"env": "production"}, tf.OptionalLabels)
	assert.Contains(t, tf.OptionalResources, "v1/Service")
	assert.Contains(t, tf.OptionalTraits, "core/autoscaling")
}

// 3.7 — Transformer with no optional criteria returns empty slices/maps, no error.
func TestLoad_TransformerNoOptionalCriteria(t *testing.T) {
	ctx := newCtx()
	providers := buildProviders(ctx, map[string]string{
		"kubernetes": noOptionalCUE,
	})

	lp, err := provider.Load(ctx, "kubernetes", providers)
	require.NoError(t, err)
	require.Len(t, lp.Transformers, 1)

	tf := lp.Transformers[0]
	assert.Empty(t, tf.OptionalLabels)
	assert.Empty(t, tf.OptionalResources)
	assert.Empty(t, tf.OptionalTraits)
}

// 3.8 — Provider with no transformers returns error.
func TestLoad_NoTransformers(t *testing.T) {
	ctx := newCtx()
	providers := buildProviders(ctx, map[string]string{
		"kubernetes": `version: "1.0.0"`,
	})

	_, err := provider.Load(ctx, "kubernetes", providers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no transformer definitions")
}

// Spec scenario: Invalid transformer definition — error identifies which transformer failed.
func TestLoad_InvalidTransformerDefinition(t *testing.T) {
	ctx := newCtx()
	// _|_ is a CUE bottom value — forces an evaluation error on the transformer.
	providers := buildProviders(ctx, map[string]string{
		"kubernetes": `transformers: { broken: _|_ }`,
	})

	_, err := provider.Load(ctx, "kubernetes", providers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kubernetes#broken")
}

// 3.9 — Requirements() returns FQN slice matching loaded transformers.
func TestLoad_Requirements(t *testing.T) {
	ctx := newCtx()
	multiTfCUE := `
transformers: {
	deployment: { requiredResources: { "apps/Deployment": {} } }
	service:    { requiredResources: { "v1/Service": {} } }
	configmap:  { requiredResources: { "v1/ConfigMap": {} } }
}
`
	providers := buildProviders(ctx, map[string]string{
		"kubernetes": multiTfCUE,
	})

	lp, err := provider.Load(ctx, "kubernetes", providers)
	require.NoError(t, err)
	assert.Len(t, lp.Transformers, 3)

	reqs := lp.Requirements()
	assert.Len(t, reqs, 3)
	assert.Contains(t, reqs, "kubernetes#deployment")
	assert.Contains(t, reqs, "kubernetes#service")
	assert.Contains(t, reqs, "kubernetes#configmap")
}
