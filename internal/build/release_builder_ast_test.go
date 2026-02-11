package build

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testModulePath returns the absolute path to a test module fixture.
func testModulePath(t *testing.T, name string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

// ---------------------------------------------------------------------------
// 4.1: TestGenerateOverlayAST_ProducesValidCUE
// ---------------------------------------------------------------------------

func TestGenerateOverlayAST_ProducesValidCUE(t *testing.T) {
	ctx := cuecontext.New()
	builder := NewReleaseBuilder(ctx, "")

	overlay := builder.generateOverlayAST("testmodule", ReleaseOptions{
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
// 4.2: TestGenerateOverlayAST_ContainsRequiredFields
// ---------------------------------------------------------------------------

func TestGenerateOverlayAST_ContainsRequiredFields(t *testing.T) {
	ctx := cuecontext.New()
	builder := NewReleaseBuilder(ctx, "")

	overlay := builder.generateOverlayAST("testmodule", ReleaseOptions{
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
// 4.3: TestGenerateOverlayAST_MatchesStringTemplate
// ---------------------------------------------------------------------------

func TestGenerateOverlayAST_MatchesStringTemplate(t *testing.T) {
	// Generate both AST and old fmt.Sprintf overlay, load both with test module,
	// assert #opmReleaseMeta.identity UUIDs match.
	modulePath := testModulePath(t, "test-module")

	ctx := cuecontext.New()
	builder := NewReleaseBuilder(ctx, "")

	name := "my-release"
	namespace := "production"

	// String overlay (old approach)
	strOverlay := fmt.Sprintf(`package testmodule

import "uuid"

#opmReleaseMeta: {
	name:      %q
	namespace: %q
	fqn:       metadata.fqn
	version:   metadata.version
	identity:  string & uuid.SHA1("c1cbe76d-5687-5a47-bfe6-83b081b15413", "\(fqn):\(name):\(namespace)")
	labels: metadata.labels & {
		"module-release.opmodel.dev/name":    name
		"module-release.opmodel.dev/version": version
		"module-release.opmodel.dev/uuid":    identity
	}
}
`, name, namespace)

	// AST overlay (new approach)
	astFile := builder.generateOverlayAST("testmodule", ReleaseOptions{
		Name:      name,
		Namespace: namespace,
	})
	astBytes, err := format.Node(astFile)
	require.NoError(t, err)

	overlayPath := filepath.Join(modulePath, "opm_release_overlay.cue")

	// Build with string overlay
	strCfg := &load.Config{
		Dir: modulePath,
		Overlay: map[string]load.Source{
			overlayPath: load.FromBytes([]byte(strOverlay)),
		},
	}
	strInsts := load.Instances([]string{"."}, strCfg)
	require.Len(t, strInsts, 1)
	require.NoError(t, strInsts[0].Err)
	strVal := ctx.BuildInstance(strInsts[0])
	require.NoError(t, strVal.Err())

	// Build with AST overlay (fresh context to avoid sharing)
	ctx2 := cuecontext.New()
	astCfg := &load.Config{
		Dir: modulePath,
		Overlay: map[string]load.Source{
			overlayPath: load.FromBytes(astBytes),
		},
	}
	astInsts := load.Instances([]string{"."}, astCfg)
	require.Len(t, astInsts, 1)
	require.NoError(t, astInsts[0].Err)
	astVal := ctx2.BuildInstance(astInsts[0])
	require.NoError(t, astVal.Err())

	// Compare identity values
	strIdentity, err := strVal.LookupPath(cue.ParsePath("#opmReleaseMeta.identity")).String()
	require.NoError(t, err, "string overlay identity should resolve")

	astIdentity, err := astVal.LookupPath(cue.ParsePath("#opmReleaseMeta.identity")).String()
	require.NoError(t, err, "AST overlay identity should resolve")

	assert.Equal(t, strIdentity, astIdentity, "both overlays should produce the same identity UUID")
}

// ---------------------------------------------------------------------------
// 4.4: TestInspectModule_StaticMetadata
// ---------------------------------------------------------------------------

func TestInspectModule_StaticMetadata(t *testing.T) {
	ctx := cuecontext.New()
	builder := NewReleaseBuilder(ctx, "")

	modulePath := testModulePath(t, "test-module")
	inspection, err := builder.InspectModule(modulePath)
	require.NoError(t, err)

	assert.Equal(t, "test-module", inspection.Name)
	assert.Equal(t, "default", inspection.DefaultNamespace)
	assert.Equal(t, "testmodule", inspection.PkgName)
}

// ---------------------------------------------------------------------------
// 4.5: TestInspectModule_MissingMetadata
// ---------------------------------------------------------------------------

func TestInspectModule_MissingMetadata(t *testing.T) {
	ctx := cuecontext.New()
	builder := NewReleaseBuilder(ctx, "")

	modulePath := testModulePath(t, "no-metadata-module")
	inspection, err := builder.InspectModule(modulePath)
	require.NoError(t, err)

	// metadata.name is a computed expression, so AST walk returns empty
	assert.Empty(t, inspection.Name, "computed metadata.name should return empty")
	assert.Equal(t, "nometadata", inspection.PkgName)
}

// ---------------------------------------------------------------------------
// 4.6: TestExtractMetadataFromAST
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
			name, ns := extractMetadataFromAST(tt.files)
			assert.Equal(t, tt.expectedName, name)
			assert.Equal(t, tt.expectedNamespace, ns)
		})
	}
}
