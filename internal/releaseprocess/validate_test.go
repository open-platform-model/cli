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

func TestValidateConfig_CollectsSchemaErrorsAndMergeConflicts(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`close({
		timezone: string
		media?: [Name=string]: {
			mountPath: string
			type:      "pvc" | *"emptyDir"
			size:      string
		}
	})`)

	v1 := ctx.CompileString(`{
		timezone: "Europe/Stockholm"
		test: "test"
		media: {
			test: "test"
		}
		invalidField: "bad"
	}`, cue.Filename("values.cue"))
	v2 := ctx.CompileString(`{
		timezone: "Etc/UTC"
	}`, cue.Filename("values2.cue"))

	merged, err := ValidateConfig(schema, []cue.Value{v1, v2}, "module", "demo")
	require.NotNil(t, err)
	assert.False(t, merged.Exists())

	groups := err.GroupedErrors()
	require.Len(t, groups, 4)

	assert.Equal(t, "field not allowed", groups[0].Message)
	assert.Equal(t, "values.test", groups[0].Locations[0].Path)
	assert.Equal(t, "field not allowed", groups[1].Message)
	assert.Equal(t, "values.invalidField", groups[1].Locations[0].Path)
	assert.Contains(t, groups[2].Message, "conflicting values \"test\"")
	assert.Equal(t, "values.media.test", groups[2].Locations[0].Path)
	assert.Contains(t, groups[3].Message, "conflicting values \"Etc/UTC\" and \"Europe/Stockholm\"")
	assert.Equal(t, "values.timezone", groups[3].Locations[0].Path)
	files := make([]string, 0, len(groups[3].Locations))
	for _, loc := range groups[3].Locations {
		files = append(files, loc.File)
	}
	assert.Contains(t, files, "values.cue")
	assert.Contains(t, files, "values2.cue")
}
