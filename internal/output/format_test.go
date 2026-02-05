package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatValid(t *testing.T) {
	tests := []struct {
		format Format
		valid  bool
	}{
		{FormatYAML, true},
		{FormatJSON, true},
		{FormatTable, true},
		{FormatDir, true},
		{Format("invalid"), false},
		{Format(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.format.Valid())
		})
	}
}

func TestFormatString(t *testing.T) {
	assert.Equal(t, "yaml", FormatYAML.String())
	assert.Equal(t, "json", FormatJSON.String())
	assert.Equal(t, "table", FormatTable.String())
	assert.Equal(t, "dir", FormatDir.String())
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input string
		want  Format
		valid bool
	}{
		{"yaml", FormatYAML, true},
		{"YAML", FormatYAML, true},
		{"json", FormatJSON, true},
		{"JSON", FormatJSON, true},
		{"table", FormatTable, true},
		{"TABLE", FormatTable, true},
		{"dir", FormatDir, true},
		{"DIR", FormatDir, true},
		{"invalid", Format("invalid"), false},
		{"", Format(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, valid := ParseFormat(tt.input)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

func TestValidFormats(t *testing.T) {
	formats := ValidFormats()

	assert.Contains(t, formats, "yaml")
	assert.Contains(t, formats, "json")
	assert.Contains(t, formats, "table")
	assert.Contains(t, formats, "dir")
	assert.Len(t, formats, 4)
}
