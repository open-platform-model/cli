// Package output provides terminal output utilities.
package output

import (
	"os"

	"github.com/charmbracelet/log"
)

// Logger is the global logger instance.
var Logger *log.Logger

func init() {
	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
		ReportCaller:    false,
	})
}

// SetupLogging configures the logger based on verbosity.
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

// Fatal logs a fatal message and exits.
func Fatal(msg string, keyvals ...interface{}) {
	Logger.Fatal(msg, keyvals...)
}

// Print prints a message to stdout without any formatting.
func Print(msg string) {
	os.Stdout.WriteString(msg)
}

// Println prints a message to stdout with a newline.
func Println(msg string) {
	os.Stdout.WriteString(msg + "\n")
}
