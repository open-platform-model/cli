package render_test

import (
	"context"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/pkg/core"
	"github.com/open-platform-model/cli/pkg/module"
	"github.com/open-platform-model/cli/pkg/provider"
	"github.com/open-platform-model/cli/pkg/render"
)

func TestModuleRenderer_RenderReturnsNonNilEmptySlices(t *testing.T) {
	ctx := cuecontext.New()
	providerVal := ctx.CompileString(`{#transformers:{}}`)
	raw := ctx.CompileString(`{components:{}}`)
	data := ctx.CompileString(`{}`)

	renderer := render.NewModule(&provider.Provider{Data: providerVal}, core.LabelManagedByValue)
	rel := &module.Instance{
		Metadata: &module.InstanceMetadata{Name: "demo"},
		Spec:     raw,
	}
	schemaComponents := rel.MatchComponents()
	result, err := renderer.Execute(context.Background(), rel, schemaComponents, data, &render.MatchPlan{Matches: map[string]map[string]render.MatchResult{}, UnhandledTraits: map[string][]string{}})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Resources)
	assert.NotNil(t, result.Components)
	assert.NotNil(t, result.Warnings)
	assert.Empty(t, result.Resources)
	assert.Empty(t, result.Components)
	assert.Empty(t, result.Warnings)
}
