package engine

import (
	"context"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/pkg/provider"
	"github.com/opmodel/cli/pkg/render"
)

func TestModuleRenderer_RenderReturnsNonNilEmptySlices(t *testing.T) {
	ctx := cuecontext.New()
	providerVal := ctx.CompileString(`{#transformers:{}}`)
	raw := ctx.CompileString(`{components:{}}`)
	data := ctx.CompileString(`{}`)

	renderer := NewModuleRenderer(&provider.Provider{Data: providerVal})
	result, err := renderer.Render(context.Background(), &render.ModuleRelease{
		Metadata:       &render.ModuleReleaseMetadata{Name: "demo"},
		RawCUE:         raw,
		DataComponents: data,
	}, &render.MatchPlan{Matches: map[string]map[string]render.MatchResult{}, UnhandledTraits: map[string][]string{}})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Resources)
	assert.NotNil(t, result.Components)
	assert.NotNil(t, result.Warnings)
	assert.Empty(t, result.Resources)
	assert.Empty(t, result.Components)
	assert.Empty(t, result.Warnings)
}

func TestBundleRenderer_RenderReturnsNonNilEmptySlices(t *testing.T) {
	ctx := cuecontext.New()
	providerVal := ctx.CompileString(`{#transformers:{}}`)
	renderer := NewBundleRenderer(&provider.Provider{Data: providerVal})
	result, err := renderer.Render(context.Background(), &render.BundleRelease{Releases: map[string]*render.ModuleRelease{}})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Resources)
	assert.NotNil(t, result.Warnings)
	assert.NotNil(t, result.ReleaseOrder)
	assert.Empty(t, result.Resources)
	assert.Empty(t, result.Warnings)
	assert.Empty(t, result.ReleaseOrder)
}
