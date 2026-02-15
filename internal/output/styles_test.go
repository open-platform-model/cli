package output

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestStatusStyle(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		wantBold bool
		wantFG   lipgloss.Color
		wantDim  bool
	}{
		{
			name:   "created returns green",
			status: StatusCreated,
			wantFG: colorGreen,
		},
		{
			name:   "configured returns yellow",
			status: StatusConfigured,
			wantFG: ColorYellow,
		},
		{
			name:    "unchanged returns faint",
			status:  StatusUnchanged,
			wantDim: true,
		},
		{
			name:   "deleted returns red",
			status: StatusDeleted,
			wantFG: colorRed,
		},
		{
			name:     "failed returns bold red",
			status:   statusFailed,
			wantBold: true,
			wantFG:   colorBoldRed,
		},
		{
			name:   "unknown returns default unstyled",
			status: "unknown-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := statusStyle(tt.status)
			if tt.wantBold {
				assert.True(t, style.GetBold(), "expected bold")
			}
			if tt.wantFG != "" {
				assert.Equal(t, tt.wantFG, style.GetForeground(), "foreground color mismatch")
			}
			if tt.wantDim {
				assert.True(t, style.GetFaint(), "expected faint")
			}
		})
	}
}

func TestFormatResourceLine(t *testing.T) {
	// Use lipgloss with no color for deterministic output in tests.
	originalHasDarkBG := lipgloss.HasDarkBackground()
	_ = originalHasDarkBG // lipgloss state is global, we test content structure

	tests := []struct {
		name      string
		kind      string
		namespace string
		resName   string
		status    string
		wantPath  string
	}{
		{
			name:      "namespaced resource",
			kind:      "Deployment",
			namespace: "production",
			resName:   "my-app",
			status:    StatusCreated,
			wantPath:  "Deployment/production/my-app",
		},
		{
			name:      "cluster-scoped resource (empty namespace)",
			kind:      "ClusterRole",
			namespace: "",
			resName:   "admin-view",
			status:    StatusUnchanged,
			wantPath:  "ClusterRole/admin-view",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatResourceLine(tt.kind, tt.namespace, tt.resName, tt.status)

			// The rendered string contains ANSI codes. Strip them for content checks.
			// We check that the path and status text appear in the output.
			assert.Contains(t, result, tt.wantPath, "should contain resource path")
			assert.Contains(t, result, tt.status, "should contain status text")
			assert.True(t, strings.HasPrefix(stripAnsi(result), "r:"), "should start with r: prefix")
		})
	}

	t.Run("alignment consistency", func(t *testing.T) {
		// Two lines with different path lengths should have status starting
		// at the same position (both paths shorter than min column width).
		line1 := FormatResourceLine("Service", "default", "svc", StatusCreated)
		line2 := FormatResourceLine("Deployment", "default", "my-app", StatusCreated)

		// Find the position of the status text in stripped output.
		stripped1 := stripAnsi(line1)
		stripped2 := stripAnsi(line2)

		idx1 := strings.Index(stripped1, StatusCreated)
		idx2 := strings.Index(stripped2, StatusCreated)

		assert.Equal(t, idx1, idx2, "status words should align to same column")
	})
}

func TestFormatCheckmark(t *testing.T) {
	result := FormatCheckmark("Module applied")
	assert.Contains(t, result, "✔", "should contain checkmark")
	assert.Contains(t, result, "Module applied", "should contain message")
}

// stripAnsi removes ANSI escape sequences for content assertions.
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}
