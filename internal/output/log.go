// Package output provides terminal output utilities.
package output

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// LogConfig holds configuration for the logger.
type LogConfig struct {
	// Verbose enables debug-level logging, timestamps, and caller info.
	Verbose bool

	// Timestamps controls timestamp display. Nil means use default (true).
	// When Verbose is true, timestamps are forced on regardless.
	Timestamps *bool
}

// Logger is the global logger instance.
// Initialized with default options; call SetupLogging to configure.
var logger = log.NewWithOptions(os.Stderr, log.Options{
	ReportTimestamp: true,
	ReportCaller:    false,
	TimeFormat:      "15:04:05",
})

// SetupLogging configures the global logger based on the provided config.
func SetupLogging(cfg LogConfig) {
	level := log.InfoLevel
	if cfg.Verbose {
		level = log.DebugLevel
	}

	// Resolve timestamps: verbose forces on, otherwise flag/config/default(true).
	showTimestamps := true
	if cfg.Timestamps != nil {
		showTimestamps = *cfg.Timestamps
	}
	if cfg.Verbose {
		showTimestamps = true
	}

	logger = log.NewWithOptions(os.Stderr, log.Options{
		Level:           level,
		ReportTimestamp: showTimestamps,
		ReportCaller:    cfg.Verbose,
		TimeFormat:      "15:04:05",
	})
}

// ModuleLogger returns a child logger scoped to a module name.
// The prefix renders as: m:<name>:
// with dim "m:" and cyan module name. The trailing ":" is appended
// automatically by charmbracelet/log's prefix renderer.
func ModuleLogger(name string) *log.Logger {
	prefix := fmt.Sprintf("%s%s",
		styleDim.Render("m:"),
		lipgloss.NewStyle().Foreground(ColorCyan).Render(name),
	)

	return logger.WithPrefix(prefix)
}

// Debug logs a debug message.
func Debug(msg string, keyvals ...interface{}) {
	logger.Debug(msg, keyvals...)
}

// Info logs an info message.
func Info(msg string, keyvals ...interface{}) {
	logger.Info(msg, keyvals...)
}

// Warn logs a warning message.
func Warn(msg string, keyvals ...interface{}) {
	logger.Warn(msg, keyvals...)
}

// Error logs an error message.
func Error(msg string, keyvals ...interface{}) {
	logger.Error(msg, keyvals...)
}

// Print prints a message to stdout without any formatting.
func Print(msg string) {
	os.Stdout.WriteString(msg)
}

// Println prints a message to stdout with a newline.
func Println(msg string) {
	os.Stdout.WriteString(msg + "\n")
}

// Details prints supplementary multi-line content to stderr.
// Use for structured error details (e.g. CUE validation output)
// that don't fit the key-value log format.
func Details(msg string) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, msg)
}

// Prompt prints an interactive prompt to stderr (no newline).
// Use for user input prompts like confirmation dialogs.
func Prompt(msg string) {
	os.Stderr.WriteString(msg)
}

// ClearScreen clears the terminal screen and moves cursor to top-left.
// Use for watch/refresh mode interfaces.
func ClearScreen() {
	os.Stdout.WriteString("\033[2J\033[H")
}

// BoolPtr returns a pointer to a bool value. Convenience for LogConfig.Timestamps.
func BoolPtr(b bool) *bool {
	return &b
}
