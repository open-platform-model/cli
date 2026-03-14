package render

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
)

func finalizeValue(cueCtx *cue.Context, v cue.Value) (cue.Value, error) {
	syntaxNode := v.Syntax(cue.Final())

	expr, ok := syntaxNode.(ast.Expr)
	if !ok {
		return cue.Value{}, fmt.Errorf(
			"finalization produced %T instead of ast.Expr; value likely contains unresolved imports or definition fields that should have been resolved upstream",
			syntaxNode,
		)
	}

	dataVal := cueCtx.BuildExpr(expr)
	if err := dataVal.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building finalized value: %w", err)
	}
	return dataVal, nil
}
