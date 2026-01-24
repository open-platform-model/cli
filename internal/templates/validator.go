package templates

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// CUE identifier validation regex.
// CUE identifiers must start with a letter or underscore and contain only letters, digits, and underscores.
var cueIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateCUEIdentifier checks if a string is a valid CUE identifier.
func ValidateCUEIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if !cueIdentifierRegex.MatchString(name) {
		return fmt.Errorf("invalid CUE identifier %q: must start with a letter or underscore and contain only letters, digits, and underscores", name)
	}

	// Check for CUE reserved words
	if isReservedWord(name) {
		return fmt.Errorf("invalid CUE identifier %q: cannot use reserved word", name)
	}

	return nil
}

// ValidateModuleName checks if a module name is valid.
// Module names are more permissive than CUE identifiers (allow hyphens for directory names).
func ValidateModuleName(name string) error {
	if name == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	// Check for invalid characters
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return fmt.Errorf("invalid module name %q: contains invalid character %q", name, r)
		}
	}

	// Must start with a letter
	if !unicode.IsLetter(rune(name[0])) {
		return fmt.Errorf("invalid module name %q: must start with a letter", name)
	}

	return nil
}

// SanitizeName converts a module name to a valid CUE package name.
func SanitizeName(name string) string {
	// Replace hyphens and dots with underscores
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' {
			result = append(result, c)
		} else if c == '-' || c == '.' {
			result = append(result, '_')
		}
	}

	// Ensure it doesn't start with a number
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = append([]byte{'_'}, result...)
	}

	// Handle empty result
	if len(result) == 0 {
		return "module"
	}

	return string(result)
}

// DeriveModulePath derives a CUE module path from a directory name.
// Format: example.com/<dirname> with hyphens converted to underscores.
func DeriveModulePath(dirname string) string {
	// Convert hyphens to underscores in the module path
	sanitized := strings.ReplaceAll(dirname, "-", "_")
	return fmt.Sprintf("example.com/%s", sanitized)
}

// isReservedWord checks if a name is a CUE reserved word.
func isReservedWord(name string) bool {
	reserved := map[string]bool{
		"_":       true,
		"bool":    true,
		"bottom":  true,
		"bytes":   true,
		"float":   true,
		"float32": true,
		"float64": true,
		"for":     true,
		"if":      true,
		"import":  true,
		"in":      true,
		"int":     true,
		"int8":    true,
		"int16":   true,
		"int32":   true,
		"int64":   true,
		"int128":  true,
		"len":     true,
		"let":     true,
		"null":    true,
		"number":  true,
		"package": true,
		"rune":    true,
		"string":  true,
		"struct":  true,
		"top":     true,
		"true":    true,
		"false":   true,
		"uint":    true,
		"uint8":   true,
		"uint16":  true,
		"uint32":  true,
		"uint64":  true,
		"uint128": true,
	}
	return reserved[name]
}
