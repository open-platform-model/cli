package output

import (
	"io"
	"os"

	"github.com/charmbracelet/log"
)

// Logger is the global logger instance.
var Logger *log.Logger

func init() {
	// Initialize with default settings
	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
		ReportCaller:    false,
	})
}

// SetupLogging configures the global logger.
func SetupLogging(verbose bool) {
	level := log.InfoLevel
	if verbose {
		level = log.DebugLevel
	}

	Logger = log.NewWithOptions(os.Stderr, log.Options{
		Level:           level,
		ReportTimestamp: verbose,
		ReportCaller:    verbose,
	})

	// Disable colors if NO_COLOR is set
	if IsNoColor() {
		Logger.SetStyles(log.DefaultStyles())
	}
}

// SetOutput sets the output writer for the logger.
func SetOutput(w io.Writer) {
	Logger = log.NewWithOptions(w, log.Options{
		ReportTimestamp: false,
		ReportCaller:    false,
	})
}

// Debug logs a debug message.
func Debug(msg string, keyvals ...interface{}) {
	Logger.Debug(msg, keyvals...)
}

// Info logs an info message.
func Info(msg string, keyvals ...interface{}) {
	Logger.Info(msg, keyvals...)
}

// Warn logs a warning message.
func Warn(msg string, keyvals ...interface{}) {
	Logger.Warn(msg, keyvals...)
}

// Error logs an error message.
func Error(msg string, keyvals ...interface{}) {
	Logger.Error(msg, keyvals...)
}

// Fatal logs an error message and exits with code 1.
func Fatal(msg string, keyvals ...interface{}) {
	Logger.Fatal(msg, keyvals...)
}

// WithPrefix returns a new logger with a prefix.
func WithPrefix(prefix string) *log.Logger {
	return Logger.WithPrefix(prefix)
}
