package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatHealthStatus(t *testing.T) {
	tests := []struct {
		status   string
		contains string
	}{
		{"Ready", "Ready"},
		{"Complete", "Complete"},
		{"NotReady", "NotReady"},
		{"Missing", "Missing"},
		{"Unknown", "Unknown"},
		{"", ""},
		{"other", "other"},
	}
	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			result := FormatHealthStatus(tc.status)
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestFormatComponent(t *testing.T) {
	t.Run("empty returns dash", func(t *testing.T) {
		assert.Equal(t, "-", FormatComponent(""))
	})
	t.Run("non-empty renders name", func(t *testing.T) {
		result := FormatComponent("server")
		assert.Contains(t, result, "server")
	})
}
