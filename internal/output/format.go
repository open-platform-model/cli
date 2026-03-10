// Package output provides terminal output utilities.
package output

import "strings"

// Format specifies the output format.
type Format string

const (
	// FormatYAML outputs as YAML.
	FormatYAML Format = "yaml"

	// FormatJSON outputs as JSON.
	FormatJSON Format = "json"

	// FormatTable outputs as a formatted table.
	FormatTable Format = "table"

	// FormatDir outputs to a directory.
	FormatDir Format = "dir"

	// FormatWide outputs as a wide table with additional columns (kubectl-style).
	FormatWide Format = "wide"
)

// Valid returns true if the format is valid.
func (f Format) Valid() bool {
	switch f {
	case FormatYAML, FormatJSON, FormatTable, FormatDir, FormatWide:
		return true
	default:
		return false
	}
}

// ValidFormats returns all valid format strings.
func ValidFormats() []string {
	return []string{string(FormatTable), string(FormatWide), string(FormatJSON), string(FormatYAML), string(FormatDir)}
}

// String returns the string representation.
func (f Format) String() string {
	return string(f)
}

// ParseFormat parses a string into an Format.
func ParseFormat(s string) (Format, bool) {
	f := Format(strings.ToLower(s))
	return f, f.Valid()
}

// IsManifestFormat reports whether the format is supported by manifest writers.
func IsManifestFormat(f Format) bool {
	return f == FormatYAML || f == FormatJSON
}
