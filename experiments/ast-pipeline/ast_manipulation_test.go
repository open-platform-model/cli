package astpipeline

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Adding things
// ---------------------------------------------------------------------------

// Input CUE (parsed string):
//
//	{ name: "original" }
//
// → appends: added: "new-field"
func TestManipulate_AddField(t *testing.T) {
	// Parse a CUE file, add a new field, build to Value, verify.
	src := `{
		name: "original"
	}`
	f, err := parser.ParseFile("test.cue", src)
	require.NoError(t, err)

	// Add a new field to the top-level struct
	f.Decls = append(f.Decls, &ast.Field{
		Label: ast.NewIdent("added"),
		Value: ast.NewString("new-field"),
	})

	ctx := cuecontext.New()
	val := ctx.BuildFile(f)
	require.NoError(t, val.Err())

	// Original field still there
	name, err := val.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "original", name)

	// New field exists
	added, err := val.LookupPath(cue.ParsePath("added")).String()
	require.NoError(t, err)
	assert.Equal(t, "new-field", added)
}

// Input CUE (parsed string):
//
//	{ output: #myDef.value }
//
// → prepends: #myDef: { value: "defined" }
func TestManipulate_AddDefinition(t *testing.T) {
	// Add a #definition to an existing AST, verify it constrains.
	src := `{
		output: #myDef.value
	}`
	f, err := parser.ParseFile("test.cue", src)
	require.NoError(t, err)

	// Add #myDef: { value: "defined" }
	f.Decls = append([]ast.Decl{
		&ast.Field{
			Label: ast.NewIdent("#myDef"),
			Value: ast.NewStruct(
				"value", ast.NewString("defined"),
			),
		},
	}, f.Decls...)

	ctx := cuecontext.New()
	val := ctx.BuildFile(f)
	require.NoError(t, val.Err())

	output, err := val.LookupPath(cue.ParsePath("output")).String()
	require.NoError(t, err)
	assert.Equal(t, "defined", output)
}

// Input CUE (loaded from testdata/test-module/):
//
//	#components: {
//	    web:    { metadata: name: "web",    spec: { ... } }
//	    api:    { metadata: name: "api",    spec: { ... } }
//	    worker: { metadata: name: "worker", spec: { ... } }
//	}
//
// → adds via astutil.Apply: cache: { metadata: name: "cache", spec: { container: image: "redis:7" } }
func TestManipulate_AddComponent(t *testing.T) {
	// Load module AST, add a new component to #components, build, verify.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	inst := instances[0]

	// Find the file containing #components and add a new component
	found := false
	for _, file := range inst.Files {
		astutil.Apply(file, func(c astutil.Cursor) bool {
			field, ok := c.Node().(*ast.Field)
			if !ok {
				return true
			}
			ident, ok := field.Label.(*ast.Ident)
			if !ok || ident.Name != "#components" {
				return true
			}
			// Found #components struct — add a new component
			structLit, ok := field.Value.(*ast.StructLit)
			if !ok {
				return true
			}
			structLit.Elts = append(structLit.Elts, &ast.Field{
				Label: ast.NewIdent("cache"),
				Value: ast.NewStruct(
					"metadata", ast.NewStruct(
						"name", ast.NewString("cache"),
					),
					"spec", ast.NewStruct(
						"container", ast.NewStruct(
							"image", ast.NewString("redis:7"),
						),
					),
				),
			})
			found = true
			return false
		}, nil)
	}
	require.True(t, found, "should have found #components in module AST")

	// Build to Value and verify
	ctx := cuecontext.New()
	val := ctx.BuildInstance(inst)
	require.NoError(t, val.Err())

	// Original components still there
	web := val.LookupPath(cue.ParsePath("#components.web"))
	assert.True(t, web.Exists(), "web component should exist")

	// New component exists
	cache := val.LookupPath(cue.ParsePath("#components.cache"))
	assert.True(t, cache.Exists(), "cache component should exist")

	cacheName, err := cache.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "cache", cacheName)
}

// Input CUE (built as AST):
//
//	import "strings"
//
//	upper: strings.ToUpper("hello")
func TestManipulate_AddImport(t *testing.T) {
	// Build AST with an import, use Sanitize, format, verify valid CUE.
	file := &ast.File{
		Decls: []ast.Decl{
			&ast.ImportDecl{
				Specs: []*ast.ImportSpec{
					ast.NewImport(nil, "strings"),
				},
			},
			&ast.Field{
				Label: ast.NewIdent("upper"),
				Value: ast.NewCall(
					&ast.SelectorExpr{
						X:   ast.NewIdent("strings"),
						Sel: ast.NewIdent("ToUpper"),
					},
					ast.NewString("hello"),
				),
			},
		},
	}

	b, err := format.Node(file)
	require.NoError(t, err)

	src := string(b)
	assert.Contains(t, src, `import "strings"`)
	assert.Contains(t, src, "strings.ToUpper")

	// Verify it parses
	_, err = parser.ParseFile("", b)
	require.NoError(t, err)
}

// Input CUE (loaded from testdata/test-module/):
//
//	package testmodule
//	metadata: { name: "test-module", ... }
//	#config: { ... }
//	#components: { ... }
//
// → injects directly into inst.Files[0].Decls: injected: "from-overlay"
func TestManipulate_InjectOverlayDecls(t *testing.T) {
	// Instead of a separate overlay file, inject declarations directly
	// into the module's AST files, then build.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	inst := instances[0]
	require.True(t, len(inst.Files) > 0)

	// Add a new field to the first file's declarations
	inst.Files[0].Decls = append(inst.Files[0].Decls, &ast.Field{
		Label: ast.NewIdent("injected"),
		Value: ast.NewString("from-overlay"),
	})

	ctx := cuecontext.New()
	val := ctx.BuildInstance(inst)
	require.NoError(t, val.Err())

	injected, err := val.LookupPath(cue.ParsePath("injected")).String()
	require.NoError(t, err)
	assert.Equal(t, "from-overlay", injected)
}

// ---------------------------------------------------------------------------
// Modifying things
// ---------------------------------------------------------------------------

// Input CUE (parsed string):
//
//	{
//	    name:     "before"
//	    replicas: 1
//	}
//
// → via astutil.Apply: name changes to "after"
func TestManipulate_ChangeFieldValue(t *testing.T) {
	// Use astutil.Apply to change a field's value.
	src := `{
		name: "before"
		replicas: 1
	}`
	f, err := parser.ParseFile("test.cue", src)
	require.NoError(t, err)

	astutil.Apply(f, func(c astutil.Cursor) bool {
		field, ok := c.Node().(*ast.Field)
		if !ok {
			return true
		}
		if ident, ok := field.Label.(*ast.Ident); ok && ident.Name == "name" {
			c.Replace(&ast.Field{
				Label: ast.NewIdent("name"),
				Value: ast.NewString("after"),
			})
		}
		return true
	}, nil)

	ctx := cuecontext.New()
	val := ctx.BuildFile(f)
	require.NoError(t, val.Err())

	name, err := val.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "after", name)
}

// Input CUE (parsed string):
//
//	{
//	    oldName: "value"
//	    other:   42
//	}
//
// → via astutil.Apply: label "oldName" renamed to "newName"
func TestManipulate_ChangeLabel(t *testing.T) {
	// Rename a field label via astutil.Apply.
	src := `{
		oldName: "value"
		other: 42
	}`
	f, err := parser.ParseFile("test.cue", src)
	require.NoError(t, err)

	astutil.Apply(f, func(c astutil.Cursor) bool {
		field, ok := c.Node().(*ast.Field)
		if !ok {
			return true
		}
		if ident, ok := field.Label.(*ast.Ident); ok && ident.Name == "oldName" {
			c.Replace(&ast.Field{
				Label: ast.NewIdent("newName"),
				Value: field.Value,
			})
		}
		return true
	}, nil)

	ctx := cuecontext.New()
	val := ctx.BuildFile(f)
	require.NoError(t, val.Err())

	// Old name should be gone
	assert.False(t, val.LookupPath(cue.ParsePath("oldName")).Exists())

	// New name should exist
	newName, err := val.LookupPath(cue.ParsePath("newName")).String()
	require.NoError(t, err)
	assert.Equal(t, "value", newName)
}

// Input CUE (parsed string):
//
//	{
//	    component: spec: {
//	        image:    "old:1.0"
//	        replicas: 1
//	    }
//	}
//
// → via astutil.Apply: entire spec struct replaced with:
//
//	{ image: "new:2.0", replicas: 5, newField: true }
func TestManipulate_ReplaceStruct(t *testing.T) {
	// Replace an entire struct value.
	src := `{
		component: {
			spec: {
				image: "old:1.0"
				replicas: 1
			}
		}
	}`
	f, err := parser.ParseFile("test.cue", src)
	require.NoError(t, err)

	astutil.Apply(f, func(c astutil.Cursor) bool {
		field, ok := c.Node().(*ast.Field)
		if !ok {
			return true
		}
		if ident, ok := field.Label.(*ast.Ident); ok && ident.Name == "spec" {
			c.Replace(&ast.Field{
				Label: ast.NewIdent("spec"),
				Value: ast.NewStruct(
					"image", ast.NewString("new:2.0"),
					"replicas", ast.NewLit(token.INT, "5"),
					"newField", ast.NewBool(true),
				),
			})
		}
		return true
	}, nil)

	ctx := cuecontext.New()
	val := ctx.BuildFile(f)
	require.NoError(t, val.Err())

	img, err := val.LookupPath(cue.ParsePath("component.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "new:2.0", img)

	nf, err := val.LookupPath(cue.ParsePath("component.spec.newField")).Bool()
	require.NoError(t, err)
	assert.True(t, nf)
}

// ---------------------------------------------------------------------------
// Removing things
// ---------------------------------------------------------------------------

// Input CUE (parsed string):
//
//	{
//	    keep:     "yes"
//	    remove:   "no"
//	    alsoKeep: 42
//	}
//
// → via astutil.Apply + cursor.Delete(): removes "remove" field
func TestManipulate_DeleteField(t *testing.T) {
	// Use cursor.Delete() to remove a field.
	src := `{
		keep: "yes"
		remove: "no"
		alsoKeep: 42
	}`
	f, err := parser.ParseFile("test.cue", src)
	require.NoError(t, err)

	astutil.Apply(f, func(c astutil.Cursor) bool {
		field, ok := c.Node().(*ast.Field)
		if !ok {
			return true
		}
		if ident, ok := field.Label.(*ast.Ident); ok && ident.Name == "remove" {
			c.Delete()
		}
		return true
	}, nil)

	ctx := cuecontext.New()
	val := ctx.BuildFile(f)
	require.NoError(t, val.Err())

	assert.True(t, val.LookupPath(cue.ParsePath("keep")).Exists())
	assert.False(t, val.LookupPath(cue.ParsePath("remove")).Exists())
	assert.True(t, val.LookupPath(cue.ParsePath("alsoKeep")).Exists())
}

// Input CUE (parsed string):
//
//	{
//	    #components: {
//	        web:    { name: "web" }
//	        api:    { name: "api" }
//	        worker: { name: "worker" }
//	    }
//	}
//
// → via astutil.Apply + cursor.Delete(): removes "api" component
func TestManipulate_DeleteComponent(t *testing.T) {
	// Remove a component from a #components struct.
	src := `{
		#components: {
			web: { name: "web" }
			api: { name: "api" }
			worker: { name: "worker" }
		}
	}`
	f, err := parser.ParseFile("test.cue", src)
	require.NoError(t, err)

	// Delete the "api" component
	inComponents := false
	astutil.Apply(f, func(c astutil.Cursor) bool {
		field, ok := c.Node().(*ast.Field)
		if !ok {
			return true
		}
		ident, ok := field.Label.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "#components" {
			inComponents = true
			return true
		}
		if inComponents && ident.Name == "api" {
			c.Delete()
			inComponents = false
		}
		return true
	}, nil)

	ctx := cuecontext.New()
	val := ctx.BuildFile(f)
	require.NoError(t, val.Err())

	assert.True(t, val.LookupPath(cue.ParsePath("#components.web")).Exists())
	assert.False(t, val.LookupPath(cue.ParsePath("#components.api")).Exists())
	assert.True(t, val.LookupPath(cue.ParsePath("#components.worker")).Exists())
}

// ---------------------------------------------------------------------------
// Composing things
// ---------------------------------------------------------------------------

// Input CUE (two parsed files merged):
//
//	// file 1:
//	{ name: "from-file-1" }
//
//	// file 2:
//	{ replicas: 3 }
//
// → f1.Decls = append(f1.Decls, f2.Decls...) → unified value
func TestManipulate_MergeTwoFiles(t *testing.T) {
	// Combine declarations from two files into one, build, verify unified result.
	src1 := `{
		name: "from-file-1"
	}`
	src2 := `{
		replicas: 3
	}`

	f1, err := parser.ParseFile("f1.cue", src1)
	require.NoError(t, err)
	f2, err := parser.ParseFile("f2.cue", src2)
	require.NoError(t, err)

	// Merge all declarations into f1
	f1.Decls = append(f1.Decls, f2.Decls...)

	ctx := cuecontext.New()
	val := ctx.BuildFile(f1)
	require.NoError(t, val.Err())

	name, err := val.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "from-file-1", name)

	replicas, err := val.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)
}

// Input CUE: testdata/test-module/ + overlay file (built as AST):
//
//	package testmodule
//
//	#overlayMeta: {
//	    releaseName: "my-release"
//	    namespace:   "production"
//	}
//
// Overlay injected via load.Config.Overlay → both module and overlay fields accessible
func TestManipulate_OverlayAsASTFile(t *testing.T) {
	// Build overlay as *ast.File, format it, inject via load.Config.Overlay.
	// Verify it loads correctly with the module.
	modulePath := testModulePath(t)

	// Build overlay as AST
	overlayFile := &ast.File{
		Decls: []ast.Decl{
			&ast.Package{Name: ast.NewIdent("testmodule")},
			&ast.Field{
				Label: ast.NewIdent("#overlayMeta"),
				Value: ast.NewStruct(
					"releaseName", ast.NewString("my-release"),
					"namespace", ast.NewString("production"),
				),
			},
		},
	}

	overlayBytes, err := format.Node(overlayFile)
	require.NoError(t, err)

	// Load module with AST-generated overlay
	overlayPath := modulePath + "/overlay_gen.cue"
	cfg := &load.Config{
		Dir: modulePath,
		Overlay: map[string]load.Source{
			overlayPath: load.FromBytes(overlayBytes),
		},
	}

	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	ctx := cuecontext.New()
	val := ctx.BuildInstance(instances[0])
	require.NoError(t, val.Err())

	// Verify overlay fields are accessible
	relName, err := val.LookupPath(cue.ParsePath("#overlayMeta.releaseName")).String()
	require.NoError(t, err)
	assert.Equal(t, "my-release", relName)

	// Verify original module fields still work
	moduleName, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "test-module", moduleName)
}
