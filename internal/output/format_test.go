package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutputFormatValid(t *testing.T) {
	tests := []struct {
		format OutputFormat
		valid  bool
	}{
		{FormatYAML, true},
		{FormatJSON, true},
		{FormatTable, true},
		{FormatDir, true},
		{OutputFormat("invalid"), false},
		{OutputFormat(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.format.Valid())
		})
	}
}

func TestOutputFormatString(t *testing.T) {
	assert.Equal(t, "yaml", FormatYAML.String())
	assert.Equal(t, "json", FormatJSON.String())
	assert.Equal(t, "table", FormatTable.String())
	assert.Equal(t, "dir", FormatDir.String())
}

func TestParseOutputFormat(t *testing.T) {
	tests := []struct {
		input string
		want  OutputFormat
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
		{"invalid", OutputFormat("invalid"), false},
		{"", OutputFormat(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, valid := ParseOutputFormat(tt.input)
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
