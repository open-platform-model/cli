package build

import (
	"errors"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestReleaseValidationError(t *testing.T) {
	t.Run("message only", func(t *testing.T) {
		err := &ReleaseValidationError{
			Message: "module missing 'values' field",
		}
		assert.Equal(t, "release validation failed: module missing 'values' field", err.Error())
		assert.Nil(t, err.Unwrap())
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("some underlying error")
		err := &ReleaseValidationError{
			Message: "failed to inject values",
			Cause:   cause,
		}
		assert.Contains(t, err.Error(), "failed to inject values")
		assert.Contains(t, err.Error(), "some underlying error")
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("with details", func(t *testing.T) {
		err := &ReleaseValidationError{
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

func TestFormatCUEDetails(t *testing.T) {
	t.Run("single CUE error with position", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`{a: string & 123}`, cue.Filename("test.cue"))
		require.Error(t, v.Err())

		details := formatCUEDetails(v.Err())
		assert.NotEmpty(t, details)
		// Should contain the CUE path and error message
		assert.Contains(t, details, "conflicting values")
		// Should contain position info
		assert.Contains(t, details, "test.cue")
	})

	t.Run("multiple CUE errors", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`{a: string & 123, b: int & "foo"}`, cue.Filename("multi.cue"))
		require.Error(t, v.Err())

		details := formatCUEDetails(v.Err())
		assert.NotEmpty(t, details)
		// Should contain both errors, not just the first
		lines := strings.Split(details, "\n")
		// At minimum we should see error text for both fields
		combined := strings.Join(lines, " ")
		assert.Contains(t, combined, "conflicting values")
		assert.Contains(t, combined, "multi.cue")
	})

	t.Run("plain Go error passthrough", func(t *testing.T) {
		err := errors.New("not a CUE error")
		details := formatCUEDetails(err)
		assert.Contains(t, details, "not a CUE error")
	})
}

func TestValidateValuesAgainstConfig(t *testing.T) {
	t.Run("catches both type mismatch and disallowed field", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	name: string
	media: [string]: {
		mountPath: string
		size:      string
	}
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	name: "test"
	media: {
		bad: "wrong-type"
	}
	extra: "not-allowed"
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(ctx, configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		// Should contain both the type mismatch AND the field-not-allowed error
		assert.Contains(t, details, "conflicting values")
		assert.Contains(t, details, "field not allowed")
	})

	t.Run("returns nil for valid values", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	name: string
	port: int
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	name: "valid"
	port: 8080
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(ctx, configDef, vals)
		assert.NoError(t, err)
	})

	t.Run("catches single error", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	name: string
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	name: "valid"
	extra: "not-allowed"
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(ctx, configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "field not allowed")
		// Should NOT contain type mismatch since name is valid
		assert.NotContains(t, details, "conflicting values")
	})
}
