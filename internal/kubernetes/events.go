package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/charmbracelet/lipgloss"
	"github.com/opmodel/cli/internal/output"
)

// EventsOptions configures an events query.
type EventsOptions struct {
	// Namespace is the target namespace for events.
	Namespace string

	// ReleaseName identifies the release (for structured output metadata).
	ReleaseName string

	// ReleaseID identifies the release by ID (for structured output metadata).
	ReleaseID string

	// Since is the time window cutoff. Only events with lastTimestamp after this are returned.
	Since time.Time

	// EventType filters events by type ("Normal", "Warning"). Empty means all.
	EventType string

	// OutputFormat selects the output serialization.
	OutputFormat output.Format

	// InventoryLive is the list of live OPM-managed resources from inventory resolution.
	InventoryLive []*unstructured.Unstructured
}

// EventsResult is the structured output for JSON/YAML serialization.
type EventsResult struct {
	ReleaseName string       `json:"releaseName,omitempty" yaml:"releaseName,omitempty"`
	ReleaseID   string       `json:"releaseId,omitempty" yaml:"releaseId,omitempty"`
	Namespace   string       `json:"namespace" yaml:"namespace"`
	Events      []EventEntry `json:"events" yaml:"events"`
}

// EventEntry represents a single Kubernetes event in structured output.
type EventEntry struct {
	LastSeen  string `json:"lastSeen" yaml:"lastSeen"`
	Type      string `json:"type" yaml:"type"`
	Kind      string `json:"kind" yaml:"kind"`
	Name      string `json:"name" yaml:"name"`
	Reason    string `json:"reason" yaml:"reason"`
	Message   string `json:"message" yaml:"message"`
	Count     int32  `json:"count" yaml:"count"`
	FirstSeen string `json:"firstSeen,omitempty" yaml:"firstSeen,omitempty"`
}

// GetModuleEvents collects Kubernetes events for all resources belonging to a release,
// including events from Kubernetes-owned children (Pods, ReplicaSets).
//
// Flow: discover children → collect UIDs → bulk fetch events → filter → sort.
func GetModuleEvents(ctx context.Context, client *Client, opts EventsOptions) (*EventsResult, error) {
	if len(opts.InventoryLive) == 0 {
		return nil, &noResourcesFoundError{
			ReleaseName: opts.ReleaseName,
			Namespace:   opts.Namespace,
		}
	}

	// Step 1: Discover children of workload resources.
	children, err := DiscoverChildren(ctx, client, opts.InventoryLive, opts.Namespace)
	if err != nil {
		return nil, fmt.Errorf("discovering children: %w", err)
	}

	// Step 2: Collect all UIDs (parents + children).
	uidSet := make(map[types.UID]bool)
	for _, res := range opts.InventoryLive {
		uidSet[res.GetUID()] = true
	}
	for _, child := range children {
		uidSet[child.GetUID()] = true
	}

	// Step 3: Bulk fetch all events in the namespace.
	eventList, err := client.Clientset.CoreV1().Events(opts.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}

	// Step 4: Filter by UID, since, and type.
	var filtered []corev1.Event
	for i := range eventList.Items {
		ev := &eventList.Items[i]

		// Filter by UID.
		if !uidSet[ev.InvolvedObject.UID] {
			continue
		}

		// Filter by since.
		evTime := eventLastTimestamp(ev)
		if !opts.Since.IsZero() && evTime.Before(opts.Since) {
			continue
		}

		// Filter by type.
		if opts.EventType != "" && ev.Type != opts.EventType {
			continue
		}

		filtered = append(filtered, *ev)
	}

	// Step 5: Sort by lastTimestamp ascending (oldest first).
	sort.Slice(filtered, func(i, j int) bool {
		ti := eventLastTimestamp(&filtered[i])
		tj := eventLastTimestamp(&filtered[j])
		return ti.Before(tj)
	})

	// Build result.
	result := &EventsResult{
		ReleaseName: opts.ReleaseName,
		ReleaseID:   opts.ReleaseID,
		Namespace:   opts.Namespace,
		Events:      make([]EventEntry, 0, len(filtered)),
	}

	for i := range filtered {
		ev := &filtered[i]
		entry := EventEntry{
			LastSeen: eventLastTimestamp(ev).Format(time.RFC3339),
			Type:     ev.Type,
			Kind:     ev.InvolvedObject.Kind,
			Name:     ev.InvolvedObject.Name,
			Reason:   ev.Reason,
			Message:  ev.Message,
			Count:    ev.Count,
		}
		if !ev.FirstTimestamp.IsZero() {
			entry.FirstSeen = ev.FirstTimestamp.Format(time.RFC3339)
		}
		result.Events = append(result.Events, entry)
	}

	return result, nil
}

// eventLastTimestamp returns the best "last seen" time for an event.
// It prefers LastTimestamp; falls back to EventTime, then CreationTimestamp.
func eventLastTimestamp(ev *corev1.Event) time.Time {
	if !ev.LastTimestamp.IsZero() {
		return ev.LastTimestamp.Time
	}
	if !ev.EventTime.IsZero() {
		return ev.EventTime.Time
	}
	return ev.CreationTimestamp.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// Since parsing
// ─────────────────────────────────────────────────────────────────────────────

// dayRegex matches a leading "Nd" component in a duration string.
var dayRegex = regexp.MustCompile(`^(\d+)d(.*)$`)

// ParseSince converts a --since flag value to a time.Time cutoff.
// Supports Go duration syntax (30m, 1h, 2h30m) plus "d" for days.
// Examples: "30m", "1h", "2h30m", "1d", "7d".
func ParseSince(since string) (time.Time, error) {
	if since == "" {
		return time.Time{}, nil
	}

	d, err := parseDurationWithDays(since)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --since value %q: %w", since, err)
	}

	return time.Now().Add(-d), nil
}

// parseDurationWithDays extends time.ParseDuration with support for "d" (days).
// It preprocesses day components into hours before delegating to time.ParseDuration.
func parseDurationWithDays(s string) (time.Duration, error) {
	if matches := dayRegex.FindStringSubmatch(s); matches != nil {
		days, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}
		remainder := matches[2]
		totalHours := time.Duration(days) * 24 * time.Hour
		if remainder == "" {
			return totalHours, nil
		}
		extra, err := time.ParseDuration(remainder)
		if err != nil {
			return 0, err
		}
		return totalHours + extra, nil
	}
	return time.ParseDuration(s)
}

// ─────────────────────────────────────────────────────────────────────────────
// Formatting
// ─────────────────────────────────────────────────────────────────────────────

// FormatEvents formats an EventsResult according to the requested output format.
func FormatEvents(result *EventsResult, format output.Format) (string, error) {
	switch format {
	case output.FormatJSON:
		return FormatEventsJSON(result)
	case output.FormatYAML:
		return FormatEventsYAML(result)
	case output.FormatTable, output.FormatWide, output.FormatDir:
		return FormatEventsTable(result), nil
	default:
		return FormatEventsTable(result), nil
	}
}

// FormatEventsTable renders events as a kubectl-style table with color coding.
//
// Columns: LAST SEEN, TYPE, RESOURCE, REASON, MESSAGE
// Colors: Warning=yellow, Normal=dim, Resource=cyan
func FormatEventsTable(result *EventsResult) string {
	if len(result.Events) == 0 {
		return "No events found.\n"
	}

	tbl := output.NewTable("LAST SEEN", "TYPE", "RESOURCE", "REASON", "MESSAGE")

	warningStyle := lipgloss.NewStyle().Foreground(output.ColorYellow)

	for _, ev := range result.Events {
		// Parse the RFC3339 lastSeen to compute relative duration.
		lastSeen := ev.LastSeen
		if t, err := time.Parse(time.RFC3339, ev.LastSeen); err == nil {
			lastSeen = formatDuration(time.Since(t))
		}

		// Color the TYPE column.
		typeStr := ev.Type
		switch ev.Type {
		case "Warning":
			typeStr = warningStyle.Render(ev.Type)
		case "Normal":
			typeStr = output.Dim(ev.Type)
		}

		// Color the RESOURCE column (cyan).
		resource := output.StyleNoun(ev.Kind + "/" + ev.Name)

		tbl.Row(lastSeen, typeStr, resource, ev.Reason, ev.Message)
	}

	return tbl.String()
}

// FormatEventsJSON serializes an EventsResult to indented JSON.
func FormatEventsJSON(result *EventsResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling events to JSON: %w", err)
	}
	return string(data), nil
}

// FormatEventsYAML serializes an EventsResult to YAML.
func FormatEventsYAML(result *EventsResult) (string, error) {
	data, err := yaml.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshaling events to YAML: %w", err)
	}
	return string(data), nil
}

// FormatSingleEventLine formats a single event for watch mode streaming output.
// Uses the same table renderer as FormatEventsTable to ensure ANSI-aware column alignment.
func FormatSingleEventLine(ev *corev1.Event) string {
	lastSeen := formatDuration(time.Since(eventLastTimestamp(ev)))

	warningStyle := lipgloss.NewStyle().Foreground(output.ColorYellow)

	typeStr := ev.Type
	switch ev.Type {
	case "Warning":
		typeStr = warningStyle.Render(ev.Type)
	case "Normal":
		typeStr = output.Dim(ev.Type)
	}

	resource := output.StyleNoun(ev.InvolvedObject.Kind + "/" + ev.InvolvedObject.Name)

	// Use the same table renderer for ANSI-aware column alignment.
	tbl := output.NewTable("LAST SEEN", "TYPE", "RESOURCE", "REASON", "MESSAGE")
	tbl.Row(lastSeen, typeStr, resource, ev.Reason, ev.Message)
	// The table String() includes a header + data row; for streaming watch mode
	// we only want the data row. Split and return the second line.
	lines := strings.SplitN(tbl.String(), "\n", 3)
	if len(lines) >= 2 {
		return lines[1]
	}
	return tbl.String()
}
