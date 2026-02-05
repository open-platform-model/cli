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
)

// Valid returns true if the format is valid.
func (f Format) Valid() bool {
	switch f {
	case FormatYAML, FormatJSON, FormatTable, FormatDir:
		return true
	default:
		return false
	}
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

// ValidFormats returns all valid output format strings.
func ValidFormats() []string {
	return []string{
		string(FormatYAML),
		string(FormatJSON),
		string(FormatTable),
		string(FormatDir),
	}
}
