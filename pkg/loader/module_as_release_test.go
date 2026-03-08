package loader

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeModuleWithComponents creates an in-memory CUE module value with #config
// and #components for use in SynthesizeModuleRelease tests.
//
// The module has:
//   - metadata.name, metadata.defaultNamespace
//   - #config: { replicas: int, image: string }
//   - #components.myapp: references #config
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

// TestSynthesizeModuleRelease_Success verifies that a valid module + debugValues
// produces a properly populated ModuleRelease.
func TestSynthesizeModuleRelease_Success(t *testing.T) {
	ctx := cuecontext.New()

	modVal := makeModuleWithComponents(ctx)
	require.NoError(t, modVal.Err())

	debugValues := ctx.CompileString(`{
		replicas: 3
		image:    "myapp:v1.2.3"
	}`)
	require.NoError(t, debugValues.Err())

	rel, err := SynthesizeModuleRelease(ctx, modVal, debugValues, "my-release", "my-namespace")
	require.NoError(t, err)
	require.NotNil(t, rel)

	// Metadata must be populated.
	require.NotNil(t, rel.Metadata)
	assert.Equal(t, "my-release", rel.Metadata.Name)
	assert.Equal(t, "my-namespace", rel.Metadata.Namespace)
	assert.Empty(t, rel.Metadata.UUID, "synthesized UUID must be empty")

	// Module metadata must be decoded from the module value.
	require.NotNil(t, rel.Module.Metadata)
	assert.Equal(t, "test-app", rel.Module.Metadata.Name)
	assert.Equal(t, "test-ns", rel.Module.Metadata.DefaultNamespace)

	// MatchComponents must return a value with a "components" field that the
	// engine match plan can iterate.
	matchComps := rel.MatchComponents()
	assert.True(t, matchComps.Exists(), "MatchComponents() must return an existing value")
	assert.NoError(t, matchComps.Err())

	// The synthetic schema must expose "components" at the top level.
	schemaCompsField := rel.Schema().LookupPath(cue.ParsePath("components"))
	assert.True(t, schemaCompsField.Exists(), "schema must have 'components' field for MatchComponents()")
}

// TestSynthesizeModuleRelease_ModuleGateFailure verifies that invalid values
// that violate #config constraints return an error.
func TestSynthesizeModuleRelease_ModuleGateFailure(t *testing.T) {
	ctx := cuecontext.New()

	modVal := makeModuleWithComponents(ctx)
	require.NoError(t, modVal.Err())

	// Provide a string where int is expected.
	invalidValues := ctx.CompileString(`{
		replicas: "not-a-number"
		image:    "myapp:v1"
	}`)
	require.NoError(t, invalidValues.Err())

	_, err := SynthesizeModuleRelease(ctx, modVal, invalidValues, "test-release", "test-ns")
	require.Error(t, err, "invalid values should fail Module Gate")
	assert.Contains(t, err.Error(), "test-release")
}

// TestSynthesizeModuleRelease_NoComponents verifies the error when the module
// has no #components field.
func TestSynthesizeModuleRelease_NoComponents(t *testing.T) {
	ctx := cuecontext.New()

	// Module without #components.
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
	require.NoError(t, modVal.Err())

	values := ctx.CompileString(`{ image: "nginx:1.25" }`)
	require.NoError(t, values.Err())

	_, err := SynthesizeModuleRelease(ctx, modVal, values, "bare-release", "default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "#components")
}

// TestSynthesizeModuleRelease_ConcreteComponents verifies task 2.3:
// ExecuteComponents() must return a fully concrete value with no open constraints.
func TestSynthesizeModuleRelease_ConcreteComponents(t *testing.T) {
	ctx := cuecontext.New()

	modVal := makeModuleWithComponents(ctx)
	require.NoError(t, modVal.Err())

	debugValues := ctx.CompileString(`{
		replicas: 2
		image:    "nginx:1.25"
	}`)
	require.NoError(t, debugValues.Err())

	rel, err := SynthesizeModuleRelease(ctx, modVal, debugValues, "concrete-release", "default")
	require.NoError(t, err)

	// Task 2.3: ExecuteComponents must be fully concrete.
	execComps := rel.ExecuteComponents()
	assert.NoError(t, execComps.Validate(cue.Concrete(true)),
		"ExecuteComponents() should be fully concrete (no open constraints)")
}

// TestSynthesizeModuleRelease_ReleaseName verifies the release name and
// namespace flow through to the output metadata.
func TestSynthesizeModuleRelease_ReleaseName(t *testing.T) {
	ctx := cuecontext.New()

	modVal := makeModuleWithComponents(ctx)
	require.NoError(t, modVal.Err())

	debugValues := ctx.CompileString(`{
		replicas: 1
		image:    "nginx:latest"
	}`)
	require.NoError(t, debugValues.Err())

	const wantName = "my-custom-name"
	const wantNamespace = "custom-namespace"

	rel, err := SynthesizeModuleRelease(ctx, modVal, debugValues, wantName, wantNamespace)
	require.NoError(t, err)

	assert.Equal(t, wantName, rel.Metadata.Name)
	assert.Equal(t, wantNamespace, rel.Metadata.Namespace)
}

// TestSynthesizeModuleRelease_MatchComponentsIterable verifies task 2.2:
// MatchComponents() returns a valid, iterable CUE value for the match plan.
func TestSynthesizeModuleRelease_MatchComponentsIterable(t *testing.T) {
	ctx := cuecontext.New()

	modVal := makeModuleWithComponents(ctx)
	require.NoError(t, modVal.Err())

	debugValues := ctx.CompileString(`{
		replicas: 1
		image:    "nginx:latest"
	}`)
	require.NoError(t, debugValues.Err())

	rel, err := SynthesizeModuleRelease(ctx, modVal, debugValues, "iter-release", "default")
	require.NoError(t, err)

	// MatchComponents() should return the components struct from the schema.
	matchComps := rel.MatchComponents()
	require.True(t, matchComps.Exists())
	require.NoError(t, matchComps.Err())

	// Must be iterable as a struct (the match plan calls Fields() on it).
	iter, err := matchComps.Fields()
	require.NoError(t, err)
	assert.True(t, iter.Next(), "should have at least one component (myapp)")
	assert.Equal(t, "myapp", iter.Selector().String())
}
