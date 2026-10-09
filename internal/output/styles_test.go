package output

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatHealthStatus(t *testing.T) {
	tests := []struct {
		status   string
		contains string
	}{
		{"Ready", "Ready"},
		{"Complete", "Complete"},
		{"Applied", "Applied"},
		{"NotReady", "NotReady"},
		{"Missing", "Missing"},
		{"Unknown", "Unknown"},
		{"", ""},
		{"other", "other"},
	}
	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			result := FormatHealthStatus(tc.status)
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestFormatComponent(t *testing.T) {
	t.Run("empty returns dash", func(t *testing.T) {
		assert.Equal(t, "-", FormatComponent(""))
	})
	t.Run("non-empty renders name", func(t *testing.T) {
		result := FormatComponent("server")
		assert.Contains(t, result, "server")
	})
}

func TestFormatPodPhase(t *testing.T) {
	tests := []struct {
		phase    string
		ready    bool
		contains string
	}{
		// ready=true always green regardless of phase name
		{"Running", true, "Running"},
		{"Pending", true, "Pending"},
		// green: succeeded
		{"Succeeded", false, "Succeeded"},
		// yellow: transitional phases (not ready)
		{"Running", false, "Running"},
		{"Pending", false, "Pending"},
		{"ContainerCreating", false, "ContainerCreating"},
		{"PodInitializing", false, "PodInitializing"},
		{"Unknown", false, "Unknown"},
		// red: error states (not ready, not transitional)
		{"CrashLoop", false, "CrashLoop"},
		{"Failed", false, "Failed"},
		{"ImagePullBackOff", false, "ImagePullBackOff"},
		{"ErrImagePull", false, "ErrImagePull"},
	}
	for _, tc := range tests {
		t.Run(tc.phase+"/ready="+fmt.Sprintf("%v", tc.ready), func(t *testing.T) {
			result := FormatPodPhase(tc.phase, tc.ready)
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestFormatReadyRatio(t *testing.T) {
	tests := []struct {
		ready    int
		total    int
		contains string
	}{
		{3, 3, "3/3"}, // all ready
		{1, 3, "1/3"}, // partial
		{0, 3, "0/3"}, // none ready
		{0, 0, "0/0"}, // edge: both zero
	}
	for _, tc := range tests {
		t.Run(tc.contains, func(t *testing.T) {
			result := FormatReadyRatio(tc.ready, tc.total)
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestFormatRestartCount(t *testing.T) {
	tests := []struct {
		count int
		text  string
	}{
		{1, ", 1 restarts"},   // yellow
		{9, ", 9 restarts"},   // yellow (boundary)
		{10, ", 10 restarts"}, // red (threshold)
		{22, ", 22 restarts"}, // red
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d", tc.count), func(t *testing.T) {
			result := FormatRestartCount(tc.count, tc.text)
			assert.Contains(t, result, tc.text)
		})
	}
}

func TestFormatEventType(t *testing.T) {
	assert.Contains(t, FormatEventType("Warning"), "Warning")
	assert.Contains(t, FormatEventType("Normal"), "Normal")
	assert.Equal(t, "Custom", FormatEventType("Custom"))
}

func TestFormatEventResource(t *testing.T) {
	styled := FormatEventResource("Pod", "api-0")
	assert.Contains(t, styled, "Pod/api-0")
}

func TestFormatResourceLine_LeftBehind(t *testing.T) {
	line := FormatResourceLine("Namespace", "", "apps", StatusLeftBehind)
	assert.Contains(t, line, "Namespace/apps")
	assert.Contains(t, line, "! left behind")
	assert.Equal(t, "!", statusIcon(StatusLeftBehind))
}

// A dry run's statuses that need attention carry the attention icon, so the
// meaning does not rest on color.
func TestFormatResourceLine_DryRunStatuses(t *testing.T) {
	for _, status := range []string{StatusWouldRefuse, StatusWouldSkip, StatusWouldKeep, StatusWouldLetGo, StatusCannotCheck} {
		assert.Equal(t, "!", statusIcon(status), status)
		line := FormatResourceLine("ConfigMap", "default", "settings", status)
		assert.Contains(t, line, "ConfigMap/default/settings")
		assert.Contains(t, line, "! "+status)
	}
}
