package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandTilde(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err, "should get home directory")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no tilde",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path without tilde",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "tilde only",
			input:    "~",
			expected: homeDir,
		},
		{
			name:     "tilde with slash",
			input:    "~/.kube/config",
			expected: filepath.Join(homeDir, ".kube", "config"),
		},
		{
			name:     "tilde with path",
			input:    "~/Documents/file.txt",
			expected: filepath.Join(homeDir, "Documents", "file.txt"),
		},
		{
			name:     "tilde username pattern (not expanded)",
			input:    "~username/file",
			expected: "~username/file",
		},
		{
			name:     "tilde in middle (not expanded)",
			input:    "/path/~/file",
			expected: "/path/~/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandTilde(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
