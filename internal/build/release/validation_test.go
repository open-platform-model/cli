package release

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
	f := token.NewFile(filename, 0, line*100)
	for i := 0; i < line; i++ {
		f.AddLine(i * 100)
	}
	offset := (line-1)*100 + (col - 1)
	return f.Pos(offset, token.NoRelPos)
}

func TestFormatCUEDetails(t *testing.T) {
	t.Run("single CUE error with position", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`{a: string & 123}`, cue.Filename("test.cue"))
		require.Error(t, v.Err())

		details := formatCUEDetails(v.Err())
		assert.NotEmpty(t, details)
		assert.Contains(t, details, "conflicting values")
		assert.Contains(t, details, "test.cue")
		assert.Contains(t, details, "→")
	})

	t.Run("multiple CUE errors", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`{a: string & 123, b: int & "foo"}`, cue.Filename("multi.cue"))
		require.Error(t, v.Err())

		details := formatCUEDetails(v.Err())
		assert.NotEmpty(t, details)
		combined := details
		assert.Contains(t, combined, "conflicting values")
		assert.Contains(t, combined, "multi.cue")
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
		assert.Contains(t, details, "a: incomplete value")
		assert.NotContains(t, details, "→", "should not have arrow when no positions")
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
		ctx := cuecontext.New()
		v1 := ctx.CompileString(`x: string`, cue.Filename("file1.cue"))
		v2 := ctx.CompileString(`x: 123`, cue.Filename("file2.cue"))
		unified := v1.Unify(v2)
		err := unified.Validate()
		require.Error(t, err)

		details := formatCUEDetails(err)
		assert.Contains(t, details, "conflicting values")
		assert.Contains(t, details, "→")
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
		assert.Contains(t, details, "conflicting values")
		arrowCount := strings.Count(details, "→")
		assert.GreaterOrEqual(t, arrowCount, 5, "should have at least 5 errors with positions")
		lines := strings.Split(details, "\n")
		assert.Greater(t, len(lines), 5, "should have multiple lines")
	})
}

func TestCueRelPath(t *testing.T) {
	t.Run("relative path from cwd", func(t *testing.T) {
		result := cueRelPath("/home/user/project/values.cue", "/home/user/project")
		assert.Equal(t, "./values.cue", result)
	})

	t.Run("adds dot prefix when needed", func(t *testing.T) {
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
		assert.Equal(t, "../values.cue", result)
	})

	t.Run("already relative path", func(t *testing.T) {
		result := cueRelPath("values.cue", "/home/user")
		assert.Equal(t, "values.cue", result)
	})
}

func TestDeduplicateCUEErrors(t *testing.T) {
	t.Run("deduplicates identical errors", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`{a: string & 123}`, cue.Filename("test.cue"))
		require.Error(t, v.Err())

		errs := cueerrors.Errors(v.Err())
		doubled := append(errs, errs...) //nolint:gocritic // intentional: create doubled slice for dedup test
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
		assert.Equal(t, 2, len(deduped), "should deduplicate identical messages with no positions")
	})

	t.Run("same line:col different files", func(t *testing.T) {
		pos1 := testPos("file1.cue", 10, 5)
		pos2 := testPos("file2.cue", 10, 5)
		err1 := &testCUEError{pos: pos1, path: []string{"a"}, msg: "error"}
		err2 := &testCUEError{pos: pos2, path: []string{"a"}, msg: "error"}

		errs := []cueerrors.Error{err1, err2}
		deduped := deduplicateCUEErrors(errs)
		assert.Equal(t, 2, len(deduped), "should preserve errors from different files")
	})
}
