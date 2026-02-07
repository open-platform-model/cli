package build

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmatchedComponentError(t *testing.T) {
	err := &UnmatchedComponentError{
		ComponentName: "web-server",
		Available: []TransformerSummary{
			{
				FQN:            "opmodel.dev/transformers/kubernetes@v0#DeploymentTransformer",
				RequiredLabels: map[string]string{"workload-type": "stateless"},
			},
		},
	}

	t.Run("Error message", func(t *testing.T) {
		assert.Equal(t, `component "web-server": no matching transformer`, err.Error())
	})

	t.Run("Component accessor", func(t *testing.T) {
		assert.Equal(t, "web-server", err.Component())
	})

	t.Run("implements RenderError", func(t *testing.T) {
		var renderErr RenderError = err
		assert.NotNil(t, renderErr)
		assert.Equal(t, "web-server", renderErr.Component())
	})
}

func TestUnhandledTraitError(t *testing.T) {
	tests := []struct {
		name     string
		err      *UnhandledTraitError
		wantMsg  string
		wantComp string
	}{
		{
			name: "basic unhandled trait",
			err: &UnhandledTraitError{
				ComponentName: "api-service",
				TraitFQN:      "opmodel.dev/traits@v0#AutoScaling",
				Strict:        false,
			},
			wantMsg:  `component "api-service": unhandled trait "opmodel.dev/traits@v0#AutoScaling"`,
			wantComp: "api-service",
		},
		{
			name: "strict mode",
			err: &UnhandledTraitError{
				ComponentName: "worker",
				TraitFQN:      "opmodel.dev/traits@v0#Monitoring",
				Strict:        true,
			},
			wantMsg:  `component "worker": unhandled trait "opmodel.dev/traits@v0#Monitoring"`,
			wantComp: "worker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
			assert.Equal(t, tt.wantComp, tt.err.Component())
		})
	}

	t.Run("implements RenderError", func(t *testing.T) {
		var renderErr RenderError = &UnhandledTraitError{
			ComponentName: "test",
			TraitFQN:      "test-trait",
		}
		assert.NotNil(t, renderErr)
	})
}

func TestTransformError(t *testing.T) {
	cause := errors.New("CUE evaluation failed: field not found")
	err := &TransformError{
		ComponentName:  "database",
		TransformerFQN: "opmodel.dev/transformers/kubernetes@v0#StatefulsetTransformer",
		Cause:          cause,
	}

	t.Run("Error message", func(t *testing.T) {
		expected := `component "database", transformer "opmodel.dev/transformers/kubernetes@v0#StatefulsetTransformer": CUE evaluation failed: field not found`
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Component accessor", func(t *testing.T) {
		assert.Equal(t, "database", err.Component())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		assert.Equal(t, cause, err.Unwrap())
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("implements RenderError", func(t *testing.T) {
		var renderErr RenderError = err
		assert.NotNil(t, renderErr)
		assert.Equal(t, "database", renderErr.Component())
	})
}

func TestRenderErrorInterface(t *testing.T) {
	// Verify all error types implement RenderError at compile time
	var _ RenderError = (*UnmatchedComponentError)(nil)
	var _ RenderError = (*UnhandledTraitError)(nil)
	var _ RenderError = (*TransformError)(nil)

	// Also verify they implement the standard error interface
	var _ error = (*UnmatchedComponentError)(nil)
	var _ error = (*UnhandledTraitError)(nil)
	var _ error = (*TransformError)(nil)
}

func TestTransformerSummary(t *testing.T) {
	summary := TransformerSummary{
		FQN: "opmodel.dev/transformers/kubernetes@v0#DeploymentTransformer",
		RequiredLabels: map[string]string{
			"workload-type": "stateless",
		},
		RequiredResources: []string{"opmodel.dev/resources@v0#Container"},
		RequiredTraits:    []string{},
	}

	assert.Equal(t, "opmodel.dev/transformers/kubernetes@v0#DeploymentTransformer", summary.FQN)
	assert.Equal(t, "stateless", summary.RequiredLabels["workload-type"])
	assert.Len(t, summary.RequiredResources, 1)
	assert.Empty(t, summary.RequiredTraits)
}
