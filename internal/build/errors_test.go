package build

import (
	"errors"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCUEError is a minimal implementation of cueerrors.Error for testing.
// CUE's Newf/Wrapf can set positions but cannot set custom paths.
// This helper allows full control over path, position, and message for edge case testing.
type testCUEError struct {
	pos  token.Pos
	path []string
	msg  string
}

func (e *testCUEError) Error() string                            { return e.msg }
func (e *testCUEError) Position() token.Pos                      { return e.pos }
func (e *testCUEError) InputPositions() []token.Pos              { return nil }
func (e *testCUEError) Path() []string                           { return e.path }
func (e *testCUEError) Msg() (format string, args []interface{}) { return "%s", []interface{}{e.msg} }

// testPos creates a valid token.Pos for testing with a synthetic file/line/col.
func testPos(filename string, line, col int) token.Pos {
	// Create a file with enough space for the requested line.
	// Each line needs at least 1 byte, so allocate line * 100 to be safe.
	f := token.NewFile(filename, 0, line*100)

	// Add line offsets: line i starts at offset i*100.
	for i := 0; i < line; i++ {
		f.AddLine(i * 100)
	}

	// Calculate the offset for the requested line and column.
	// Line numbers are 1-indexed, offsets are 0-indexed.
	offset := (line-1)*100 + (col - 1)

	return f.Pos(offset, token.NoRelPos)
}

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
		// Should contain the CUE path and error message.
		assert.Contains(t, details, "conflicting values")
		// Should contain position info with arrow prefix.
		assert.Contains(t, details, "test.cue")
		assert.Contains(t, details, "→")
	})

	t.Run("multiple CUE errors", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`{a: string & 123, b: int & "foo"}`, cue.Filename("multi.cue"))
		require.Error(t, v.Err())

		details := formatCUEDetails(v.Err())
		assert.NotEmpty(t, details)
		// Should contain both errors, not just the first.
		combined := details
		assert.Contains(t, combined, "conflicting values")
		assert.Contains(t, combined, "multi.cue")
		// Each position line should have an arrow prefix.
		for _, line := range strings.Split(details, "\n") {
			if strings.Contains(line, "multi.cue") {
				assert.Contains(t, line, "→", "position lines should have arrow prefix")
			}
		}
	})

	t.Run("plain Go error passthrough", func(t *testing.T) {
		err := errors.New("not a CUE error")
		details := formatCUEDetails(err)
		assert.Contains(t, details, "not a CUE error")
	})
}

func TestCueErrorMessage(t *testing.T) {
	t.Run("single message", func(t *testing.T) {
		err := cueerrors.Newf(token.NoPos, "field not allowed")
		msg := cueErrorMessage(err)
		assert.Equal(t, "field not allowed", msg)
	})

	t.Run("wrapped chain", func(t *testing.T) {
		inner := cueerrors.Newf(token.NoPos, "inner msg")
		outer := cueerrors.Wrapf(inner, token.NoPos, "outer msg")
		msg := cueErrorMessage(outer)
		assert.Equal(t, "outer msg: inner msg", msg)
	})

	t.Run("empty format skipped", func(t *testing.T) {
		inner := cueerrors.Newf(token.NoPos, "real msg")
		wrapper := cueerrors.Wrapf(inner, token.NoPos, "")
		msg := cueErrorMessage(wrapper)
		// Empty format strings should be skipped, not produce ": : real msg"
		assert.Equal(t, "real msg", msg)
	})

	t.Run("promoted plain error tail", func(t *testing.T) {
		plainErr := errors.New("plain")
		wrapper := cueerrors.Wrapf(plainErr, token.NoPos, "cue wrapper")
		msg := cueErrorMessage(wrapper)
		assert.Equal(t, "cue wrapper: plain", msg)
	})
}

func TestFormatCUEDetailsEdgeCases(t *testing.T) {
	t.Run("error with no path", func(t *testing.T) {
		pos := testPos("test.cue", 5, 10)
		err := &testCUEError{
			path: nil,
			msg:  "top-level error",
			pos:  pos,
		}
		details := formatCUEDetails(err)
		// Should start with message directly, no path prefix.
		assert.NotContains(t, details, ": top-level error", "should not have ': ' prefix")
		assert.Contains(t, details, "top-level error")
		assert.Contains(t, details, "→")
	})

	t.Run("error with no positions", func(t *testing.T) {
		err := &testCUEError{
			path: []string{"a"},
			msg:  "incomplete value",
			pos:  token.NoPos,
		}
		details := formatCUEDetails(err)
		// Should have path and message but no colon at end, no arrow lines.
		assert.Contains(t, details, "a: incomplete value")
		assert.NotContains(t, details, "→", "should not have arrow when no positions")
		// The message should not end with a trailing colon.
		assert.NotContains(t, details, "incomplete value:")
	})

	t.Run("deeply nested path", func(t *testing.T) {
		pos := testPos("deep.cue", 1, 1)
		err := &testCUEError{
			path: []string{"a", "b", "c", "d", "e"},
			msg:  "type mismatch",
			pos:  pos,
		}
		details := formatCUEDetails(err)
		assert.Contains(t, details, "a.b.c.d.e: type mismatch")
		assert.Contains(t, details, "→")
	})

	t.Run("multiple positions per error", func(t *testing.T) {
		// CUE errors can have multiple input positions (e.g., conflicting values
		// from different locations). However, the testCUEError helper only supports
		// one position. To test multiple positions, we need a real CUE error.
		// Compile two separate values and unify them.
		ctx := cuecontext.New()
		v1 := ctx.CompileString(`x: string`, cue.Filename("file1.cue"))
		v2 := ctx.CompileString(`x: 123`, cue.Filename("file2.cue"))
		unified := v1.Unify(v2)
		err := unified.Validate()
		require.Error(t, err)

		details := formatCUEDetails(err)
		// Should have at least one error message.
		assert.Contains(t, details, "conflicting values")
		// Should have arrows for positions.
		assert.Contains(t, details, "→")
		// Count arrow occurrences - should have multiple.
		arrowCount := strings.Count(details, "→")
		assert.GreaterOrEqual(t, arrowCount, 2, "should have multiple position lines")
	})

	t.Run("5+ errors in one expression", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`{
			a: string & 1
			b: string & 2
			c: string & 3
			d: string & 4
			e: string & 5
		}`, cue.Filename("multi.cue"))
		err := v.Validate()
		require.Error(t, err)

		details := formatCUEDetails(err)
		// Should have multiple errors.
		assert.Contains(t, details, "conflicting values")
		// Count how many arrow lines we have (one per error).
		arrowCount := strings.Count(details, "→")
		assert.GreaterOrEqual(t, arrowCount, 5, "should have at least 5 errors with positions")
		// Verify newline separation - each error should be on its own block.
		lines := strings.Split(details, "\n")
		assert.Greater(t, len(lines), 5, "should have multiple lines")
	})
}

func TestCueRelPath(t *testing.T) {
	t.Run("relative path from cwd", func(t *testing.T) {
		result := cueRelPath("/home/user/project/values.cue", "/home/user/project")
		// CUE convention: always prefix with "./" for IDE compatibility.
		assert.Equal(t, "./values.cue", result)
	})

	t.Run("adds dot prefix when needed", func(t *testing.T) {
		// filepath.Rel of a subdirectory doesn't start with "."
		result := cueRelPath("/home/user/project/sub/values.cue", "/home/user/project")
		assert.True(t, strings.HasPrefix(result, "."), "should start with dot: %s", result)
		assert.Contains(t, result, "sub")
		assert.Contains(t, result, "values.cue")
	})

	t.Run("empty cwd returns original", func(t *testing.T) {
		result := cueRelPath("/absolute/path.cue", "")
		assert.Equal(t, "/absolute/path.cue", result)
	})

	t.Run("empty path returns empty", func(t *testing.T) {
		result := cueRelPath("", "/home/user")
		assert.Equal(t, "", result)
	})

	t.Run("parent directory", func(t *testing.T) {
		result := cueRelPath("/home/user/values.cue", "/home/user/project")
		// Result is "../values.cue" which starts with ".." which starts with "."
		// So no extra "./" prefix should be added.
		assert.Equal(t, "../values.cue", result)
	})

	t.Run("already relative path", func(t *testing.T) {
		result := cueRelPath("values.cue", "/home/user")
		// filepath.Rel will fail on a relative path input (not absolute).
		// Should fall back to the original path.
		assert.Equal(t, "values.cue", result)
	})
}

func TestDeduplicateCUEErrors(t *testing.T) {
	t.Run("deduplicates identical errors", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`{a: string & 123}`, cue.Filename("test.cue"))
		require.Error(t, v.Err())

		errs := cueerrors.Errors(v.Err())
		// Duplicate the list.
		doubled := append(errs, errs...)
		deduped := deduplicateCUEErrors(doubled)
		assert.Equal(t, len(errs), len(deduped), "should remove duplicates")
	})

	t.Run("preserves distinct errors", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`{a: string & 123, b: int & "foo"}`, cue.Filename("test.cue"))
		require.Error(t, v.Err())

		errs := cueerrors.Errors(v.Err())
		deduped := deduplicateCUEErrors(errs)
		assert.Equal(t, len(errs), len(deduped), "should preserve distinct errors")
	})

	t.Run("handles single error", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`{a: string & 123}`, cue.Filename("test.cue"))
		require.Error(t, v.Err())

		errs := cueerrors.Errors(v.Err())
		deduped := deduplicateCUEErrors(errs)
		assert.Equal(t, len(errs), len(deduped))
	})

	t.Run("handles empty slice", func(t *testing.T) {
		deduped := deduplicateCUEErrors(nil)
		assert.Nil(t, deduped)
	})

	t.Run("no-position errors dedup by message", func(t *testing.T) {
		err1 := &testCUEError{pos: token.NoPos, path: []string{"a"}, msg: "same"}
		err2 := &testCUEError{pos: token.NoPos, path: []string{"a"}, msg: "same"}
		err3 := &testCUEError{pos: token.NoPos, path: []string{"a"}, msg: "different"}

		errs := []cueerrors.Error{err1, err2, err3}
		deduped := deduplicateCUEErrors(errs)
		// Should have 2 errors: one "same" (deduped) and one "different".
		assert.Equal(t, 2, len(deduped), "should deduplicate identical messages with no positions")
	})

	t.Run("same line:col different files", func(t *testing.T) {
		pos1 := testPos("file1.cue", 10, 5)
		pos2 := testPos("file2.cue", 10, 5)
		err1 := &testCUEError{pos: pos1, path: []string{"a"}, msg: "error"}
		err2 := &testCUEError{pos: pos2, path: []string{"a"}, msg: "error"}

		errs := []cueerrors.Error{err1, err2}
		deduped := deduplicateCUEErrors(errs)
		// Should preserve both since they're from different files.
		assert.Equal(t, 2, len(deduped), "should preserve errors from different files")
	})
}

func TestPathRewrittenError(t *testing.T) {
	t.Run("Path returns rewritten path", func(t *testing.T) {
		inner := cueerrors.Newf(testPos("values.cue", 5, 3), "field not allowed")
		rewritten := &pathRewrittenError{
			inner:   inner,
			newPath: []string{"values", "extra"},
		}
		assert.Equal(t, []string{"values", "extra"}, rewritten.Path())
	})

	t.Run("delegates Position to inner", func(t *testing.T) {
		pos := testPos("values.cue", 10, 5)
		inner := cueerrors.Newf(pos, "some error")
		rewritten := &pathRewrittenError{inner: inner, newPath: []string{"a"}}

		assert.Equal(t, inner.Position(), rewritten.Position())
	})

	t.Run("delegates Msg to inner", func(t *testing.T) {
		inner := cueerrors.Newf(token.NoPos, "test message %d", 42)
		rewritten := &pathRewrittenError{inner: inner, newPath: []string{"a"}}

		format, args := rewritten.Msg()
		innerFormat, innerArgs := inner.Msg()
		assert.Equal(t, innerFormat, format)
		assert.Equal(t, innerArgs, args)
	})

	t.Run("formatCUEDetails renders rewritten paths", func(t *testing.T) {
		pos := testPos("values.cue", 5, 3)
		inner := cueerrors.Newf(pos, "field not allowed")
		rewritten := &pathRewrittenError{
			inner:   inner,
			newPath: []string{"values", `"extra-field"`},
		}

		details := formatCUEDetails(rewritten)
		assert.Contains(t, details, `values."extra-field"`)
		assert.Contains(t, details, "field not allowed")
		assert.Contains(t, details, "→")
	})
}

func TestRewriteErrorPath(t *testing.T) {
	t.Run("prepends base path to error path", func(t *testing.T) {
		inner := &testCUEError{
			pos:  testPos("test.cue", 1, 1),
			path: []string{"host"},
			msg:  "conflicting values",
		}
		rewritten := rewriteErrorPath(inner, []string{"values", "db"})
		assert.Equal(t, []string{"values", "db", "host"}, rewritten.Path())
	})

	t.Run("handles empty inner path", func(t *testing.T) {
		inner := &testCUEError{
			pos:  testPos("test.cue", 1, 1),
			path: nil,
			msg:  "error",
		}
		rewritten := rewriteErrorPath(inner, []string{"values", "field"})
		assert.Equal(t, []string{"values", "field"}, rewritten.Path())
	})
}

func TestFindSourcePosition(t *testing.T) {
	t.Run("single-source value returns Pos directly", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`x: 42`, cue.Filename("single.cue"))
		field := v.LookupPath(cue.ParsePath("x"))
		pos := findSourcePosition(field)
		assert.True(t, pos.IsValid())
		assert.Contains(t, pos.Filename(), "single.cue")
	})

	t.Run("unified value returns valid position", func(t *testing.T) {
		ctx := cuecontext.New()
		a := ctx.CompileString(`x: int`, cue.Filename("a.cue"))
		b := ctx.CompileString(`x: 42`, cue.Filename("b.cue"))
		unified := a.Unify(b)
		field := unified.LookupPath(cue.ParsePath("x"))
		pos := findSourcePosition(field)
		assert.True(t, pos.IsValid())
		// Should be from one of the source files.
		filename := pos.Filename()
		assert.True(t, strings.Contains(filename, "a.cue") || strings.Contains(filename, "b.cue"),
			"position should be from a.cue or b.cue, got: %s", filename)
	})

	t.Run("returns NoPos for empty value", func(t *testing.T) {
		var v cue.Value
		pos := findSourcePosition(v)
		assert.False(t, pos.IsValid())
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

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		// Should contain both the type mismatch AND the field-not-allowed error.
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

		err := validateValuesAgainstConfig(configDef, vals)
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

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "field not allowed")
		// Path should be values-rooted.
		assert.Contains(t, details, "values.extra")
		// Should NOT contain type mismatch since name is valid.
		assert.NotContains(t, details, "conflicting values")
	})

	t.Run("empty values struct", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	name: string
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		// Empty values are valid — no fields means nothing to validate.
		assert.NoError(t, err)
	})

	t.Run("closed empty config", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{extra: "nope"}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "field not allowed")
		assert.Contains(t, details, "values.extra")
	})

	t.Run("nested type mismatch", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	db: {
		host: string
		port: int
	}
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	db: {
		host: 123
		port: "wrong"
	}
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		// Should catch errors deep inside nested structs with values-rooted paths.
		assert.Contains(t, details, "conflicting values")
		assert.Contains(t, details, "values.db")
	})

	t.Run("multiple disallowed fields", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	name: string
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	name: "ok"
	x: 1
	y: 2
	z: 3
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		// Should contain "field not allowed" for the disallowed fields with values-rooted paths.
		assert.Contains(t, details, "field not allowed")
		assert.Contains(t, details, "values.x")
		assert.Contains(t, details, "values.y")
		assert.Contains(t, details, "values.z")
	})

	t.Run("top-level disallowed field has source position", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	name: string
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	name: "ok"
	extra: "not-allowed"
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		// Should have values-rooted path.
		assert.Contains(t, details, "values.extra")
		// Should have source position from values file.
		assert.Contains(t, details, "values.cue")
		assert.Contains(t, details, "→")
	})

	t.Run("nested disallowed field inside pattern-constrained struct", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	media: [string]: {
		mountPath: string
		size:      string
	}
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	media: {
		tvshows: {
			mountPath: "/data/tv"
			size:      "100Gi"
			badField:  "oops"
		}
	}
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		// Should have full nested path with values root.
		assert.Contains(t, details, "values.media.tvshows.badField")
		assert.Contains(t, details, "field not allowed")
		assert.Contains(t, details, "values.cue")
	})

	t.Run("pattern constraint fields are accepted", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	media: [string]: {
		mountPath: string
		size:      string
	}
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	media: {
		tvshows: {
			mountPath: "/data/tvshows"
			size:      "100Gi"
		}
		movies: {
			mountPath: "/data/movies"
			size:      "200Gi"
		}
	}
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		assert.NoError(t, err, "pattern-matched fields should be accepted")
	})

	t.Run("type mismatch at nested level does not recurse", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	media: [string]: {
		mountPath: string
		size:      string
	}
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	media: {
		movies: "not-a-struct"
	}
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "values.media.movies")
		assert.Contains(t, details, "conflicting values")
	})

	t.Run("optional fields in schema are accepted", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	name: string
	url?: string
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	name: "test"
	url:  "https://example.com"
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		assert.NoError(t, err, "optional fields should be accepted")
	})

	t.Run("deeply nested struct validates with full path", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	level1: {
		level2: {
			level3: {
				value: string
			}
		}
	}
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	level1: {
		level2: {
			level3: {
				value: "ok"
				bad:   "not-allowed"
			}
		}
	}
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "values.level1.level2.level3.bad")
		assert.Contains(t, details, "field not allowed")
	})

	t.Run("multi-source unified value attributes position to source file", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {
	name: string
}
`, cue.Filename("schema.cue"))

		configDef := schema.LookupPath(cue.ParsePath("#config"))

		// Simulate two values files unified together.
		a := ctx.CompileString(`{name: "test"}`, cue.Filename("base.cue"))
		b := ctx.CompileString(`{extra: "bad"}`, cue.Filename("overrides.cue"))
		unified := a.Unify(b)

		err := validateValuesAgainstConfig(configDef, unified)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "values.extra")
		assert.Contains(t, details, "field not allowed")
		// Should have a source position from the overrides file.
		assert.Contains(t, details, "overrides.cue")
	})

	t.Run("split values across files validates correctly", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string, port: int }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		a := ctx.CompileString(`{name: "test"}`, cue.Filename("base.cue"))
		b := ctx.CompileString(`{port: 8080}`, cue.Filename("env.cue"))
		unified := a.Unify(b)

		err := validateValuesAgainstConfig(configDef, unified)
		assert.NoError(t, err, "split values that together satisfy schema should pass")
	})

	t.Run("conflicting values between files uses CUE native error", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { port: int }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		a := ctx.CompileString(`{port: 8080}`, cue.Filename("a.cue"))
		b := ctx.CompileString(`{port: 9090}`, cue.Filename("b.cue"))
		unified := a.Unify(b)

		err := validateValuesAgainstConfig(configDef, unified)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "conflicting values")
		assert.Contains(t, details, "a.cue")
		assert.Contains(t, details, "b.cue")
	})

	// Edge Case Tests (from edge-case-tests.md)

	t.Run("non-identifier field names accepted by pattern", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { [Name=string]: string }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
			"my-app": "test"
			"test.key": "val"
		}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		assert.NoError(t, err, "non-identifier fields should be accepted by [string] pattern")
	})

	t.Run("non-identifier disallowed field shows quoted path", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
			name: "ok"
			"extra-field": "bad"
			"test.key": "also-bad"
		}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, `values."extra-field"`, "hyphenated field should be quoted")
		assert.Contains(t, details, `values."test.key"`, "dotted field should be quoted")
		assert.Contains(t, details, "field not allowed")
	})

	t.Run("open struct allows arbitrary fields", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string, ... }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
			name: "ok"
			anything: "goes"
			nested: {
				deep: true
				values: [1, 2, 3]
			}
		}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		assert.NoError(t, err, "open struct should allow extra fields")
	})

	t.Run("disjunction type flexibility", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { port: int | string }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		valsStr := ctx.CompileString(`{ port: "8080" }`, cue.Filename("values.cue"))
		err := validateValuesAgainstConfig(configDef, valsStr)
		assert.NoError(t, err, "string should satisfy int|string disjunction")

		valsInt := ctx.CompileString(`{ port: 8080 }`, cue.Filename("values.cue"))
		err = validateValuesAgainstConfig(configDef, valsInt)
		assert.NoError(t, err, "int should satisfy int|string disjunction")

		valsBool := ctx.CompileString(`{ port: true }`, cue.Filename("values.cue"))
		err = validateValuesAgainstConfig(configDef, valsBool)
		require.Error(t, err, "bool should NOT satisfy int|string disjunction")

		details := formatCUEDetails(err)
		assert.Contains(t, details, "values.port")
	})

	t.Run("empty struct at leaf - matching empty", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { metadata: {} }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{ metadata: {} }`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		assert.NoError(t, err, "empty struct should match empty schema struct")
	})

	t.Run("empty struct at leaf - disallowed field", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { metadata: {} }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{ metadata: { extra: "bad" } }`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "values.metadata.extra")
		assert.Contains(t, details, "field not allowed")
	})

	t.Run("list values validated as leaves - valid", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { tags: [...string] }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{ tags: ["a", "b", "c"] }`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		assert.NoError(t, err, "list of correct type should pass")
	})

	t.Run("list values validated as leaves - type mismatch", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { tags: [...string] }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{ tags: [1, 2, 3] }`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "values.tags")
		assert.Contains(t, details, "conflicting values")
	})

	t.Run("list with mixed element types", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { items: [...string] }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{ items: ["ok", 42, "also-ok"] }`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		// Error should be at the list level, not per-element
		assert.Contains(t, details, "values.items")
		assert.Contains(t, details, "conflicting values")
	})

	t.Run("numeric constraint violation", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { port: >0 & <65536 & int }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		valsNeg := ctx.CompileString(`{ port: -999 }`, cue.Filename("values.cue"))
		err := validateValuesAgainstConfig(configDef, valsNeg)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "values.port")
		assert.Contains(t, details, "invalid value", "should mention constraint violation")

		valsValid := ctx.CompileString(`{ port: 8080 }`, cue.Filename("values.cue"))
		err = validateValuesAgainstConfig(configDef, valsValid)
		assert.NoError(t, err, "value within bounds should pass")
	})

	t.Run("enum disjunction constraint", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { level: "debug" | "info" | "warn" | "error" }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		valsValid := ctx.CompileString(`{ level: "info" }`, cue.Filename("values.cue"))
		err := validateValuesAgainstConfig(configDef, valsValid)
		assert.NoError(t, err, "value in enum should pass")

		valsInvalid := ctx.CompileString(`{ level: "trace" }`, cue.Filename("values.cue"))
		err = validateValuesAgainstConfig(configDef, valsInvalid)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "values.level")
		assert.Contains(t, details, "disjunction", "should mention disjunction failure")
	})

	t.Run("errors at every recursion level", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`
#config: {
	db: {
		host: string
		port: int
	}
}
`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	topBad: "x"
	db: {
		host: 123
		port: 8080
		extra: "y"
	}
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)

		// Should have all 3 errors:
		assert.Contains(t, details, "values.topBad", "should catch top-level disallowed field")
		assert.Contains(t, details, "values.db.host", "should catch nested type mismatch")
		assert.Contains(t, details, "values.db.extra", "should catch nested disallowed field")

		// Count occurrences of "field not allowed" (should be 2: topBad + extra)
		assert.Equal(t, 2, strings.Count(details, "field not allowed"))
		// Count type mismatches (should be 1: host)
		assert.Contains(t, details, "conflicting values")
	})

	t.Run("nested pattern constraints - valid", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`
#config: {
	[Name=string]: {
		[Name=string]: {
			size: string
		}
	}
}
`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	media: {
		tvshows: {
			size: "10Gi"
		}
		movies: {
			size: "20Gi"
		}
	}
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		assert.NoError(t, err, "nested pattern-matched fields should pass")
	})

	t.Run("nested pattern constraints - disallowed at depth 2", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`
#config: {
	[Name=string]: {
		[Name=string]: {
			size: string
		}
	}
}
`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
	media: {
		tvshows: {
			size: "10Gi"
			bad: "oops"
		}
	}
}`, cue.Filename("values.cue"))

		err := validateValuesAgainstConfig(configDef, vals)
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "values.media.tvshows.bad")
		assert.Contains(t, details, "field not allowed")
	})
}
