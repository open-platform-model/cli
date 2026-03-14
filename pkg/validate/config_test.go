package validate

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/opmodel/cli/pkg/errors"
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

func TestValidateConfig_Valid(t *testing.T) {
	ctx := cuecontext.New()

	// schema: #config requires a string name and int replicas
	schema := ctx.CompileString(`{
		name:     string
		replicas: int & >=1
	}`)
	require.NoError(t, schema.Err())

	values := ctx.CompileString(`{
		name:     "my-app"
		replicas: 3
	}`)
	require.NoError(t, values.Err())

	_, err := ValidateConfig(schema, []cue.Value{values}, "module", "test-release")
	assert.Nil(t, err, "valid values should produce no error")
}

func TestValidateConfig_TypeMismatch(t *testing.T) {
	ctx := cuecontext.New()

	schema := ctx.CompileString(`{
		replicas: int
	}`)
	require.NoError(t, schema.Err())

	// string where int is expected
	values := ctx.CompileString(`{
		replicas: "not-a-number"
	}`)
	require.NoError(t, values.Err())

	_, cfgErr := ValidateConfig(schema, []cue.Value{values}, "module", "my-release")
	require.NotNil(t, cfgErr, "type mismatch should produce ConfigError")
	assert.IsType(t, &oerrors.ConfigError{}, cfgErr)
	assert.Equal(t, "module", cfgErr.Context)
	assert.Equal(t, "my-release", cfgErr.Name)
	assert.Contains(t, cfgErr.Error(), "my-release")
}

func TestValidateConfig_MissingRequired(t *testing.T) {
	ctx := cuecontext.New()

	// schema requires name but has no default
	schema := ctx.CompileString(`{
		name:     string
		replicas: *1 | int
	}`)
	require.NoError(t, schema.Err())

	// values does not provide name
	values := ctx.CompileString(`{
		replicas: 2
	}`)
	require.NoError(t, values.Err())

	_, cfgErr := ValidateConfig(schema, []cue.Value{values}, "module", "incomplete-release")
	require.NotNil(t, cfgErr, "missing required field should produce ConfigError")
	assert.Equal(t, "incomplete-release", cfgErr.Name)
}

func TestValidateConfig_MissingSchema(t *testing.T) {
	ctx := cuecontext.New()

	values := ctx.CompileString(`{ name: "x" }`)
	require.NoError(t, values.Err())

	// Look up a non-existent field to get a non-existing cue.Value.
	root := ctx.CompileString(`{}`)
	nonExistent := root.LookupPath(cue.ParsePath("nonexistent"))

	_, result := ValidateConfig(nonExistent, []cue.Value{values}, "module", "test")
	assert.Nil(t, result, "missing schema should return nil")
}

func TestValidateConfig_BundleContext(t *testing.T) {
	ctx := cuecontext.New()

	schema := ctx.CompileString(`{ count: int }`)
	require.NoError(t, schema.Err())

	values := ctx.CompileString(`{ count: "bad" }`)
	require.NoError(t, values.Err())

	_, cfgErr := ValidateConfig(schema, []cue.Value{values}, "bundle", "my-bundle")
	require.NotNil(t, cfgErr)
	assert.Equal(t, "bundle", cfgErr.Context)
	assert.Equal(t, "my-bundle", cfgErr.Name)
	assert.Contains(t, cfgErr.Error(), `bundle "my-bundle"`)
}
