package release_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/token"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	buildmodule "github.com/opmodel/cli/internal/build/module"
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
// TestLoad_StaticMetadata
// ---------------------------------------------------------------------------

func TestLoad_StaticMetadata(t *testing.T) {
	ctx := cuecontext.New()

	modulePath := testModulePath(t, "test-module")
	mod, err := buildmodule.Load(ctx, modulePath, "")
	require.NoError(t, err)

	assert.Equal(t, "test-module", mod.Metadata.Name)
	assert.Equal(t, "default", mod.Metadata.DefaultNamespace)
	assert.Equal(t, "testmodule", mod.PkgName())
}

// ---------------------------------------------------------------------------
// TestLoad_MissingMetadata
// ---------------------------------------------------------------------------

func TestLoad_MissingMetadata(t *testing.T) {
	ctx := cuecontext.New()

	modulePath := testModulePath(t, "no-metadata-module")
	mod, err := buildmodule.Load(ctx, modulePath, "")
	require.NoError(t, err)

	// metadata.name is a computed expression, so AST walk returns empty
	assert.Empty(t, mod.Metadata.Name, "computed metadata.name should return empty")
	assert.Equal(t, "nometadata", mod.PkgName())
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
			name, ns := buildmodule.ExtractMetadataFromAST(tt.files)
			assert.Equal(t, tt.expectedName, name)
			assert.Equal(t, tt.expectedNamespace, ns)
		})
	}
}
