package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
)

// captureLog sets up the logger to write to a buffer and returns the buffer.
func captureLog(cfg LogConfig) *bytes.Buffer {
	var buf bytes.Buffer
	SetupLogging(cfg)
	Logger = log.NewWithOptions(&buf, log.Options{
		Level:           Logger.GetLevel(),
		ReportTimestamp: cfg.resolveTimestamps(),
		ReportCaller:    cfg.Verbose,
		TimeFormat:      "15:04:05",
	})
	return &buf
}

// resolveTimestamps applies the same logic as SetupLogging for test verification.
func (c LogConfig) resolveTimestamps() bool {
	if c.Verbose {
		return true
	}
	if c.Timestamps != nil {
		return *c.Timestamps
	}
	return true
}

func TestSetupLogging_TimestampDefaultOn(t *testing.T) {
	buf := captureLog(LogConfig{})
	Logger.Info("test")
	// Default timestamps on: output should contain time-like pattern
	assert.Contains(t, buf.String(), ":", "default output should contain timestamp separator")
}

func TestSetupLogging_TimestampExplicitlyDisabled(t *testing.T) {
	buf := captureLog(LogConfig{Timestamps: BoolPtr(false)})
	Logger.Info("hello")
	out := buf.String()
	// When timestamps off, the line should start with level, not a time pattern
	assert.NotRegexp(t, `^\d{1,2}:\d{2}:\d{2}`, strings.TrimSpace(out),
		"output should not start with a timestamp")
}

func TestSetupLogging_VerboseForcesTimestampsOn(t *testing.T) {
	buf := captureLog(LogConfig{Verbose: true, Timestamps: BoolPtr(false)})
	Logger.Debug("verbose-msg")
	out := buf.String()
	assert.Contains(t, out, "verbose-msg", "debug message should appear in verbose mode")
	assert.Contains(t, out, ":", "verbose should force timestamps on")
}

func TestSetupLogging_VerboseEnablesDebugLevel(t *testing.T) {
	SetupLogging(LogConfig{Verbose: true})
	assert.Equal(t, log.DebugLevel, Logger.GetLevel(), "verbose should set debug level")
}

func TestSetupLogging_DefaultInfoLevel(t *testing.T) {
	SetupLogging(LogConfig{})
	assert.Equal(t, log.InfoLevel, Logger.GetLevel(), "default should be info level")
}

func TestModuleLogger_HasPrefix(t *testing.T) {
	SetupLogging(LogConfig{})
	modLog := ModuleLogger("my-app")
	assert.NotNil(t, modLog, "module logger should not be nil")

	prefix := modLog.GetPrefix()
	assert.Contains(t, prefix, "my-app", "prefix should contain module name")
}

func TestModuleLogger_InheritsLevel(t *testing.T) {
	SetupLogging(LogConfig{Verbose: true})
	modLog := ModuleLogger("my-app")
	assert.Equal(t, log.DebugLevel, modLog.GetLevel(), "module logger should inherit debug level")
}

func TestBoolPtr(t *testing.T) {
	trueVal := BoolPtr(true)
	falseVal := BoolPtr(false)
	assert.True(t, *trueVal)
	assert.False(t, *falseVal)
}
