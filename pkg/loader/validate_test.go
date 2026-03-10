package loader

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/pkg/releaseprocess"
)

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

	_, err := releaseprocess.ValidateConfig(schema, []cue.Value{values}, "module", "test-release")
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

	_, cfgErr := releaseprocess.ValidateConfig(schema, []cue.Value{values}, "module", "my-release")
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

	_, cfgErr := releaseprocess.ValidateConfig(schema, []cue.Value{values}, "module", "incomplete-release")
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

	_, result := releaseprocess.ValidateConfig(nonExistent, []cue.Value{values}, "module", "test")
	assert.Nil(t, result, "missing schema should return nil")
}

func TestValidateConfig_BundleContext(t *testing.T) {
	ctx := cuecontext.New()

	schema := ctx.CompileString(`{ count: int }`)
	require.NoError(t, schema.Err())

	values := ctx.CompileString(`{ count: "bad" }`)
	require.NoError(t, values.Err())

	_, cfgErr := releaseprocess.ValidateConfig(schema, []cue.Value{values}, "bundle", "my-bundle")
	require.NotNil(t, cfgErr)
	assert.Equal(t, "bundle", cfgErr.Context)
	assert.Equal(t, "my-bundle", cfgErr.Name)
	assert.Contains(t, cfgErr.Error(), `bundle "my-bundle"`)
}
