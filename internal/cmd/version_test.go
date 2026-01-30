// Package cmd provides CLI command implementations.
package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVersionCmd(t *testing.T) {
	cmd := NewVersionCmd()

	assert.Equal(t, "version", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestVersionCmd_Execute(t *testing.T) {
	cmd := NewVersionCmd()

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	// Note: output.Println writes to stdout, not cmd.SetOut()
	// We just verify the command executes without error
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestExtractMajorMinorCmd(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v0.15.0", "0.15"},
		{"0.15.0", "0.15"},
		{"v1.2.3", "1.2"},
		{"v0.15.0-alpha.1", "0.15"},
		{"v0.15", "0.15"},
		{"v0", "0"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractMajorMinor(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
