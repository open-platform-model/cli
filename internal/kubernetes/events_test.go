package kubernetes

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/opmodel/cli/internal/output"
)

func TestParseSince(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "30 minutes", input: "30m"},
		{name: "1 hour", input: "1h"},
		{name: "2h30m", input: "2h30m"},
		{name: "1 day", input: "1d"},
		{name: "7 days", input: "7d"},
		{name: "1d12h", input: "1d12h"},
		{name: "empty string", input: ""},
		{name: "invalid", input: "foo", wantErr: true},
		{name: "invalid day", input: "xd", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSince(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.input == "" {
				assert.True(t, result.IsZero())
			} else {
				assert.True(t, result.Before(time.Now()))
			}
		})
	}
}

func TestGetModuleEvents(t *testing.T) {
	now := time.Now()

	// Create fake resources.
	parentRes := makeParent("Deployment", "web", "default", "uid-deploy-1")

	// Create fake events.
	events := []corev1.Event{
		{
			ObjectMeta:     metav1.ObjectMeta{Name: "ev1", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{UID: "uid-deploy-1", Kind: "Deployment", Name: "web"},
			Type:           "Normal",
			Reason:         "ScalingUp",
			Message:        "Scaled up",
			LastTimestamp:  metav1.NewTime(now.Add(-10 * time.Minute)),
			Count:          1,
		},
		{
			ObjectMeta:     metav1.ObjectMeta{Name: "ev2", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{UID: "uid-deploy-1", Kind: "Deployment", Name: "web"},
			Type:           "Warning",
			Reason:         "FailedCreate",
			Message:        "Error creating",
			LastTimestamp:  metav1.NewTime(now.Add(-5 * time.Minute)),
			Count:          3,
		},
		{
			ObjectMeta:     metav1.ObjectMeta{Name: "ev3", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{UID: "uid-unrelated", Kind: "Pod", Name: "other"},
			Type:           "Normal",
			Reason:         "Pulled",
			Message:        "Image pulled",
			LastTimestamp:  metav1.NewTime(now.Add(-2 * time.Minute)),
			Count:          1,
		},
		{
			ObjectMeta:     metav1.ObjectMeta{Name: "ev4", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{UID: "uid-deploy-1", Kind: "Deployment", Name: "web"},
			Type:           "Normal",
			Reason:         "OldEvent",
			Message:        "Very old",
			LastTimestamp:  metav1.NewTime(now.Add(-3 * time.Hour)),
			Count:          1,
		},
	}

	clientset := fake.NewSimpleClientset() //nolint:staticcheck // matches existing test patterns
	for i := range events {
		_, err := clientset.CoreV1().Events("default").Create(context.Background(), &events[i], metav1.CreateOptions{})
		require.NoError(t, err)
	}

	client := &Client{Clientset: clientset}

	t.Run("UID filtering", func(t *testing.T) {
		result, err := GetModuleEvents(context.Background(), client, EventsOptions{
			Namespace:     "default",
			ReleaseName:   "test",
			InventoryLive: []*unstructured.Unstructured{parentRes},
		})
		require.NoError(t, err)
		// Should include ev1, ev2, ev4 (all uid-deploy-1), not ev3 (uid-unrelated).
		assert.Len(t, result.Events, 3)
		for _, ev := range result.Events {
			assert.Equal(t, "web", ev.Name)
		}
	})

	t.Run("since filtering", func(t *testing.T) {
		since := now.Add(-30 * time.Minute)
		result, err := GetModuleEvents(context.Background(), client, EventsOptions{
			Namespace:     "default",
			ReleaseName:   "test",
			Since:         since,
			InventoryLive: []*unstructured.Unstructured{parentRes},
		})
		require.NoError(t, err)
		// Should exclude ev4 (3 hours old).
		assert.Len(t, result.Events, 2)
	})

	t.Run("type filtering", func(t *testing.T) {
		result, err := GetModuleEvents(context.Background(), client, EventsOptions{
			Namespace:     "default",
			ReleaseName:   "test",
			EventType:     "Warning",
			InventoryLive: []*unstructured.Unstructured{parentRes},
		})
		require.NoError(t, err)
		assert.Len(t, result.Events, 1)
		assert.Equal(t, "Warning", result.Events[0].Type)
	})

	t.Run("sort order ascending", func(t *testing.T) {
		result, err := GetModuleEvents(context.Background(), client, EventsOptions{
			Namespace:     "default",
			ReleaseName:   "test",
			InventoryLive: []*unstructured.Unstructured{parentRes},
		})
		require.NoError(t, err)
		require.Len(t, result.Events, 3)
		// Oldest first: ev4 (3h ago), ev1 (10m ago), ev2 (5m ago)
		assert.Equal(t, "OldEvent", result.Events[0].Reason)
		assert.Equal(t, "ScalingUp", result.Events[1].Reason)
		assert.Equal(t, "FailedCreate", result.Events[2].Reason)
	})

	t.Run("empty results", func(t *testing.T) {
		noMatchRes := makeParent("ConfigMap", "cfg", "default", "uid-no-match")
		result, err := GetModuleEvents(context.Background(), client, EventsOptions{
			Namespace:     "default",
			ReleaseName:   "test",
			InventoryLive: []*unstructured.Unstructured{noMatchRes},
		})
		require.NoError(t, err)
		assert.Empty(t, result.Events)
	})

	t.Run("no resources returns error", func(t *testing.T) {
		_, err := GetModuleEvents(context.Background(), client, EventsOptions{
			Namespace:     "default",
			ReleaseName:   "test",
			InventoryLive: nil,
		})
		assert.Error(t, err)
		assert.True(t, IsNoResourcesFound(err))
	})
}

func TestFormatEventsTable(t *testing.T) {
	result := &EventsResult{
		Namespace: "default",
		Events: []EventEntry{
			{LastSeen: time.Now().Add(-5 * time.Minute).Format(time.RFC3339), Type: "Warning", Kind: "Pod", Name: "web-x1", Reason: "OOMKilled", Message: "Container OOM"},
			{LastSeen: time.Now().Add(-10 * time.Minute).Format(time.RFC3339), Type: "Normal", Kind: "Deployment", Name: "web", Reason: "ScalingUp", Message: "Scaled up"},
		},
	}

	table := FormatEventsTable(result)
	assert.Contains(t, table, "LAST SEEN")
	assert.Contains(t, table, "TYPE")
	assert.Contains(t, table, "RESOURCE")
	assert.Contains(t, table, "REASON")
	assert.Contains(t, table, "MESSAGE")
	assert.Contains(t, table, "OOMKilled")
	assert.Contains(t, table, "ScalingUp")
}

func TestFormatEventsTable_Empty(t *testing.T) {
	result := &EventsResult{Events: []EventEntry{}}
	table := FormatEventsTable(result)
	assert.Contains(t, table, "No events found")
}

func TestFormatEventsJSON(t *testing.T) {
	result := &EventsResult{
		ReleaseName: "test",
		Namespace:   "default",
		Events: []EventEntry{
			{LastSeen: "2025-01-01T00:00:00Z", Type: "Normal", Kind: "Pod", Name: "web-x1", Reason: "Pulled", Message: "Image pulled", Count: 1},
		},
	}

	jsonStr, err := FormatEventsJSON(result)
	require.NoError(t, err)

	var parsed EventsResult
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "test", parsed.ReleaseName)
	assert.Len(t, parsed.Events, 1)
}

func TestFormatEventsYAML(t *testing.T) {
	result := &EventsResult{
		ReleaseName: "test",
		Namespace:   "default",
		Events: []EventEntry{
			{LastSeen: "2025-01-01T00:00:00Z", Type: "Warning", Kind: "Pod", Name: "web-x1", Reason: "BackOff", Message: "Back-off", Count: 5},
		},
	}

	yamlStr, err := FormatEventsYAML(result)
	require.NoError(t, err)
	assert.Contains(t, yamlStr, "releaseName: test")
	assert.Contains(t, yamlStr, "BackOff")
}

func TestFormatEvents_Dispatcher(t *testing.T) {
	result := &EventsResult{Events: []EventEntry{}}

	_, err := FormatEvents(result, output.FormatTable)
	assert.NoError(t, err)

	_, err = FormatEvents(result, output.FormatJSON)
	assert.NoError(t, err)

	_, err = FormatEvents(result, output.FormatYAML)
	assert.NoError(t, err)
}
