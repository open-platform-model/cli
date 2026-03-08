package loader

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
)

// finalizeValue produces a constraint-free data value from a CUE value by
// stripping schema constraints and taking defaults.
//
// It uses Syntax(cue.Final()) to materialise the value into an ast.Expr, then
// recompiles it with BuildExpr. This removes matchN validators, close()
// enforcement, and definition fields, leaving a plain data value suitable for
// FillPath injection into transformers.
//
// This single strategy is sufficient because finalizeValue is only ever called
// on concrete component values — self-contained Kubernetes resource specs with
// all #config references resolved through CUE unification. Such values never
// carry import declarations or unresolved definition references, so Syntax always
// returns ast.Expr (not *ast.File). If it ever returns something else, that
// indicates a bug upstream (schema constraints not resolved before finalization)
// and we surface a clear error rather than silently degrading.
func finalizeValue(cueCtx *cue.Context, v cue.Value) (cue.Value, error) {
	syntaxNode := v.Syntax(cue.Final())

	expr, ok := syntaxNode.(ast.Expr)
	if !ok {
		return cue.Value{}, fmt.Errorf(
			"finalization produced %T instead of ast.Expr; "+
				"value likely contains unresolved imports or definition fields "+
				"that should have been resolved upstream",
			syntaxNode,
		)
	}

	dataVal := cueCtx.BuildExpr(expr)
	if err := dataVal.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building finalized value: %w", err)
	}
	return dataVal, nil
}
