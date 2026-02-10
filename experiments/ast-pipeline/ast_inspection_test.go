package astpipeline

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Input CUE (loaded from testdata/test-module/module.cue):
//
//	package testmodule
//	metadata:    { name: "test-module", version: "1.0.0", ... }
//	#config:     { image: string, replicas: int & >=1, ... }
//	values:      { image: "nginx:1.25", replicas: 2, ... }
//	#components: { web: { ... }, api: { ... }, worker: { ... } }
//
// Inspects: top-level field names from file.Decls
func TestInspect_WalkFindAllFields(t *testing.T) {
	// Walk over the test module, collect all top-level field names.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	topLevelFields := make(map[string]bool)
	for _, file := range instances[0].Files {
		for _, decl := range file.Decls {
			if field, ok := decl.(*ast.Field); ok {
				if ident, ok := field.Label.(*ast.Ident); ok {
					topLevelFields[ident.Name] = true
				}
			}
		}
	}

	// Should find these top-level fields across files
	assert.True(t, topLevelFields["metadata"], "should find metadata")
	assert.True(t, topLevelFields["#config"], "should find #config")
	assert.True(t, topLevelFields["values"], "should find values")
	assert.True(t, topLevelFields["#components"], "should find #components")
}

// Input CUE (loaded from testdata/test-module/):
//
//	#config:     { ... }
//	#components: {
//	    web: {
//	        #resources: { ... }
//	        #traits:    { ... }
//	    }
//	    ...
//	}
//
// Inspects: all fields with labels starting with "#" (recursive walk)
func TestInspect_FindDefinitions(t *testing.T) {
	// Walk AST, find all #definition fields (names starting with #).
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	var definitions []string
	for _, file := range instances[0].Files {
		ast.Walk(file, func(n ast.Node) bool {
			field, ok := n.(*ast.Field)
			if !ok {
				return true
			}
			ident, ok := field.Label.(*ast.Ident)
			if !ok {
				return true
			}
			if strings.HasPrefix(ident.Name, "#") {
				definitions = append(definitions, ident.Name)
			}
			return true
		}, nil)
	}

	assert.Contains(t, definitions, "#config")
	assert.Contains(t, definitions, "#components")
	// Should also find nested #resources and #traits
	assert.Contains(t, definitions, "#resources")
	assert.Contains(t, definitions, "#traits")
}

// Input CUE (loaded from testdata/test-module/):
//
//	package testmodule
//	// (no import declarations)
//
// Inspects: file.ImportSpecs() → expects empty (test module has no imports)
func TestInspect_FindImports(t *testing.T) {
	// Use file.ImportSpecs() to list imports from AST.
	// Our test module has no imports, so verify the API works and returns empty.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	var imports []string
	for _, file := range instances[0].Files {
		for spec := range file.ImportSpecs() {
			imports = append(imports, spec.Path.Value)
		}
	}

	// Test module has no imports
	assert.Empty(t, imports, "test module should have no imports")
}

// Input CUE (loaded from testdata/test-module/):
//
//	package testmodule
//
// Inspects: package name via inst.PkgName and via *ast.Package declaration
func TestInspect_ExtractPackageName(t *testing.T) {
	// Get package name from AST without building Value.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	// Method 1: From instance directly
	assert.Equal(t, "testmodule", instances[0].PkgName)

	// Method 2: From AST (inspect Package declaration)
	var pkgName string
	for _, file := range instances[0].Files {
		for _, decl := range file.Decls {
			if pkg, ok := decl.(*ast.Package); ok {
				pkgName = pkg.Name.Name
				break
			}
		}
		if pkgName != "" {
			break
		}
	}
	assert.Equal(t, "testmodule", pkgName)
}

// Input CUE (loaded from testdata/test-module/):
//
//	metadata: {
//	    name:    "test-module"
//	    version: "1.0.0"
//	    ...
//	}
//
// Inspects: walks AST → Field("metadata") → StructLit → Field("name") → BasicLit
// Then compares with Value.LookupPath("metadata.name") to verify they match
func TestInspect_ExtractMetadataField(t *testing.T) {
	// Walk AST to find metadata.name as a string literal — no evaluation needed.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	// Find the metadata field, then look for name inside it
	var metadataName string
	for _, file := range instances[0].Files {
		for _, decl := range file.Decls {
			field, ok := decl.(*ast.Field)
			if !ok {
				continue
			}
			ident, ok := field.Label.(*ast.Ident)
			if !ok || ident.Name != "metadata" {
				continue
			}
			// Found metadata — look for name field inside
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
				if !ok || innerIdent.Name != "name" {
					continue
				}
				lit, ok := innerField.Value.(*ast.BasicLit)
				if ok && lit.Kind == token.STRING {
					// Remove quotes
					metadataName = strings.Trim(lit.Value, `"`)
				}
			}
		}
	}

	assert.Equal(t, "test-module", metadataName)

	// Verify it matches what Value.LookupPath would return
	ctx := cuecontext.New()
	val := ctx.BuildInstance(instances[0])
	require.NoError(t, val.Err())

	valueName, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, metadataName, valueName, "AST-extracted name should match Value-extracted name")
}

// Input CUE (loaded from testdata/test-module/module.cue):
//
//	// Module metadata
//	metadata: { ... }
//	// Configuration schema
//	#config: { ... }
//	// Component definitions
//	#components: { ... }
//
// Inspects: all comment text via ast.Comments(node) in recursive walk
func TestInspect_FindComments(t *testing.T) {
	// Walk AST, extract all doc comments.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	var comments []string
	for _, file := range instances[0].Files {
		ast.Walk(file, func(n ast.Node) bool {
			for _, cg := range ast.Comments(n) {
				for _, c := range cg.List {
					comments = append(comments, c.Text)
				}
			}
			return true
		}, nil)
	}

	// Should find our comments from module.cue
	found := false
	for _, c := range comments {
		if strings.Contains(c, "Module metadata") || strings.Contains(c, "Configuration schema") {
			found = true
			break
		}
	}
	assert.True(t, found, "should find doc comments from module source. Found: %v", comments)
}

// Input CUE (loaded from testdata/test-module/):
//
//	#config:     { image: string, replicas: int & >=1, ... }
//	values:      { image: "nginx:1.25", ... }
//	#components: { web: { ... }, api: { ... }, worker: { ... } }
//
// Inspects: checks for presence of #config, values, and #components top-level fields
// to detect the module's architectural pattern
func TestInspect_IdentifyConfigPattern(t *testing.T) {
	// Detect whether a module uses #config / values pattern by inspecting AST.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	hasConfig := false
	hasValues := false
	hasComponents := false

	for _, file := range instances[0].Files {
		for _, decl := range file.Decls {
			field, ok := decl.(*ast.Field)
			if !ok {
				continue
			}
			ident, ok := field.Label.(*ast.Ident)
			if !ok {
				continue
			}
			switch ident.Name {
			case "#config":
				hasConfig = true
			case "values":
				hasValues = true
			case "#components":
				hasComponents = true
			}
		}
	}

	assert.True(t, hasConfig, "module should have #config definition")
	assert.True(t, hasValues, "module should have values field")
	assert.True(t, hasComponents, "module should have #components definition")
}
