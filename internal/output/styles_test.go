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
			name:   "valid returns green",
			status: StatusValid,
			wantFG: colorGreen,
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

func TestStatusValidSameColorAsCreated(t *testing.T) {
	validStyle := statusStyle(StatusValid)
	createdStyle := statusStyle(StatusCreated)
	assert.Equal(t, createdStyle.GetForeground(), validStyle.GetForeground(),
		"valid and created should have the same color")
}

func TestFormatVetCheck(t *testing.T) {
	tests := []struct {
		name       string
		label      string
		detail     string
		wantLabel  string
		wantDetail string
	}{
		{
			name:       "with detail",
			label:      "Config file found",
			detail:     "~/.opm/config.cue",
			wantLabel:  "Config file found",
			wantDetail: "~/.opm/config.cue",
		},
		{
			name:      "without detail",
			label:     "CUE evaluation passed",
			detail:    "",
			wantLabel: "CUE evaluation passed",
		},
		{
			name:       "short label with detail",
			label:      "File exists",
			detail:     "/path/to/file",
			wantLabel:  "File exists",
			wantDetail: "/path/to/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatVetCheck(tt.label, tt.detail)

			// The rendered string contains ANSI codes. Strip them for content checks.
			assert.Contains(t, result, "✔", "should contain checkmark")
			assert.Contains(t, result, tt.wantLabel, "should contain label")

			if tt.detail != "" {
				assert.Contains(t, result, tt.wantDetail, "should contain detail")
			} else {
				// No detail means no trailing whitespace beyond the label
				stripped := stripAnsi(result)
				assert.False(t, strings.HasSuffix(stripped, " "), "should not have trailing whitespace when detail is empty")
			}
		})
	}

	t.Run("alignment consistency", func(t *testing.T) {
		// Multiple check lines with different label lengths should have
		// detail text starting at the same column position.
		line1 := FormatVetCheck("Config file found", "~/.opm/config.cue")
		line2 := FormatVetCheck("Module metadata found", "~/.opm/cue.mod/module.cue")

		stripped1 := stripAnsi(line1)
		stripped2 := stripAnsi(line2)

		// Find the position of the detail text in stripped output.
		idx1 := strings.Index(stripped1, "~/.opm/config.cue")
		idx2 := strings.Index(stripped2, "~/.opm/cue.mod/module.cue")

		assert.Equal(t, idx1, idx2, "detail text should align to same column")
	})
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

func TestFormatFQN(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "provider separator replaced",
			input: "kubernetes#opmodel.dev/transformers@v0#Foo",
			want:  "kubernetes - opmodel.dev/transformers@v0#Foo",
		},
		{
			name:  "no hash unchanged",
			input: "nohash",
			want:  "nohash",
		},
		{
			name:  "only one hash",
			input: "prov#name",
			want:  "prov - name",
		},
		{
			name:  "multiple hashes only first replaced",
			input: "prov#path@v0#Name",
			want:  "prov - path@v0#Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFQN(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFormatTransformerMatch(t *testing.T) {
	tests := []struct {
		name      string
		component string
		fqn       string
	}{
		{
			name:      "basic match",
			component: "web",
			fqn:       "kubernetes#opmodel.dev/transformers@v0#DeploymentTransformer",
		},
		{
			name:      "short names",
			component: "db",
			fqn:       "test#Foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTransformerMatch(tt.component, tt.fqn)
			stripped := stripAnsi(result)

			assert.Contains(t, stripped, "▸", "should contain bullet")
			assert.Contains(t, stripped, tt.component, "should contain component name")
			assert.Contains(t, stripped, "←", "should contain arrow")
			assert.Contains(t, stripped, FormatFQN(tt.fqn), "should contain formatted FQN")
		})
	}
}

func TestFormatTransformerMatchVerbose(t *testing.T) {
	t.Run("with reason", func(t *testing.T) {
		result := FormatTransformerMatchVerbose("web", "kubernetes#opmodel.dev/t@v0#Foo", "Matched: requiredResources[Container]")
		stripped := stripAnsi(result)

		assert.Contains(t, result, "\n", "should contain newline")
		assert.Contains(t, stripped, "▸", "should contain bullet")
		assert.Contains(t, stripped, "web", "should contain component")
		assert.Contains(t, stripped, "Matched: requiredResources[Container]", "should contain reason")

		lines := strings.Split(stripped, "\n")
		assert.Len(t, lines, 2, "should have exactly 2 lines")
		assert.True(t, strings.HasPrefix(lines[1], "     "), "reason line should be indented")
	})

	t.Run("empty reason", func(t *testing.T) {
		component := "web"
		fqn := "kubernetes#opmodel.dev/t@v0#Foo"

		resultVerbose := FormatTransformerMatchVerbose(component, fqn, "")
		resultBasic := FormatTransformerMatch(component, fqn)

		assert.Equal(t, resultBasic, resultVerbose, "empty reason should return same as basic format")
		assert.NotContains(t, resultVerbose, "\n", "should not contain newline when reason is empty")
	})
}

func TestFormatTransformerUnmatched(t *testing.T) {
	result := FormatTransformerUnmatched("orphan-component")
	stripped := stripAnsi(result)

	assert.Contains(t, stripped, "▸", "should contain bullet")
	assert.Contains(t, stripped, "orphan-component", "should contain component name")
	assert.Contains(t, stripped, "(no matching transformer)", "should contain unmatched message")
}
