package release

import (
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/token"

	"github.com/opmodel/cli/internal/core"
)

// opmNamespaceUUID is the UUID v5 namespace for computing deterministic identities.
// Computed as: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("opmodel.dev")).String()
// Used by the CUE overlay to compute release identity via uuid.SHA1.
const opmNamespaceUUID = "c1cbe76d-5687-5a47-bfe6-83b081b15413"

// generateOverlayAST builds the CUE overlay file as a typed AST.
//
// The overlay adds a #opmReleaseMeta definition to the module's CUE package.
// This definition computes:
//   - Release identity (uuid.SHA1 of fqn:name:namespace)
//   - Standard release labels (module-release.opmodel.dev/*)
//   - Module labels (inherited from module.metadata.labels)
//
// Key rules:
//   - Field labels referenced from nested scopes (name, namespace, fqn, version, identity)
//     use ast.NewIdent (unquoted identifier labels) for CUE scope resolution.
//   - Label keys with special characters use ast.NewString (quoted string labels).
//   - astutil.Resolve is called after construction to wire up scope references.
func generateOverlayAST(pkgName string, opts Options) *ast.File {
	// Build the uuid.SHA1(...) call expression
	uuidCall := ast.NewCall(
		&ast.SelectorExpr{
			X:   ast.NewIdent("uuid"),
			Sel: ast.NewIdent("SHA1"),
		},
		ast.NewString(opmNamespaceUUID),
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
	// Label keys use ast.NewString (quoted) because they contain special chars.
	// The values (name, version, identity) are ast.NewIdent references to sibling fields.
	labelsExpr := &ast.BinaryExpr{
		X: &ast.SelectorExpr{
			X:   ast.NewIdent("metadata"),
			Sel: ast.NewIdent("labels"),
		},
		Op: token.AND,
		Y: ast.NewStruct(
			ast.NewString(core.LabelReleaseName), ast.NewIdent("name"),
			ast.NewString(core.LabelReleaseUUID), ast.NewIdent("identity"),
		),
	}

	// Build #opmReleaseMeta struct with *ast.Field entries using ast.NewIdent labels.
	// Using ast.NewIdent for labels produces unquoted identifiers,
	// which CUE can resolve as references from nested scopes.
	releaseMetaStruct := ast.NewStruct(
		&ast.Field{Label: ast.NewIdent("name"), Value: ast.NewString(opts.Name)},
		&ast.Field{Label: ast.NewIdent("namespace"), Value: ast.NewString(opts.Namespace)},
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
