package astpipeline

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Construction → Value
// ---------------------------------------------------------------------------

// Input CUE (built as AST):
//
//	{
//	    name:     "my-app"
//	    replicas: 3
//	    enabled:  true
//	}
func TestAST_StructToValue(t *testing.T) {
	// Build a struct purely from AST helpers, convert to Value, read fields back.
	expr := ast.NewStruct(
		"name", ast.NewString("my-app"),
		"replicas", ast.NewLit(token.INT, "3"),
		"enabled", ast.NewBool(true),
	)

	ctx := cuecontext.New()
	val := ctx.BuildExpr(expr)
	require.NoError(t, val.Err())

	name, err := val.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "my-app", name)

	replicas, err := val.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)

	enabled, err := val.LookupPath(cue.ParsePath("enabled")).Bool()
	require.NoError(t, err)
	assert.True(t, enabled)
}

// Input CUE (built as *ast.File):
//
//	metadata: {
//	    name:    "test"
//	    version: "1.0.0"
//	}
//	replicas: 5
func TestAST_FileToValue(t *testing.T) {
	// Build a complete *ast.File with package + fields, convert to Value.
	file := &ast.File{
		Decls: []ast.Decl{
			&ast.Field{
				Label: ast.NewIdent("metadata"),
				Value: ast.NewStruct(
					"name", ast.NewString("test"),
					"version", ast.NewString("1.0.0"),
				),
			},
			&ast.Field{
				Label: ast.NewIdent("replicas"),
				Value: ast.NewLit(token.INT, "5"),
			},
		},
	}

	ctx := cuecontext.New()
	val := ctx.BuildFile(file)
	require.NoError(t, val.Err())

	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "test", name)

	replicas, err := val.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(5), replicas)
}

// Input CUE (built as AST):
//
//	#config: {
//	    image:    string
//	    replicas: int
//	}
//	output: {
//	    img: #config.image
//	}
//
// Then fills #config with: {image: "nginx:1.25", replicas: 3}
func TestAST_DefinitionsWork(t *testing.T) {
	// Build AST with a #config definition, verify FillPath works.
	file := &ast.File{
		Decls: []ast.Decl{
			// #config: { image: string, replicas: int }
			&ast.Field{
				Label: ast.NewIdent("#config"),
				Value: ast.NewStruct(
					"image", ast.NewIdent("string"),
					"replicas", ast.NewIdent("int"),
				),
			},
			// output: { img: #config.image }
			&ast.Field{
				Label: ast.NewIdent("output"),
				Value: ast.NewStruct(
					"img", &ast.SelectorExpr{
						X:   ast.NewIdent("#config"),
						Sel: ast.NewIdent("image"),
					},
				),
			},
		},
	}

	ctx := cuecontext.New()
	val := ctx.BuildFile(file)
	require.NoError(t, val.Err())

	// Fill #config with concrete values
	filled := val.FillPath(cue.ParsePath("#config"), ctx.CompileString(`{image: "nginx:1.25", replicas: 3}`))
	require.NoError(t, filled.Err())

	// Verify the output resolved through the definition
	img, err := filled.LookupPath(cue.ParsePath("output.img")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:1.25", img)
}

// Input CUE (built as AST):
//
//	a: b: c: d: "deep-value"
func TestAST_NestedStructs(t *testing.T) {
	// Build deeply nested AST, verify path lookup works.
	file := &ast.File{
		Decls: []ast.Decl{
			&ast.Field{
				Label: ast.NewIdent("a"),
				Value: ast.NewStruct(
					"b", ast.NewStruct(
						"c", ast.NewStruct(
							"d", ast.NewString("deep-value"),
						),
					),
				),
			},
		},
	}

	ctx := cuecontext.New()
	val := ctx.BuildFile(file)
	require.NoError(t, val.Err())

	deep, err := val.LookupPath(cue.ParsePath("a.b.c.d")).String()
	require.NoError(t, err)
	assert.Equal(t, "deep-value", deep)
}

// Input CUE (built as AST):
//
//	items:       ["a", "b", "c"]
//	constrained: int & >=1
//	withDefault: *8080 | int
func TestAST_ListsAndExpressions(t *testing.T) {
	// Build AST with lists and binary expressions.
	file := &ast.File{
		Decls: []ast.Decl{
			// items: ["a", "b", "c"]
			&ast.Field{
				Label: ast.NewIdent("items"),
				Value: ast.NewList(
					ast.NewString("a"),
					ast.NewString("b"),
					ast.NewString("c"),
				),
			},
			// constrained: int & >=1
			&ast.Field{
				Label: ast.NewIdent("constrained"),
				Value: &ast.BinaryExpr{
					X:  ast.NewIdent("int"),
					Op: token.AND,
					Y: &ast.UnaryExpr{
						Op: token.GEQ,
						X:  ast.NewLit(token.INT, "1"),
					},
				},
			},
			// withDefault: *8080 | int
			&ast.Field{
				Label: ast.NewIdent("withDefault"),
				Value: &ast.BinaryExpr{
					X: &ast.UnaryExpr{
						Op: token.MUL,
						X:  ast.NewLit(token.INT, "8080"),
					},
					Op: token.OR,
					Y:  ast.NewIdent("int"),
				},
			},
		},
	}

	ctx := cuecontext.New()
	val := ctx.BuildFile(file)
	require.NoError(t, val.Err())

	// Verify the list
	iter, err := val.LookupPath(cue.ParsePath("items")).List()
	require.NoError(t, err)
	var items []string
	for iter.Next() {
		s, err := iter.Value().String()
		require.NoError(t, err)
		items = append(items, s)
	}
	assert.Equal(t, []string{"a", "b", "c"}, items)

	// Verify the default value resolves
	defVal, hasDefault := val.LookupPath(cue.ParsePath("withDefault")).Default()
	require.True(t, hasDefault, "withDefault should have a default value")
	port, err := defVal.Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(8080), port)
}

// Input CUE (built as AST with doc comment):
//
//	// The application name
//	name: "test"
func TestAST_CommentsPreserved(t *testing.T) {
	// Build AST with doc comments, verify they survive round-trip.
	nameField := &ast.Field{
		Label: ast.NewIdent("name"),
		Value: ast.NewString("test"),
	}
	ast.AddComment(nameField, &ast.CommentGroup{
		Doc:  true,
		List: []*ast.Comment{{Text: "// The application name"}},
	})

	file := &ast.File{
		Decls: []ast.Decl{nameField},
	}

	// Format to CUE source
	b, err := format.Node(file)
	require.NoError(t, err)
	src := string(b)

	assert.Contains(t, src, "// The application name")
	assert.Contains(t, src, `name: "test"`)

	// Parse back and verify comment is still there
	parsed, err := parser.ParseFile("", b, parser.ParseComments)
	require.NoError(t, err)

	found := false
	ast.Walk(parsed, func(n ast.Node) bool {
		for _, cg := range ast.Comments(n) {
			for _, c := range cg.List {
				if strings.Contains(c.Text, "application name") {
					found = true
				}
			}
		}
		return true
	}, nil)
	assert.True(t, found, "comment should survive format → parse round-trip")
}

// ---------------------------------------------------------------------------
// Value → AST
// ---------------------------------------------------------------------------

// Input CUE (compiled string):
//
//	{
//	    name:     "test"
//	    replicas: 3
//	}
func TestValue_SyntaxReturnsAST(t *testing.T) {
	ctx := cuecontext.New()
	val := ctx.CompileString(`{
		name: "test"
		replicas: 3
	}`)
	require.NoError(t, val.Err())

	node := val.Syntax()
	require.NotNil(t, node)

	// Should be a usable AST node (StructLit or File)
	switch node.(type) {
	case *ast.StructLit, *ast.File:
		// expected
	default:
		t.Fatalf("unexpected node type: %T", node)
	}
}

// Input CUE (compiled string):
//
//	{
//	    name:   "test"
//	    items:  ["a", "b"]
//	    nested: { x: 1, y: 2 }
//	}
func TestValue_SyntaxFormatsCleanly(t *testing.T) {
	ctx := cuecontext.New()
	val := ctx.CompileString(`{
		name: "test"
		items: ["a", "b"]
		nested: { x: 1, y: 2 }
	}`)
	require.NoError(t, val.Err())

	node := val.Syntax()
	b, err := format.Node(node)
	require.NoError(t, err)

	// Should be valid, parseable CUE
	_, err = parser.ParseFile("", b)
	require.NoError(t, err, "Syntax() output should be parseable: %s", string(b))
}

// Input CUE (compiled string):
//
//	{
//	    name:     "test"
//	    replicas: *3 | int
//	}
//
// Exports with: cue.Final(), cue.Concrete(true) → replicas becomes 3
func TestValue_SyntaxConcrete(t *testing.T) {
	ctx := cuecontext.New()
	val := ctx.CompileString(`{
		name: "test"
		replicas: *3 | int
	}`)
	require.NoError(t, val.Err())

	// With Final + Concrete, defaults should resolve to literals
	node := val.Syntax(cue.Final(), cue.Concrete(true))
	b, err := format.Node(node)
	require.NoError(t, err)

	src := string(b)
	// Should contain the concrete value 3, not the constraint
	assert.Contains(t, src, "3")
	assert.NotContains(t, src, "int")
}

// Input CUE (compiled string):
//
//	{
//	    #config: { image: string }
//	    output:  { img: #config.image }
//	}
//
// Exports without Concrete → #config definition preserved in AST
func TestValue_SyntaxWithDefinitions(t *testing.T) {
	ctx := cuecontext.New()
	val := ctx.CompileString(`{
		#config: { image: string }
		output: { img: #config.image }
	}`)
	require.NoError(t, val.Err())

	// Without Concrete, definitions should be preserved
	node := val.Syntax()
	b, err := format.Node(node)
	require.NoError(t, err)

	src := string(b)
	assert.Contains(t, src, "#config")
}

// ---------------------------------------------------------------------------
// Round-trip
// ---------------------------------------------------------------------------

// Input CUE (built as AST):
//
//	name:  "round-trip"
//	count: 42
//
// Path: AST → BuildFile → Value → Syntax(Final) → format.Node → verify
func TestRoundTrip_ASTToValueToAST(t *testing.T) {
	// Build AST → Value → Syntax() → compare
	original := &ast.File{
		Decls: []ast.Decl{
			&ast.Field{
				Label: ast.NewIdent("name"),
				Value: ast.NewString("round-trip"),
			},
			&ast.Field{
				Label: ast.NewIdent("count"),
				Value: ast.NewLit(token.INT, "42"),
			},
		},
	}

	ctx := cuecontext.New()
	val := ctx.BuildFile(original)
	require.NoError(t, val.Err())

	// Get AST back from value
	node := val.Syntax(cue.Final())

	// Format both to compare
	b, err := format.Node(node)
	require.NoError(t, err)

	src := string(b)
	assert.Contains(t, src, `"round-trip"`)
	assert.Contains(t, src, "42")
}

// Input CUE (compiled string):
//
//	{
//	    name:     "test"
//	    replicas: 3
//	    tags:     ["web", "prod"]
//	}
//
// Path: CompileString → Value → Syntax(Final, Concrete) → BuildFile/BuildExpr → Value2 → compare
func TestRoundTrip_ValueToASTToValue(t *testing.T) {
	ctx := cuecontext.New()

	// Start with a compiled value
	val1 := ctx.CompileString(`{
		name: "test"
		replicas: 3
		tags: ["web", "prod"]
	}`)
	require.NoError(t, val1.Err())

	// Convert to AST
	node := val1.Syntax(cue.Final(), cue.Concrete(true))

	// Build a new Value from the AST
	var val2 cue.Value
	switch n := node.(type) {
	case *ast.File:
		val2 = ctx.BuildFile(n)
	case ast.Expr:
		val2 = ctx.BuildExpr(n)
	default:
		t.Fatalf("unexpected Syntax() return type: %T", node)
	}
	require.NoError(t, val2.Err())

	// Verify fields match
	name1, _ := val1.LookupPath(cue.ParsePath("name")).String()
	name2, _ := val2.LookupPath(cue.ParsePath("name")).String()
	assert.Equal(t, name1, name2)

	rep1, _ := val1.LookupPath(cue.ParsePath("replicas")).Int64()
	rep2, _ := val2.LookupPath(cue.ParsePath("replicas")).Int64()
	assert.Equal(t, rep1, rep2)
}

// Input CUE (built as AST):
//
//	metadata: {
//	    name:    "test"
//	    version: "1.0.0"
//	}
//
// Path: AST → format.Node → parser.ParseFile → format.Node → bytes must match
func TestRoundTrip_FormatParseIdentity(t *testing.T) {
	// Build AST → format → parse → format again → bytes should match
	file := &ast.File{
		Decls: []ast.Decl{
			&ast.Field{
				Label: ast.NewIdent("metadata"),
				Value: ast.NewStruct(
					"name", ast.NewString("test"),
					"version", ast.NewString("1.0.0"),
				),
			},
		},
	}

	b1, err := format.Node(file)
	require.NoError(t, err)

	parsed, err := parser.ParseFile("", b1)
	require.NoError(t, err)

	b2, err := format.Node(parsed)
	require.NoError(t, err)

	assert.Equal(t, string(b1), string(b2), "format should be idempotent after parse")
}
