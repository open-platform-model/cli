package modulerelease

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/core/component"
	"github.com/opmodel/cli/internal/core/module"
	opmerrors "github.com/opmodel/cli/internal/errors"
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

// cueErrorDetails formats a CUE error into a string containing paths and messages.
// Used in tests instead of release.formatCUEDetails (which adds lipgloss styling).
func cueErrorDetails(err error) string {
	return cueerrors.Details(err, nil)
}

// cueErrorPaths returns all error paths from a CUE error as a joined string.
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

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)

		details := cueErrorDetails(err)
		assert.Contains(t, details, "field not allowed")
		paths := cueErrorPaths(err)
		assert.Contains(t, paths, "values.extra")
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

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		assert.NoError(t, err)
	})

	t.Run("catches single disallowed field", func(t *testing.T) {
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

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)

		errs := cueerrors.Errors(err)
		require.Len(t, errs, 1)
		assert.Equal(t, []string{"values", "extra"}, errs[0].Path())
		assert.Contains(t, errs[0].Error(), "field not allowed")
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

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		assert.NoError(t, err)
	})

	t.Run("closed empty config rejects extra fields", func(t *testing.T) {
		ctx := cuecontext.New()

		schema := ctx.CompileString(`
#config: {}
`, cue.Filename("schema.cue"))

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

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)
		paths := cueErrorPaths(err)
		assert.Contains(t, paths, "values.db")
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

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)

		paths := cueErrorPaths(err)
		assert.Contains(t, paths, "values.x")
		assert.Contains(t, paths, "values.y")
		assert.Contains(t, paths, "values.z")
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

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		assert.NoError(t, err, "pattern-matched fields should be accepted")
	})

	t.Run("disallowed field inside pattern struct", func(t *testing.T) {
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

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)
		paths := cueErrorPaths(err)
		assert.Contains(t, paths, "values.media.tvshows.badField")
	})

	t.Run("split values across files validates correctly", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string, port: int }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		a := ctx.CompileString(`{name: "test"}`, cue.Filename("base.cue"))
		b := ctx.CompileString(`{port: 8080}`, cue.Filename("env.cue"))
		unified := a.Unify(b)

		err := validateFieldsRecursive(configDef, unified, []string{"values"}, nil)
		assert.NoError(t, err, "split values that together satisfy schema should pass")
	})

	t.Run("open struct allows arbitrary fields", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string, ... }`, cue.Filename("schema.cue"))
		configDef := schema.LookupPath(cue.ParsePath("#config"))

		vals := ctx.CompileString(`{
			name: "ok"
			anything: "goes"
		}`, cue.Filename("values.cue"))

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		assert.NoError(t, err, "open struct should allow extra fields")
	})

	t.Run("deeply nested path on disallowed field", func(t *testing.T) {
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

		err := validateFieldsRecursive(configDef, vals, []string{"values"}, nil)
		require.Error(t, err)
		paths := cueErrorPaths(err)
		assert.Contains(t, paths, "values.level1.level2.level3.bad")
	})
}

func TestValidateValuesAgainstConfig_ValidateValues(t *testing.T) {
	// Tests ValidateValues() receiver method on ModuleRelease.
	t.Run("returns nil when config or values not present", func(t *testing.T) {
		rel := &ModuleRelease{}
		assert.NoError(t, rel.ValidateValues())
	})

	t.Run("valid values return nil", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string }`, cue.Filename("schema.cue"))
		configVal := schema.LookupPath(cue.ParsePath("#config"))
		valuesVal := ctx.CompileString(`{ name: "ok" }`, cue.Filename("values.cue"))

		rel := &ModuleRelease{
			Module: module.Module{Config: configVal},
			Values: valuesVal,
		}
		assert.NoError(t, rel.ValidateValues())
	})

	t.Run("invalid values return ValidationError", func(t *testing.T) {
		ctx := cuecontext.New()
		schema := ctx.CompileString(`#config: { name: string }`, cue.Filename("schema.cue"))
		configVal := schema.LookupPath(cue.ParsePath("#config"))
		valuesVal := ctx.CompileString(`{ name: "ok", extra: "bad" }`, cue.Filename("values.cue"))

		rel := &ModuleRelease{
			Module: module.Module{Config: configVal},
			Values: valuesVal,
		}
		err := rel.ValidateValues()
		require.Error(t, err)

		var valErr *opmerrors.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Message, "#config schema")
	})
}

func TestValidate_ModuleRelease(t *testing.T) {
	// Tests Validate() receiver method on ModuleRelease.
	t.Run("empty components map passes", func(t *testing.T) {
		rel := &ModuleRelease{Components: map[string]*component.Component{}}
		assert.NoError(t, rel.Validate())
	})

	t.Run("nil components map passes", func(t *testing.T) {
		rel := &ModuleRelease{}
		assert.NoError(t, rel.Validate())
	})

	t.Run("concrete component passes", func(t *testing.T) {
		ctx := cuecontext.New()
		concVal := ctx.CompileString(`{ x: 42 }`)
		require.NoError(t, concVal.Err())

		rel := &ModuleRelease{
			Components: map[string]*component.Component{
				"web": {Value: concVal},
			},
		}
		assert.NoError(t, rel.Validate())
	})

	t.Run("non-concrete component fails", func(t *testing.T) {
		ctx := cuecontext.New()
		openVal := ctx.CompileString(`{ x: int }`)
		require.NoError(t, openVal.Err())

		rel := &ModuleRelease{
			Components: map[string]*component.Component{
				"web": {Value: openVal},
			},
		}
		err := rel.Validate()
		require.Error(t, err)

		var valErr *opmerrors.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Message, "non-concrete values")
	})
}
