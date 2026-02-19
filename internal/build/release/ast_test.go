package release

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/build/module"
)

// testModulePath returns the absolute path to a test module fixture.
// Points to the testdata directory in the parent build package.
func testModulePath(t *testing.T, name string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	// Go up one level from release/ to build/ then into testdata/
	return filepath.Join(filepath.Dir(filename), "..", "testdata", name)
}

// ---------------------------------------------------------------------------
// TestGenerateOverlayAST_ProducesValidCUE
// ---------------------------------------------------------------------------

func TestGenerateOverlayAST_ProducesValidCUE(t *testing.T) {
	overlay := generateOverlayAST("testmodule", Options{
		Name:      "my-release",
		Namespace: "production",
	})

	b, err := format.Node(overlay)
	require.NoError(t, err, "format.Node should succeed")

	// Should parse back without errors
	_, err = parser.ParseFile("overlay.cue", b, parser.ParseComments)
	require.NoError(t, err, "AST-generated overlay should produce parseable CUE")
}

// ---------------------------------------------------------------------------
// TestGenerateOverlayAST_ContainsRequiredFields
// ---------------------------------------------------------------------------

func TestGenerateOverlayAST_ContainsRequiredFields(t *testing.T) {
	overlay := generateOverlayAST("testmodule", Options{
		Name:      "my-release",
		Namespace: "production",
	})

	b, err := format.Node(overlay)
	require.NoError(t, err)

	file, err := parser.ParseFile("overlay.cue", b)
	require.NoError(t, err)

	// Find #opmReleaseMeta and collect its field names
	var fieldNames []string
	for _, decl := range file.Decls {
		field, ok := decl.(*ast.Field)
		if !ok {
			continue
		}
		ident, ok := field.Label.(*ast.Ident)
		if !ok || ident.Name != "#opmReleaseMeta" {
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
			switch label := innerField.Label.(type) {
			case *ast.Ident:
				fieldNames = append(fieldNames, label.Name)
			case *ast.BasicLit:
				fieldNames = append(fieldNames, strings.Trim(label.Value, `"`))
			}
		}
	}

	assert.Contains(t, fieldNames, "name")
	assert.Contains(t, fieldNames, "namespace")
	assert.Contains(t, fieldNames, "fqn")
	assert.Contains(t, fieldNames, "version")
	assert.Contains(t, fieldNames, "identity")
	assert.Contains(t, fieldNames, "labels")
}

// ---------------------------------------------------------------------------
// TestInspectModule_StaticMetadata
// ---------------------------------------------------------------------------

func TestInspectModule_StaticMetadata(t *testing.T) {
	ctx := cuecontext.New()
	builder := NewBuilder(ctx, "")

	modulePath := testModulePath(t, "test-module")
	inspection, err := builder.InspectModule(modulePath)
	require.NoError(t, err)

	assert.Equal(t, "test-module", inspection.Name)
	assert.Equal(t, "default", inspection.DefaultNamespace)
	assert.Equal(t, "testmodule", inspection.PkgName)
}

// ---------------------------------------------------------------------------
// TestInspectModule_MissingMetadata
// ---------------------------------------------------------------------------

func TestInspectModule_MissingMetadata(t *testing.T) {
	ctx := cuecontext.New()
	builder := NewBuilder(ctx, "")

	modulePath := testModulePath(t, "no-metadata-module")
	inspection, err := builder.InspectModule(modulePath)
	require.NoError(t, err)

	// metadata.name is a computed expression, so AST walk returns empty
	assert.Empty(t, inspection.Name, "computed metadata.name should return empty")
	assert.Equal(t, "nometadata", inspection.PkgName)
}

// ---------------------------------------------------------------------------
// TestExtractMetadataFromAST
// ---------------------------------------------------------------------------

func TestExtractMetadataFromAST(t *testing.T) {
	tests := []struct {
		name              string
		files             []*ast.File
		expectedName      string
		expectedNamespace string
	}{
		{
			name: "static string literals",
			files: []*ast.File{
				{
					Decls: []ast.Decl{
						&ast.Field{
							Label: ast.NewIdent("metadata"),
							Value: ast.NewStruct(
								&ast.Field{Label: ast.NewIdent("name"), Value: ast.NewString("my-module")},
								&ast.Field{Label: ast.NewIdent("defaultNamespace"), Value: ast.NewString("production")},
							),
						},
					},
				},
			},
			expectedName:      "my-module",
			expectedNamespace: "production",
		},
		{
			name: "missing fields",
			files: []*ast.File{
				{
					Decls: []ast.Decl{
						&ast.Field{
							Label: ast.NewIdent("metadata"),
							Value: ast.NewStruct(
								&ast.Field{Label: ast.NewIdent("version"), Value: ast.NewString("1.0.0")},
							),
						},
					},
				},
			},
			expectedName:      "",
			expectedNamespace: "",
		},
		{
			name: "non-string expression for name",
			files: []*ast.File{
				{
					Decls: []ast.Decl{
						&ast.Field{
							Label: ast.NewIdent("metadata"),
							Value: ast.NewStruct(
								&ast.Field{
									Label: ast.NewIdent("name"),
									Value: &ast.BinaryExpr{
										X:  ast.NewString("prefix-"),
										Op: token.ADD,
										Y:  ast.NewIdent("_suffix"),
									},
								},
								&ast.Field{Label: ast.NewIdent("defaultNamespace"), Value: ast.NewString("default")},
							),
						},
					},
				},
			},
			expectedName:      "",
			expectedNamespace: "default",
		},
		{
			name:              "no metadata field at all",
			files:             []*ast.File{{Decls: []ast.Decl{}}},
			expectedName:      "",
			expectedNamespace: "",
		},
		{
			name: "metadata in second file",
			files: []*ast.File{
				{Decls: []ast.Decl{
					&ast.Field{Label: ast.NewIdent("values"), Value: ast.NewStruct()},
				}},
				{Decls: []ast.Decl{
					&ast.Field{
						Label: ast.NewIdent("metadata"),
						Value: ast.NewStruct(
							&ast.Field{Label: ast.NewIdent("name"), Value: ast.NewString("second-file-module")},
						),
					},
				}},
			},
			expectedName:      "second-file-module",
			expectedNamespace: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, ns := module.ExtractMetadataFromAST(tt.files)
			assert.Equal(t, tt.expectedName, name)
			assert.Equal(t, tt.expectedNamespace, ns)
		})
	}
}
