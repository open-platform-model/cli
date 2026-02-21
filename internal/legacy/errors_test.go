package build

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/legacy/transform"
)

func TestUnmatchedComponentError(t *testing.T) {
	err := &UnmatchedComponentError{
		ComponentName: "web-server",
		Available: []core.TransformerRequirements{
			&transform.LoadedTransformer{
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

func TestTransformError(t *testing.T) {
	cause := errors.New("CUE evaluation failed: field not found")
	err := &core.TransformError{
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
	var _ RenderError = (*core.TransformError)(nil)

	// Also verify they implement the standard error interface
	var _ error = (*UnmatchedComponentError)(nil)
	var _ error = (*core.TransformError)(nil)
}

func TestReleaseValidationError(t *testing.T) {
	t.Run("message only", func(t *testing.T) {
		err := &core.ValidationError{
			Message: "module missing 'values' field",
		}
		assert.Equal(t, "release validation failed: module missing 'values' field", err.Error())
		assert.Nil(t, err.Unwrap())
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("some underlying error")
		err := &core.ValidationError{
			Message: "failed to inject values",
			Cause:   cause,
		}
		assert.Contains(t, err.Error(), "failed to inject values")
		assert.Contains(t, err.Error(), "some underlying error")
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("with details", func(t *testing.T) {
		err := &core.ValidationError{
			Message: "failed to inject values",
			Cause:   errors.New("dummy"),
			Details: "values.foo: conflicting values\n    ./test.cue:1:5",
		}
		// Error() should NOT include details (they are printed separately by the command layer)
		assert.Contains(t, err.Error(), "failed to inject values")
		// Details are stored for structured printing
		assert.Contains(t, err.Details, "values.foo")
		assert.Contains(t, err.Details, "./test.cue:1:5")
	})
}
