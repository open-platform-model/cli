package build

import (
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/token"
)

// LoadedComponent is a component with extracted metadata.
// Components are extracted by ReleaseBuilder during the build phase.
type LoadedComponent struct {
	Name        string
	Labels      map[string]string    // Effective labels (merged from resources/traits)
	Annotations map[string]string    // Annotations from metadata.annotations
	Resources   map[string]cue.Value // FQN -> resource value
	Traits      map[string]cue.Value // FQN -> trait value
	Value       cue.Value            // Full component value
}


// extractMetadataFromAST walks CUE AST files to extract metadata.name and
// metadata.defaultNamespace as string literals without CUE evaluation.
// Returns empty strings for fields that are not static string literals.
func extractMetadataFromAST(files []*ast.File) (name, defaultNamespace string) {
	for _, file := range files {
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
			n, ns := extractFieldsFromMetadataStruct(structLit)
			if n != "" && name == "" {
				name = n
			}
			if ns != "" && defaultNamespace == "" {
				defaultNamespace = ns
			}
		}
		if name != "" && defaultNamespace != "" {
			break
		}
	}
	return name, defaultNamespace
}

// extractFieldsFromMetadataStruct scans a metadata struct literal for
// name and defaultNamespace fields with string literal values.
func extractFieldsFromMetadataStruct(s *ast.StructLit) (name, defaultNamespace string) {
	for _, elt := range s.Elts {
		innerField, ok := elt.(*ast.Field)
		if !ok {
			continue
		}
		innerIdent, ok := innerField.Label.(*ast.Ident)
		if !ok {
			continue
		}
		lit, ok := innerField.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		switch innerIdent.Name {
		case "name":
			name = strings.Trim(lit.Value, `"`)
		case "defaultNamespace":
			defaultNamespace = strings.Trim(lit.Value, `"`)
		}
	}
	return name, defaultNamespace
}
