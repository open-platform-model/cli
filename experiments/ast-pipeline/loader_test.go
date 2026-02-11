package astpipeline

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Hypothesis 2: Single-load with AST inspection
// ---------------------------------------------------------------------------

// Input CUE (loaded from testdata/test-module/):
//
//	package testmodule
//
// Inspects: inst.PkgName → "testmodule" (available without evaluation)
func TestSingleLoad_PackageNameFromInstance(t *testing.T) {
	// inst.PkgName should be available without a separate detectPackageName call.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	assert.Equal(t, "testmodule", instances[0].PkgName)
}

// Input CUE (loaded from testdata/test-module/):
//
//	module.cue  → package testmodule; metadata: {...}; #config: {...}; ...
//	values.cue  → package testmodule; values: {...}
//
// Inspects: inst.Files contains []*ast.File, each with filename and declarations
func TestSingleLoad_FilesAvailable(t *testing.T) {
	// inst.Files should contain []*ast.File with all source files.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	inst := instances[0]
	require.True(t, len(inst.Files) > 0, "should have at least one file")

	// Each file should have a filename and be a valid AST
	for _, file := range inst.Files {
		assert.NotEmpty(t, file.Filename, "file should have a filename")
		assert.True(t, len(file.Decls) > 0, "file %s should have declarations", file.Filename)
		t.Logf("File: %s (%d declarations)", file.Filename, len(file.Decls))
	}
}

// Single-load workflow:
//
//	Step 1: load.Instances → inst.PkgName = "testmodule"
//	Step 2: build overlay AST using discovered package name:
//	    package testmodule
//	    #singleLoadMeta: {
//	        releaseName: "test-release"
//	        namespace:   "staging"
//	        moduleName:  metadata.name   // cross-file reference
//	    }
//	Step 3: load again with overlay → build Value
//
// Verifies: #singleLoadMeta.moduleName resolves to "test-module" (from module metadata)
func TestSingleLoad_InspectThenBuild(t *testing.T) {
	// Single-load workflow:
	// 1. Load once → get inst with Files and PkgName
	// 2. Inspect AST for package name (no second load needed)
	// 3. Build overlay AST using the package name
	// 4. Load again with overlay → build Value
	// Verify we get the same result as current double-load approach.

	modulePath := testModulePath(t)

	// Step 1: Single load
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	inst := instances[0]
	pkgName := inst.PkgName
	assert.Equal(t, "testmodule", pkgName)

	// Step 2: Build overlay using discovered package name
	overlay := &ast.File{
		Decls: []ast.Decl{
			&ast.Package{Name: ast.NewIdent(pkgName)},
			&ast.Field{
				Label: ast.NewIdent("#singleLoadMeta"),
				Value: ast.NewStruct(
					"releaseName", ast.NewString("test-release"),
					"namespace", ast.NewString("staging"),
					// Reference module metadata (resolves because same package)
					"moduleName", &ast.SelectorExpr{
						X:   ast.NewIdent("metadata"),
						Sel: ast.NewIdent("name"),
					},
				),
			},
		},
	}

	overlayBytes, err := format.Node(overlay)
	require.NoError(t, err)

	// Step 3: Load again with overlay
	overlayPath := modulePath + "/overlay_gen.cue"
	cfgWithOverlay := &load.Config{
		Dir: modulePath,
		Overlay: map[string]load.Source{
			overlayPath: load.FromBytes(overlayBytes),
		},
	}

	instances2 := load.Instances([]string{"."}, cfgWithOverlay)
	require.Len(t, instances2, 1)
	require.NoError(t, instances2[0].Err)

	ctx := cuecontext.New()
	val := ctx.BuildInstance(instances2[0])
	require.NoError(t, val.Err())

	// Verify overlay fields
	relName, err := val.LookupPath(cue.ParsePath("#singleLoadMeta.releaseName")).String()
	require.NoError(t, err)
	assert.Equal(t, "test-release", relName)

	// Verify metadata reference resolved
	modName, err := val.LookupPath(cue.ParsePath("#singleLoadMeta.moduleName")).String()
	require.NoError(t, err)
	assert.Equal(t, "test-module", modName)
}

// Input CUE (loaded from testdata/test-module/):
//
//	metadata: {
//	    name:    "test-module"
//	    version: "1.0.0"
//	    ...
//	}
//
// Extracts name and version two ways:
//
//  1. AST walk: Field("metadata") → StructLit → Field("name"/"version") → BasicLit
//  2. Value:    val.LookupPath(cue.ParsePath("metadata.name")).String()
//
// Verifies both methods return identical results
func TestSingleLoad_ASTInspectVsValueLookup(t *testing.T) {
	// Compare metadata extracted from AST walk vs Value.LookupPath.
	// Both should find the same name and version.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	inst := instances[0]

	// AST extraction
	var astName, astVersion string
	for _, file := range inst.Files {
		for _, decl := range file.Decls {
			field, ok := decl.(*ast.Field)
			if !ok {
				continue
			}
			ident, ok := field.Label.(*ast.Ident)
			if !ok || ident.Name != "metadata" {
				continue
			}
			structLit, ok := field.Value.(*ast.StructLit)
			if !ok {
				continue
			}
			for _, elt := range structLit.Elts {
				innerField, ok := elt.(*ast.Field)
				if !ok {
					continue
				}
				innerIdent, ok := innerField.Label.(*ast.Ident)
				if !ok {
					continue
				}
				if lit, ok := innerField.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					val := strings.Trim(lit.Value, `"`)
					switch innerIdent.Name {
					case "name":
						astName = val
					case "version":
						astVersion = val
					}
				}
			}
		}
	}

	// Value extraction
	ctx := cuecontext.New()
	val := ctx.BuildInstance(inst)
	require.NoError(t, val.Err())

	valName, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	valVersion, err := val.LookupPath(cue.ParsePath("metadata.version")).String()
	require.NoError(t, err)

	// Compare
	assert.Equal(t, valName, astName, "AST and Value should agree on metadata.name")
	assert.Equal(t, valVersion, astVersion, "AST and Value should agree on metadata.version")

	t.Logf("metadata.name: AST=%q Value=%q", astName, valName)
	t.Logf("metadata.version: AST=%q Value=%q", astVersion, valVersion)
}
