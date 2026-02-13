package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockWarningLogger records which logging method was called and with what message.
type mockWarningLogger struct {
	warnCalled  bool
	debugCalled bool
	lastMsg     string
}

func (m *mockWarningLogger) Warn(msg string, keyvals ...interface{}) {
	m.warnCalled = true
	m.lastMsg = msg
}

func (m *mockWarningLogger) Debug(msg string, keyvals ...interface{}) {
	m.debugCalled = true
	m.lastMsg = msg
}

func TestOpmWarningHandler(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantWarn  bool
		wantDebug bool
	}{
		{
			name:     "warn level routes to Warn",
			level:    "warn",
			wantWarn: true,
		},
		{
			name:      "debug level routes to Debug",
			level:     "debug",
			wantDebug: true,
		},
		{
			name:  "suppress level drops warnings",
			level: "suppress",
		},
		{
			name:     "unknown level defaults to Warn",
			level:    "invalid",
			wantWarn: true,
		},
		{
			name:     "empty level defaults to Warn",
			level:    "",
			wantWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWarningLogger{}
			handler := &opmWarningHandler{level: tt.level, logger: mock}

			handler.HandleWarningHeader(299, "kubernetes", "test warning message")

			assert.Equal(t, tt.wantWarn, mock.warnCalled, "Warn() called")
			assert.Equal(t, tt.wantDebug, mock.debugCalled, "Debug() called")

			if tt.wantWarn || tt.wantDebug {
				assert.Contains(t, mock.lastMsg, "test warning message")
				assert.Contains(t, mock.lastMsg, "k8s API warning:")
			}
		})
	}
}
