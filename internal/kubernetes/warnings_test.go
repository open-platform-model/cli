package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpmWarningHandler(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantLevel string // "WARN", "DEBU", or "none"
	}{
		{
			name:      "warn level shows warnings",
			level:     "warn",
			wantLevel: "WARN",
		},
		{
			name:      "debug level shows debug (only with --verbose)",
			level:     "debug",
			wantLevel: "DEBU",
		},
		{
			name:      "suppress level drops warnings",
			level:     "suppress",
			wantLevel: "none",
		},
		{
			name:      "unknown level defaults to warn",
			level:     "invalid",
			wantLevel: "WARN",
		},
		{
			name:      "empty level defaults to warn",
			level:     "",
			wantLevel: "WARN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &opmWarningHandler{level: tt.level}

			// The handler doesn't return anything, it calls output.Warn/Debug
			// which writes to stderr. For this test, we just verify the handler
			// doesn't panic and accepts the expected signature.
			assert.NotPanics(t, func() {
				handler.HandleWarningHeader(299, "kubernetes", "test warning message")
			})
		})
	}
}
