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
