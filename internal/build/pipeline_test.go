package build

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/config"
)

func TestNewPipeline(t *testing.T) {
	// Test that NewPipeline creates a valid pipeline
	cfg := &config.OPMConfig{
		CueContext: cuecontext.New(),
		Registry:   "",
		Providers:  make(map[string]cue.Value),
	}
	p := NewPipeline(cfg)
	assert.NotNil(t, p)
}

func TestRenderOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    RenderOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: RenderOptions{
				ModulePath: "/some/path",
			},
			wantErr: false,
		},
		{
			name:    "missing module path",
			opts:    RenderOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPipeline_IdentityFieldsPropagated(t *testing.T) {
	// This test verifies that Identity and ReleaseIdentity fields
	// are properly propagated from ReleaseMetadata to ModuleReleaseMetadata
	// through BuiltRelease.ToModuleReleaseMetadata.

	rel := &BuiltRelease{
		Components: map[string]*LoadedComponent{
			"web": {Name: "web"},
		},
		Metadata: ReleaseMetadata{
			Name:            "my-app",
			Namespace:       "production",
			Version:         "1.0.0",
			FQN:             "example.com/modules/app@v1",
			Labels:          map[string]string{"env": "prod"},
			Identity:        "module-uuid-1234",
			ReleaseIdentity: "release-uuid-5678",
		},
	}

	meta := rel.ToModuleReleaseMetadata("app") // "app" = canonical module name

	assert.Equal(t, "my-app", meta.Name)
	assert.Equal(t, "app", meta.ModuleName, "ModuleName should be the canonical module name, not the release name")
	assert.Equal(t, "production", meta.Namespace)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Equal(t, "module-uuid-1234", meta.Identity, "Identity should be propagated")
	assert.Equal(t, "release-uuid-5678", meta.ReleaseIdentity, "ReleaseIdentity should be propagated")
	assert.Equal(t, map[string]string{"env": "prod"}, meta.Labels)
	assert.Contains(t, meta.Components, "web")
}
