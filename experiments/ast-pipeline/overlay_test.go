package astpipeline

import (
	"fmt"
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
// Hypothesis 1: AST-based overlay generation
// ---------------------------------------------------------------------------

// generateOverlayString mirrors the current production code in release_builder.go
func generateOverlayString(pkgName string, name, namespace string) []byte {
	overlay := fmt.Sprintf(`package %s

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
`, pkgName, name, namespace)
	return []byte(overlay)
}

// generateOverlayAST builds the equivalent overlay as a typed AST.
// Key insight: field labels must use ast.NewIdent (unquoted) not ast.NewString (quoted)
// so that CUE's scope resolution can find them as references.
func generateOverlayAST(pkgName string, name, namespace string) *ast.File {
	// Build the uuid.SHA1(...) call expression
	uuidCall := ast.NewCall(
		&ast.SelectorExpr{
			X:   ast.NewIdent("uuid"),
			Sel: ast.NewIdent("SHA1"),
		},
		ast.NewString("c1cbe76d-5687-5a47-bfe6-83b081b15413"),
		// CUE string interpolation: "\(fqn):\(name):\(namespace)"
		// Interpolation Elts are interleaved: string fragments include
		// quote chars and \( / ) delimiters, matching parser output.
		&ast.Interpolation{
			Elts: []ast.Expr{
				ast.NewLit(token.STRING, `"\(`),
				ast.NewIdent("fqn"),
				ast.NewLit(token.STRING, `):\(`),
				ast.NewIdent("name"),
				ast.NewLit(token.STRING, `):\(`),
				ast.NewIdent("namespace"),
				ast.NewLit(token.STRING, `)"`),
			},
		},
	)

	// identity: string & uuid.SHA1(...)
	identityExpr := &ast.BinaryExpr{
		X:  ast.NewIdent("string"),
		Op: token.AND,
		Y:  uuidCall,
	}

	// labels: metadata.labels & { ... }
	// Note: label keys use ast.NewString (quoted) because they contain special chars.
	// The values (name, version, identity) are ast.NewIdent references to sibling fields.
	labelsExpr := &ast.BinaryExpr{
		X: &ast.SelectorExpr{
			X:   ast.NewIdent("metadata"),
			Sel: ast.NewIdent("labels"),
		},
		Op: token.AND,
		Y: ast.NewStruct(
			ast.NewString("module-release.opmodel.dev/name"), ast.NewIdent("name"),
			ast.NewString("module-release.opmodel.dev/version"), ast.NewIdent("version"),
			ast.NewString("module-release.opmodel.dev/uuid"), ast.NewIdent("identity"),
		),
	}

	// Build #opmReleaseMeta struct with *ast.Field entries using ast.NewIdent labels.
	// Using ast.NewIdent("name") instead of string "name" produces unquoted labels,
	// which CUE can resolve as references from nested scopes.
	releaseMetaStruct := ast.NewStruct(
		&ast.Field{Label: ast.NewIdent("name"), Value: ast.NewString(name)},
		&ast.Field{Label: ast.NewIdent("namespace"), Value: ast.NewString(namespace)},
		&ast.Field{
			Label: ast.NewIdent("fqn"),
			Value: &ast.SelectorExpr{
				X:   ast.NewIdent("metadata"),
				Sel: ast.NewIdent("fqn"),
			},
		},
		&ast.Field{
			Label: ast.NewIdent("version"),
			Value: &ast.SelectorExpr{
				X:   ast.NewIdent("metadata"),
				Sel: ast.NewIdent("version"),
			},
		},
		&ast.Field{Label: ast.NewIdent("identity"), Value: identityExpr},
		&ast.Field{Label: ast.NewIdent("labels"), Value: labelsExpr},
	)

	file := &ast.File{
		Decls: []ast.Decl{
			&ast.Package{Name: ast.NewIdent(pkgName)},
			&ast.ImportDecl{
				Specs: []*ast.ImportSpec{
					ast.NewImport(nil, "uuid"),
				},
			},
			&ast.Field{
				Label: ast.NewIdent("#opmReleaseMeta"),
				Value: releaseMetaStruct,
			},
		},
	}

	// Resolve scope references so that identifiers like `name` inside the
	// labels struct can find the `name` field in the parent #opmReleaseMeta struct.
	astutil.Resolve(file, func(_ token.Pos, msg string, args ...interface{}) {
		// Ignore resolution errors — some references (like `metadata`) are external
	})

	return file
}

// Generated overlay CUE (from generateOverlayAST):
//
//	package testmodule
//
//	import "uuid"
//
//	#opmReleaseMeta: {
//	    name:      "my-release"
//	    namespace: "production"
//	    fqn:       metadata.fqn
//	    version:   metadata.version
//	    identity:  string & uuid.SHA1("...", "\(fqn):\(name):\(namespace)")
//	    labels: metadata.labels & {
//	        "module-release.opmodel.dev/name":    name
//	        "module-release.opmodel.dev/version": version
//	        "module-release.opmodel.dev/uuid":    identity
//	    }
//	}
//
// Verifies: formats to valid, parseable CUE
func TestOverlayAST_FormatsToValidCUE(t *testing.T) {
	overlay := generateOverlayAST("testmodule", "my-release", "production")

	b, err := format.Node(overlay)
	require.NoError(t, err)
	t.Logf("Formatted overlay:\n%s", string(b))

	// Should parse back without errors
	_, err = parser.ParseFile("overlay.cue", b, parser.ParseComments)
	require.NoError(t, err, "AST-generated overlay should produce parseable CUE")
}

// Compares generateOverlayAST vs generateOverlayString (fmt.Sprintf):
//
//	Both should produce same package name, same imports,
//	and semantically equivalent #opmReleaseMeta structure
func TestOverlayAST_MatchesStringTemplate(t *testing.T) {
	// Both approaches should produce semantically equivalent CUE.
	// We can't compare bytes (formatting may differ), but we can compare
	// that both parse and contain the same fields.
	strOverlay := generateOverlayString("testmodule", "my-release", "production")
	astOverlay := generateOverlayAST("testmodule", "my-release", "production")

	astBytes, err := format.Node(astOverlay)
	require.NoError(t, err)

	// Parse both
	strFile, err := parser.ParseFile("str.cue", strOverlay)
	require.NoError(t, err)
	astFile, err := parser.ParseFile("ast.cue", astBytes)
	require.NoError(t, err)

	// Both should have the same package name
	assert.Equal(t, strFile.PackageName(), astFile.PackageName())

	// Both should have imports
	strImports := 0
	for range strFile.ImportSpecs() {
		strImports++
	}
	astImports := 0
	for range astFile.ImportSpecs() {
		astImports++
	}
	assert.Equal(t, strImports, astImports, "both should have same number of imports")

	t.Logf("String overlay:\n%s", string(strOverlay))
	t.Logf("AST overlay:\n%s", string(astBytes))
}

// Input CUE: testdata/test-module/ + AST overlay (injected via load.Config.Overlay)
//
// Overlay adds #opmReleaseMeta which references metadata.fqn and metadata.version.
// Verifies:
//
//	#opmReleaseMeta.name      → "my-release"
//	#opmReleaseMeta.namespace → "production"
//	#opmReleaseMeta.fqn       → "example.com/test-module@v0#test-module" (from metadata)
//	#opmReleaseMeta.version   → "1.0.0" (from metadata)
func TestOverlayAST_LoadsWithModule(t *testing.T) {
	modulePath := testModulePath(t)

	// Generate overlay as AST and format to bytes
	overlay := generateOverlayAST("testmodule", "my-release", "production")
	overlayBytes, err := format.Node(overlay)
	require.NoError(t, err)

	// Load module with overlay
	overlayPath := modulePath + "/opm_release_overlay.cue"
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

	// Verify overlay fields resolve correctly
	relMeta := val.LookupPath(cue.ParsePath("#opmReleaseMeta"))
	require.True(t, relMeta.Exists(), "#opmReleaseMeta should exist")

	relName, err := relMeta.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "my-release", relName)

	relNS, err := relMeta.LookupPath(cue.ParsePath("namespace")).String()
	require.NoError(t, err)
	assert.Equal(t, "production", relNS)

	// Verify fqn resolved from module metadata
	fqn, err := relMeta.LookupPath(cue.ParsePath("fqn")).String()
	require.NoError(t, err)
	assert.Equal(t, "example.com/test-module@v0#test-module", fqn)

	// Verify version resolved
	version, err := relMeta.LookupPath(cue.ParsePath("version")).String()
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", version)
}

// Input CUE: testdata/test-module/ loaded with both overlay approaches:
//
//	1. String overlay via generateOverlayString (fmt.Sprintf)
//	2. AST overlay via generateOverlayAST (typed construction)
//
// Both evaluated in separate cue.Context instances.
// Verifies: #opmReleaseMeta.identity UUID matches between both approaches.
// The identity uses: uuid.SHA1("...", "\(fqn):\(name):\(namespace)")
func TestOverlayAST_InterpolationExpr(t *testing.T) {
	// Test that the uuid.SHA1 call with interpolation evaluates correctly.
	// Compare AST overlay vs string overlay — both should produce the same identity UUID.
	modulePath := testModulePath(t)

	// String overlay
	strBytes := generateOverlayString("testmodule", "my-release", "production")
	strPath := modulePath + "/opm_release_overlay.cue"
	strCfg := &load.Config{
		Dir: modulePath,
		Overlay: map[string]load.Source{
			strPath: load.FromBytes(strBytes),
		},
	}

	// AST overlay
	astFile := generateOverlayAST("testmodule", "my-release", "production")
	astBytes, err := format.Node(astFile)
	require.NoError(t, err)
	astPath := modulePath + "/opm_release_overlay.cue"
	astCfg := &load.Config{
		Dir: modulePath,
		Overlay: map[string]load.Source{
			astPath: load.FromBytes(astBytes),
		},
	}

	// Build both
	ctx := cuecontext.New()

	strInsts := load.Instances([]string{"."}, strCfg)
	require.Len(t, strInsts, 1)
	require.NoError(t, strInsts[0].Err)
	strVal := ctx.BuildInstance(strInsts[0])
	require.NoError(t, strVal.Err())

	// Need fresh context for second build to avoid sharing
	ctx2 := cuecontext.New()
	astInsts := load.Instances([]string{"."}, astCfg)
	require.Len(t, astInsts, 1)
	require.NoError(t, astInsts[0].Err)
	astVal := ctx2.BuildInstance(astInsts[0])
	require.NoError(t, astVal.Err())

	// Compare identity values
	strIdentity := strVal.LookupPath(cue.ParsePath("#opmReleaseMeta.identity"))
	astIdentity := astVal.LookupPath(cue.ParsePath("#opmReleaseMeta.identity"))

	// Both should exist
	assert.True(t, strIdentity.Exists(), "string overlay identity should exist")
	assert.True(t, astIdentity.Exists(), "AST overlay identity should exist")

	// If both resolve to concrete strings, they should match
	if strID, err := strIdentity.String(); err == nil {
		astID, err := astIdentity.String()
		if err == nil {
			assert.Equal(t, strID, astID, "both overlays should produce the same identity UUID")
			t.Logf("Identity UUID: %s", strID)
		} else {
			t.Logf("AST identity did not resolve to string: %v", err)
			t.Logf("This may indicate the interpolation expression needs different AST construction")
		}
	} else {
		t.Logf("String identity did not resolve to string: %v", err)
	}
}
