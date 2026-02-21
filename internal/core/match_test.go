package core

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformerMatchPlan_Execute_EmptyPlan(t *testing.T) {
	cueCtx := cuecontext.New()
	plan := &TransformerMatchPlan{
		Matches:   []*TransformerMatch{},
		Unmatched: []string{},
		cueCtx:    cueCtx,
	}
	rel := &ModuleRelease{
		Metadata: &ReleaseMetadata{Name: "test", Namespace: "default"},
		Module:   Module{Metadata: &ModuleMetadata{Name: "test", Version: "1.0.0"}},
	}

	resources, errs := plan.Execute(context.Background(), rel)

	assert.NotNil(t, resources, "Execute should return a non-nil slice when there are no matches")
	assert.Empty(t, resources)
	assert.Empty(t, errs)
}

func TestTransformerMatchPlan_Execute_UnmatchedSkipped(t *testing.T) {
	// Matches with Matched=false should be skipped; only matched pairs are executed.
	cueCtx := cuecontext.New()
	plan := &TransformerMatchPlan{
		Matches: []*TransformerMatch{
			{
				Matched: false,
				Detail:  &TransformerMatchDetail{ComponentName: "web", TransformerFQN: "test#Tf"},
			},
		},
		cueCtx: cueCtx,
	}
	rel := &ModuleRelease{
		Metadata: &ReleaseMetadata{Name: "test", Namespace: "default"},
		Module:   Module{Metadata: &ModuleMetadata{Name: "test", Version: "1.0.0"}},
	}

	resources, errs := plan.Execute(context.Background(), rel)

	assert.NotNil(t, resources)
	assert.Empty(t, resources)
	assert.Empty(t, errs)
}

func TestTransformerMatchPlan_Execute_SingleResource(t *testing.T) {
	cueCtx := cuecontext.New()

	transformerCUE := cueCtx.CompileString(`{
		#transform: {
			#component: _
			#context: {
				name: string
				namespace: string
				...
			}
			output: {
				apiVersion: "apps/v1"
				kind: "Deployment"
				metadata: {
					name: "test-deploy"
					namespace: "default"
				}
				spec: {}
			}
		}
	}`)
	require.NoError(t, transformerCUE.Err())

	componentCUE := cueCtx.CompileString(`{
		metadata: { name: "web" }
		spec: {}
	}`)
	require.NoError(t, componentCUE.Err())

	transformValue := transformerCUE.LookupPath(cue.ParsePath("#transform"))

	tf := &Transformer{
		Metadata:  &TransformerMetadata{Name: "DeploymentTransformer", FQN: "test#DeploymentTransformer"},
		Transform: transformValue,
	}
	comp := &Component{
		Metadata:  &ComponentMetadata{Name: "web", Labels: map[string]string{}, Annotations: map[string]string{}},
		Resources: map[string]cue.Value{},
		Traits:    map[string]cue.Value{},
		Value:     componentCUE,
	}

	plan := &TransformerMatchPlan{
		cueCtx: cueCtx,
		Matches: []*TransformerMatch{
			{Matched: true, Transformer: tf, Component: comp},
		},
	}
	rel := &ModuleRelease{
		Metadata: &ReleaseMetadata{Name: "test", Namespace: "default", Labels: map[string]string{}},
		Module:   Module{Metadata: &ModuleMetadata{Name: "test", Version: "1.0.0", FQN: "test@v0#Test", Labels: map[string]string{}}},
	}

	resources, errs := plan.Execute(context.Background(), rel)

	require.Empty(t, errs)
	require.Len(t, resources, 1)
	assert.Equal(t, "Deployment", resources[0].Object.GetKind())
	assert.Equal(t, "test-deploy", resources[0].Object.GetName())
}

func TestTransformerMatchPlan_Execute_ContextCancellation(t *testing.T) {
	cueCtx := cuecontext.New()

	// A plan with a matched entry but a cancelled context: should stop immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Execute is called

	plan := &TransformerMatchPlan{
		cueCtx: cueCtx,
		Matches: []*TransformerMatch{
			{
				Matched: true,
				Transformer: &Transformer{
					Metadata:  &TransformerMetadata{Name: "Tf", FQN: "test#Tf"},
					Transform: cueCtx.CompileString(`{}`),
				},
				Component: &Component{
					Metadata:  &ComponentMetadata{Name: "web"},
					Resources: map[string]cue.Value{},
					Traits:    map[string]cue.Value{},
					Value:     cueCtx.CompileString(`{}`),
				},
			},
		},
	}
	rel := &ModuleRelease{
		Metadata: &ReleaseMetadata{Name: "test", Namespace: "default"},
		Module:   Module{Metadata: &ModuleMetadata{Name: "test", Version: "1.0.0"}},
	}

	resources, errs := plan.Execute(ctx, rel)

	// Should have stopped immediately with a cancellation error.
	assert.NotNil(t, resources, "resources slice should be non-nil even on cancellation")
	require.Len(t, errs, 1)
	assert.ErrorIs(t, errs[0], context.Canceled)
}
