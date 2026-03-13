package render

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeModuleWithComponents creates an in-memory CUE module value with #config
// and #components for use in SynthesizeModule tests.
func makeModuleWithComponents(ctx *cue.Context) cue.Value {
	return ctx.CompileString(`{
		metadata: {
			name:             "test-app"
			defaultNamespace: "test-ns"
			version:          "0.1.0"
			modulePath:       "example.com/modules"
		}

		#config: {
			replicas: *1 | int
			image:    *"nginx:latest" | string
		}

		#components: {
			myapp: {
				metadata: name: "myapp"
				spec: {
					replicas: #config.replicas
					image:    #config.image
				}
			}
		}
	}`)
}

func TestSynthesizeModule_Success(t *testing.T) {
	ctx := cuecontext.New()

	modVal := makeModuleWithComponents(ctx)
	require.NoError(t, modVal.Err())

	debugValues := ctx.CompileString(`{
		replicas: 3
		image:    "myapp:v1.2.3"
	}`)
	require.NoError(t, debugValues.Err())

	rel, err := SynthesizeModule(ctx, modVal, []cue.Value{debugValues}, "my-release", "my-namespace")
	require.NoError(t, err)
	require.NotNil(t, rel)
	require.NotNil(t, rel.Metadata)
	assert.Equal(t, "my-release", rel.Metadata.Name)
	assert.Equal(t, "my-namespace", rel.Metadata.Namespace)
	assert.Empty(t, rel.Metadata.UUID)
	require.NotNil(t, rel.Module.Metadata)
	assert.Equal(t, "test-app", rel.Module.Metadata.Name)
	assert.Equal(t, "test-ns", rel.Module.Metadata.DefaultNamespace)
	matchComps := rel.MatchComponents()
	assert.True(t, matchComps.Exists())
	assert.NoError(t, matchComps.Err())
	rawCompsField := rel.RawCUE.LookupPath(cue.ParsePath("components"))
	assert.True(t, rawCompsField.Exists())
	assert.True(t, rel.Config.Exists())
	assert.True(t, rel.Values.Exists())
	assert.True(t, rel.DataComponents.Exists())
	assert.NoError(t, rel.ExecuteComponents().Validate(cue.Concrete(true)))
	iter, err := matchComps.Fields()
	require.NoError(t, err)
	assert.True(t, iter.Next())
	assert.Equal(t, "myapp", iter.Selector().String())
}

func TestSynthesizeModule_ModuleGateFailure(t *testing.T) {
	ctx := cuecontext.New()
	modVal := makeModuleWithComponents(ctx)
	invalidValues := ctx.CompileString(`{
		replicas: "not-a-number"
		image:    "myapp:v1"
	}`)
	_, err := SynthesizeModule(ctx, modVal, []cue.Value{invalidValues}, "test-release", "test-ns")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test-release")
}

func TestSynthesizeModule_NoComponents(t *testing.T) {
	ctx := cuecontext.New()
	modVal := ctx.CompileString(`{
		metadata: {
			name:             "bare-module"
			defaultNamespace: "default"
			version:          "0.1.0"
			modulePath:       "example.com/modules"
		}
		#config: {
			image: *"nginx:latest" | string
		}
	}`)
	values := ctx.CompileString(`{ image: "nginx:1.25" }`)
	_, err := SynthesizeModule(ctx, modVal, []cue.Value{values}, "bare-release", "default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "#components")
}
