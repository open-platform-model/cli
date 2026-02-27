package builder

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opmerrors "github.com/opmodel/cli/internal/errors"
)

// --- test helpers -----------------------------------------------------------

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

// testPos creates a synthetic token.Pos for testing.
func testPos(filename string, line, col int) token.Pos {
	f := token.NewFile(filename, 0, line*100)
	for i := 0; i < line; i++ {
		f.AddLine(i * 100)
	}
	offset := (line-1)*100 + (col - 1)
	return f.Pos(offset, token.NoRelPos)
}

// cueErrorPaths returns all error paths from a cueerrors.Error as a
// space-separated string. Used to assert on recursive walker output.
func cueErrorPaths(err error) string {
	errs := cueerrors.Errors(err)
	paths := make([]string, 0, len(errs))
	for _, e := range errs {
		if p := strings.Join(e.Path(), "."); p != "" {
			paths = append(paths, p)
		}
	}
	return strings.Join(paths, " ")
}

// collectPaths extracts the Path field from each FieldError.
func collectPaths(err *opmerrors.ValuesValidationError) []string {
	paths := make([]string, 0, len(err.Errors))
	for _, fe := range err.Errors {
		paths = append(paths, fe.Path)
	}
	return paths
}

// rewriteErrorPath wraps a CUE error with a prepended base path.
// Used in tests to verify pathRewrittenError behaviour.
func rewriteErrorPath(e cueerrors.Error, basePath []string) cueerrors.Error {
	errPath := e.Path()
	newPath := make([]string, 0, len(basePath)+len(errPath))
	newPath = append(newPath, basePath...)
	newPath = append(newPath, errPath...)
	return &pathRewrittenError{inner: e, newPath: newPath}
}

// --- pathRewrittenError unit tests ------------------------------------------

func TestPathRewrittenError(t *testing.T) {
	t.Run("Path returns rewritten path", func(t *testing.T) {
		inner := cueerrors.Newf(testPos("values.cue", 5, 3), "field not allowed")
		rewritten := &pathRewrittenError{
			inner:   inner,
			newPath: []string{"values", "extra"},
		}
		assert.Equal(t, []string{"values", "extra"}, rewritten.Path())
	})

	t.Run("delegates Position to inner when no override", func(t *testing.T) {
		pos := testPos("values.cue", 10, 5)
		inner := cueerrors.Newf(pos, "some error")
		rewritten := &pathRewrittenError{inner: inner, newPath: []string{"a"}}
		assert.Equal(t, inner.Position(), rewritten.Position())
	})

	t.Run("posOverride takes precedence over inner Position", func(t *testing.T) {
		innerPos := testPos("schema.cue", 1, 1)
		valuesPos := testPos("values-prod.cue", 7, 12)
		inner := cueerrors.Newf(innerPos, "conflicting values")
		rewritten := &pathRewrittenError{
			inner:       inner,
			newPath:     []string{"values", "port"},
			posOverride: valuesPos,
		}
		got := rewritten.Position()
		assert.Equal(t, valuesPos, got)
		assert.Contains(t, got.Filename(), "values-prod.cue")
		assert.Equal(t, 7, got.Line())
	})

	t.Run("delegates Msg to inner", func(t *testing.T) {
		inner := cueerrors.Newf(token.NoPos, "test message %d", 42)
		rewritten := &pathRewrittenError{inner: inner, newPath: []string{"a"}}
		format, args := rewritten.Msg()
		innerFormat, innerArgs := inner.Msg()
		assert.Equal(t, innerFormat, format)
		assert.Equal(t, innerArgs, args)
	})

	t.Run("Error returns inner message", func(t *testing.T) {
		inner := cueerrors.Newf(token.NoPos, "field not allowed")
		rewritten := &pathRewrittenError{
			inner:   inner,
			newPath: []string{"values", `"extra-field"`},
		}
		assert.Contains(t, rewritten.Error(), "field not allowed")
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

// --- findSourcePosition unit tests ------------------------------------------

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

// --- validateFieldsRecursive unit tests -------------------------------------

func TestValidateFieldsRecursive(t *testing.T) {
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

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)
		assert.Contains(t, cueerrors.Details(err, nil), "field not allowed")
		assert.Contains(t, cueErrorPaths(err), "values.extra")
	})

	t.Run("returns nil for valid values", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string, port: int }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		vals := ctx.CompileString(`{ name: "valid", port: 8080 }`, cue.Filename("values.cue"))
		assert.NoError(t, validateFieldsRecursive(configDef, vals, []string{"values"}, nil))
	})

	t.Run("catches single disallowed field", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		vals := ctx.CompileString(`{ name: "valid", extra: "not-allowed" }`, cue.Filename("values.cue"))

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)
		errs := cueerrors.Errors(err)
		require.Len(t, errs, 1)
		assert.Equal(t, []string{"values", "extra"}, errs[0].Path())
		assert.Contains(t, errs[0].Error(), "field not allowed")
	})

	t.Run("empty values struct", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		vals := ctx.CompileString(`{}`, cue.Filename("values.cue"))
		assert.NoError(t, validateFieldsRecursive(configDef, vals, []string{"values"}, nil))
	})

	t.Run("closed empty config rejects extra fields", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: {}`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		vals := ctx.CompileString(`{extra: "nope"}`, cue.Filename("values.cue"))

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)
		errs := cueerrors.Errors(err)
		require.Len(t, errs, 1)
		assert.Equal(t, []string{"values", "extra"}, errs[0].Path())
	})

	t.Run("nested type mismatch", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { db: { host: string, port: int } }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		vals := ctx.CompileString(`{ db: { host: 123, port: "wrong" } }`, cue.Filename("values.cue"))

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)
		assert.Contains(t, cueErrorPaths(err), "values.db")
	})

	t.Run("multiple disallowed fields all reported", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		vals := ctx.CompileString(`{ name: "ok", x: 1, y: 2, z: 3 }`, cue.Filename("values.cue"))

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)
		paths := cueErrorPaths(err)
		assert.Contains(t, paths, "values.x")
		assert.Contains(t, paths, "values.y")
		assert.Contains(t, paths, "values.z")
	})

	t.Run("pattern constraint fields are accepted", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { media: [string]: { mountPath: string, size: string } }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		vals := ctx.CompileString(`{
	media: {
		tvshows: { mountPath: "/data/tvshows", size: "100Gi" }
		movies:  { mountPath: "/data/movies",  size: "200Gi" }
	}
}`, cue.Filename("values.cue"))
		assert.NoError(t, validateFieldsRecursive(configDef, vals, []string{"values"}, nil),
			"pattern-matched fields should be accepted")
	})

	t.Run("disallowed field inside pattern struct", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { media: [string]: { mountPath: string, size: string } }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		vals := ctx.CompileString(`{
	media: {
		tvshows: { mountPath: "/data/tv", size: "100Gi", badField: "oops" }
	}
}`, cue.Filename("values.cue"))

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)
		assert.Contains(t, cueErrorPaths(err), "values.media.tvshows.badField")
	})

	t.Run("split values unified before validation passes", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string, port: int }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		a := ctx.CompileString(`{name: "test"}`, cue.Filename("base.cue"))
		b := ctx.CompileString(`{port: 8080}`, cue.Filename("env.cue"))
		assert.NoError(t, validateFieldsRecursive(configDef, a.Unify(b), []string{"values"}, nil),
			"split values that together satisfy schema should pass")
	})

	t.Run("open struct allows arbitrary fields", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string, ... }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		vals := ctx.CompileString(`{ name: "ok", anything: "goes" }`, cue.Filename("values.cue"))
		assert.NoError(t, validateFieldsRecursive(configDef, vals, []string{"values"}, nil),
			"open struct should allow extra fields")
	})

	t.Run("deeply nested path on disallowed field", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`
#config: { level1: { level2: { level3: { value: string } } } }
`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))
		vals := ctx.CompileString(`{
	level1: { level2: { level3: { value: "ok", bad: "not-allowed" } } }
}`, cue.Filename("values.cue"))

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)
		assert.Contains(t, cueErrorPaths(err), "values.level1.level2.level3.bad")
	})
}

// --- ValidateValues integration tests (use real temp files) -----------------

func TestValidateValues(t *testing.T) {
	t.Run("valid single file returns unified value", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string, port: int }`, cue.Filename("schema.cue"))
		config := schema.LookupPath(cue.ParsePath("#config"))

		path := writeTempCUE(t, "values.cue", `values: { name: "ok", port: 8080 }`)
		got, err := ValidateValues(ctx, config, []string{path})
		require.NoError(t, err)
		assert.True(t, got.Exists())
	})

	t.Run("absent config skips schema validation", func(t *testing.T) {
		ctx := cuecontext.New()
		path := writeTempCUE(t, "values.cue", `values: { anything: "goes" }`)
		got, err := ValidateValues(ctx, cue.Value{}, []string{path})
		require.NoError(t, err)
		assert.True(t, got.Exists())
	})

	t.Run("type mismatch returns ValuesValidationError with file attribution", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { port: int }`, cue.Filename("schema.cue"))
		config := schema.LookupPath(cue.ParsePath("#config"))

		path := writeTempCUE(t, "prod-values.cue", `values: { port: "not-a-number" }`)
		_, err := ValidateValues(ctx, config, []string{path})
		require.Error(t, err)

		var valErr *opmerrors.ValuesValidationError
		require.ErrorAs(t, err, &valErr)
		require.NotEmpty(t, valErr.Errors)
		assert.Equal(t, "prod-values.cue", valErr.Errors[0].File)
		assert.Greater(t, valErr.Errors[0].Line, 0)
		assert.Contains(t, valErr.Errors[0].Path, "values.port")
	})

	t.Run("disallowed field returns ValuesValidationError", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string }`, cue.Filename("schema.cue"))
		config := schema.LookupPath(cue.ParsePath("#config"))

		path := writeTempCUE(t, "values.cue", `values: { name: "ok", extra: "bad" }`)
		_, err := ValidateValues(ctx, config, []string{path})
		require.Error(t, err)

		var valErr *opmerrors.ValuesValidationError
		require.ErrorAs(t, err, &valErr)
		paths := collectPaths(valErr)
		assert.Contains(t, paths, "values.extra")
	})

	t.Run("all errors collected not just first", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string }`, cue.Filename("schema.cue"))
		config := schema.LookupPath(cue.ParsePath("#config"))

		path := writeTempCUE(t, "values.cue", `values: { name: "ok", x: 1, y: 2 }`)
		_, err := ValidateValues(ctx, config, []string{path})
		require.Error(t, err)

		var valErr *opmerrors.ValuesValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Len(t, valErr.Errors, 2)
		paths := collectPaths(valErr)
		assert.Contains(t, paths, "values.x")
		assert.Contains(t, paths, "values.y")
	})

	t.Run("two compatible files unified successfully", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string, port: int }`, cue.Filename("schema.cue"))
		config := schema.LookupPath(cue.ParsePath("#config"))

		a := writeTempCUE(t, "base.cue", `values: { name: "svc" }`)
		b := writeTempCUE(t, "prod.cue", `values: { port: 9090 }`)
		got, err := ValidateValues(ctx, config, []string{a, b})
		require.NoError(t, err)

		name, err2 := got.LookupPath(cue.ParsePath("name")).String()
		require.NoError(t, err2)
		assert.Equal(t, "svc", name)

		port, err3 := got.LookupPath(cue.ParsePath("port")).Int64()
		require.NoError(t, err3)
		assert.Equal(t, int64(9090), port)
	})

	t.Run("cross-file type conflict reported as ValuesValidationError with positions", func(t *testing.T) {
		ctx := cuecontext.New()
		config := cue.Value{} // skip schema validation; conflict is the focus

		a := writeTempCUE(t, "base.cue", `values: { size: "15Gi" }`)
		b := writeTempCUE(t, "prod.cue", `values: { size: 1 }`)
		_, err := ValidateValues(ctx, config, []string{a, b})
		require.Error(t, err)

		var valErr *opmerrors.ValuesValidationError
		require.ErrorAs(t, err, &valErr)
		require.NotEmpty(t, valErr.Errors)

		// At least one error should point to one of the two files.
		files := make(map[string]bool)
		for _, fe := range valErr.Errors {
			files[fe.File] = true
		}
		assert.True(t, files["base.cue"] || files["prod.cue"],
			"conflict errors must point to one of the source files, got: %v", files)
	})

	t.Run("per-file errors show correct file names", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string, port: int }`, cue.Filename("schema.cue"))
		config := schema.LookupPath(cue.ParsePath("#config"))

		// Both files have errors — errors from each file should be attributed correctly.
		a := writeTempCUE(t, "base.cue", `values: { name: "ok", bad_a: 1 }`)
		b := writeTempCUE(t, "prod.cue", `values: { port: 9090, bad_b: 2 }`)
		_, err := ValidateValues(ctx, config, []string{a, b})
		require.Error(t, err)

		var valErr *opmerrors.ValuesValidationError
		require.ErrorAs(t, err, &valErr)
		paths := collectPaths(valErr)
		assert.Contains(t, paths, "values.bad_a")
		assert.Contains(t, paths, "values.bad_b")
	})

	t.Run("file without values field returns error", func(t *testing.T) {
		ctx := cuecontext.New()
		path := writeTempCUE(t, "bad.cue", `name: "missing-wrapper"`)
		_, err := ValidateValues(ctx, cue.Value{}, []string{path})
		require.Error(t, err)

		var valErr *opmerrors.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Message, "values:")
	})
}
