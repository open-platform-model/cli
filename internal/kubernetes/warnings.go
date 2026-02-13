package kubernetes

import (
	"fmt"

	"github.com/opmodel/cli/internal/output"
)

// warningLogger abstracts the logging calls used by opmWarningHandler,
// enabling test injection without modifying the output package.
type warningLogger interface {
	Warn(msg string, keyvals ...interface{})
	Debug(msg string, keyvals ...interface{})
}

// outputWarningLogger delegates to the output package's global logger.
type outputWarningLogger struct{}

func (outputWarningLogger) Warn(msg string, keyvals ...interface{})  { output.Warn(msg, keyvals...) }
func (outputWarningLogger) Debug(msg string, keyvals ...interface{}) { output.Debug(msg, keyvals...) }

// opmWarningHandler implements rest.WarningHandler to route K8s API warnings
// through charmbracelet/log instead of klog.
type opmWarningHandler struct {
	// level controls how warnings are displayed:
	// "warn" - show as WARN level (default)
	// "debug" - show as DEBUG level (only visible with --verbose)
	// "suppress" - drop entirely
	level  string
	logger warningLogger
}

// HandleWarningHeader implements rest.WarningHandler interface.
// Routes K8s API deprecation warnings through charmbracelet/log based on configured level.
func (h *opmWarningHandler) HandleWarningHeader(code int, agent, text string) {
	// Format the warning message
	msg := fmt.Sprintf("k8s API warning: %s", text)

	switch h.level {
	case "warn":
		h.logger.Warn(msg)
	case "debug":
		h.logger.Debug(msg)
	case "suppress":
		// Drop the warning entirely
		return
	default:
		// Fallback to warn for unknown levels
		h.logger.Warn(msg)
	}
}
