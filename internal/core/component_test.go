package core_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/core"
)

func TestComponent_Validate(t *testing.T) {
	ctx := cuecontext.New()
	val := ctx.CompileString(`{}`)

	t.Run("nil metadata returns error", func(t *testing.T) {
		c := &core.Component{Value: val}
		err := c.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "metadata is nil")
	})

	t.Run("empty name returns error", func(t *testing.T) {
		c := &core.Component{
			Metadata:  &core.ComponentMetadata{Name: ""},
			Resources: map[string]cue.Value{"r": val},
			Value:     val,
		}
		err := c.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is empty")
	})

	t.Run("no resources returns error", func(t *testing.T) {
		c := &core.Component{
			Metadata:  &core.ComponentMetadata{Name: "mycomp"},
			Resources: map[string]cue.Value{},
			Value:     val,
		}
		err := c.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no resources")
	})

	t.Run("zero CUE value returns error", func(t *testing.T) {
		c := &core.Component{
			Metadata:  &core.ComponentMetadata{Name: "mycomp"},
			Resources: map[string]cue.Value{"r": val},
			Value:     cue.Value{},
		}
		err := c.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no CUE value")
	})

	t.Run("valid component passes", func(t *testing.T) {
		c := &core.Component{
			Metadata:  &core.ComponentMetadata{Name: "mycomp"},
			Resources: map[string]cue.Value{"r": val},
			Value:     val,
		}
		assert.NoError(t, c.Validate())
	})
}

func TestComponent_IsConcrete(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("concrete value returns true", func(t *testing.T) {
		val := ctx.CompileString(`{name: "app", replicas: 3}`)
		c := &core.Component{Value: val}
		assert.True(t, c.IsConcrete())
	})

	t.Run("non-concrete value returns false", func(t *testing.T) {
		val := ctx.CompileString(`{name: string, replicas: int}`)
		c := &core.Component{Value: val}
		assert.False(t, c.IsConcrete())
	})
}

func TestExtractComponents(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("extracts components with name, labels, resources, traits", func(t *testing.T) {
		v := ctx.CompileString(`{
			web: {
				metadata: {
					name: "web-server"
					labels: { "app": "web" }
					annotations: { "opm.dev/desc": "web component" }
				}
				#resources: {
					"opmodel.dev/core/v0#Deployment": {}
				}
				#traits: {
					"opmodel.dev/core/v0#Ingress": {}
				}
				spec: {
					replicas: 2
					image: "nginx:latest"
				}
			}
		}`)
		require.NoError(t, v.Err())

		components, err := core.ExtractComponents(v)
		require.NoError(t, err)
		require.Len(t, components, 1)

		comp := components["web"]
		require.NotNil(t, comp)
		assert.Equal(t, "web-server", comp.Metadata.Name)
		assert.Equal(t, "web", comp.Metadata.Labels["app"])
		assert.Equal(t, "web component", comp.Metadata.Annotations["opm.dev/desc"])
		assert.Len(t, comp.Resources, 1)
		assert.Len(t, comp.Traits, 1)
		// Spec must be extracted
		assert.True(t, comp.Spec.Exists(), "Spec must be extracted from spec field")
		// Blueprints must be initialized (non-nil) even when absent
		assert.NotNil(t, comp.Blueprints, "Blueprints must be a non-nil map")
	})

	t.Run("extracts blueprints when present", func(t *testing.T) {
		v := ctx.CompileString(`{
			svc: {
				#resources: {
					"opmodel.dev/core/v0#Deployment": {}
				}
				#blueprints: {
					"opmodel.dev/blueprints/v0#Standard": {}
				}
			}
		}`)
		require.NoError(t, v.Err())

		components, err := core.ExtractComponents(v)
		require.NoError(t, err)
		comp := components["svc"]
		require.NotNil(t, comp)
		assert.Len(t, comp.Blueprints, 1)
		assert.Contains(t, comp.Blueprints, "opmodel.dev/blueprints/v0#Standard")
	})

	t.Run("uses field name when metadata.name absent", func(t *testing.T) {
		v := ctx.CompileString(`{
			mycomp: {
				#resources: {
					"opmodel.dev/core/v0#Deployment": {}
				}
			}
		}`)
		require.NoError(t, v.Err())

		components, err := core.ExtractComponents(v)
		require.NoError(t, err)
		comp := components["mycomp"]
		require.NotNil(t, comp)
		assert.Equal(t, "mycomp", comp.Metadata.Name)
	})

	t.Run("returns error when component has no resources", func(t *testing.T) {
		v := ctx.CompileString(`{
			bad: {
				metadata: { name: "bad" }
			}
		}`)
		require.NoError(t, v.Err())

		_, err := core.ExtractComponents(v)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad")
		assert.Contains(t, err.Error(), "no resources")
	})

	t.Run("multiple components extracted", func(t *testing.T) {
		v := ctx.CompileString(`{
			a: {
				#resources: { r1: {} }
			}
			b: {
				#resources: { r2: {} }
			}
		}`)
		require.NoError(t, v.Err())

		components, err := core.ExtractComponents(v)
		require.NoError(t, err)
		assert.Len(t, components, 2)
		assert.Contains(t, components, "a")
		assert.Contains(t, components, "b")
	})
}
