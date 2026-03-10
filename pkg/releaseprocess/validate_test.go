package releaseprocess

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig_MergesValues(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`{
		name: string
		replicas: int
	}`)
	v1 := ctx.CompileString(`{name: "app"}`)
	v2 := ctx.CompileString(`{replicas: 2}`)

	merged, err := ValidateConfig(schema, []cue.Value{v1, v2}, "module", "demo")
	require.Nil(t, err)
	require.True(t, merged.Exists())
	assert.NoError(t, merged.Validate(cue.Concrete(true)))
	name, _ := merged.LookupPath(cue.ParsePath("name")).String()
	assert.Equal(t, "app", name)
}

func TestValidateConfig_ConflictingValues(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`{ replicas: int }`)
	v1 := ctx.CompileString(`{replicas: 1}`)
	v2 := ctx.CompileString(`{replicas: 2}`)

	_, err := ValidateConfig(schema, []cue.Value{v1, v2}, "module", "demo")
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "demo")
}
