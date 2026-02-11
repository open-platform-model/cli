package kubernetes

import (
	"fmt"

	"github.com/opmodel/cli/internal/output"
)

// opmWarningHandler implements rest.WarningHandler to route K8s API warnings
// through charmbracelet/log instead of klog.
type opmWarningHandler struct {
	// level controls how warnings are displayed:
	// "warn" - show as WARN level (default)
	// "debug" - show as DEBUG level (only visible with --verbose)
	// "suppress" - drop entirely
	level string
}

// HandleWarningHeader implements rest.WarningHandler interface.
// Routes K8s API deprecation warnings through charmbracelet/log based on configured level.
func (h *opmWarningHandler) HandleWarningHeader(code int, agent string, text string) {
	// Format the warning message
	msg := fmt.Sprintf("k8s API warning: %s", text)

	switch h.level {
	case "warn":
		output.Warn(msg)
	case "debug":
		output.Debug(msg)
	case "suppress":
		// Drop the warning entirely
		return
	default:
		// Fallback to warn for unknown levels
		output.Warn(msg)
	}
}
