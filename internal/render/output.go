package render

import (
	"fmt"
	"sort"
	"time"
)

// aggregateResults performs Phase 5: Result aggregation.
// Implements deterministic sorting (FR-023) and fail-on-end error aggregation (FR-024).
func aggregateResults(results []Result, unmatchedErrors []error) (PhaseRecord, []error, []Manifest, error) {
	phase5Start := time.Now()
	var phase5Steps []PhaseStep

	var errors []error
	errors = append(errors, unmatchedErrors...)

	collectionStart := time.Now()
	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, result.Error)
		}
	}
	collectionDuration := time.Since(collectionStart)
	phase5Steps = append(phase5Steps, PhaseStep{Name: "Collect results", Duration: collectionDuration})

	// Sort results for deterministic output (FR-023)
	sort.Slice(results, func(i, j int) bool {
		if results[i].TransformerID != results[j].TransformerID {
			return results[i].TransformerID < results[j].TransformerID
		}
		return results[i].ComponentName < results[j].ComponentName
	})

	// Build manifest list
	var manifests []Manifest
	for _, r := range results {
		if r.Error == nil {
			manifests = append(manifests, Manifest{
				Object:        r.Output,
				TransformerID: r.TransformerID,
				ComponentName: r.ComponentName,
			})
		}
	}

	successCount := len(manifests)

	record := PhaseRecord{
		Name:     "Aggregation & Output",
		Duration: time.Since(phase5Start),
		Steps:    phase5Steps,
		Details:  fmt.Sprintf("%d resources, %d errors", successCount, len(errors)),
	}

	return record, errors, manifests, nil
}

// formatDuration returns a concise duration string.
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// printTimingSummary outputs an ASCII table with phase timing details.
func printTimingSummary(phases []PhaseRecord) {
	fmt.Println("")
	fmt.Println("╭──────────────────────────────┬──────────┬───────────────────────────────────────────────╮")
	fmt.Println("│ Phase                        │ Duration │ Details                                       │")
	fmt.Println("├──────────────────────────────┼──────────┼───────────────────────────────────────────────┤")

	var totalDuration time.Duration
	for i, phase := range phases {
		totalDuration += phase.Duration

		// Format duration (keep it concise)
		durStr := formatDuration(phase.Duration)

		// Build details string (include sub-steps if any)
		details := phase.Details
		if len(phase.Steps) > 0 {
			stepStrs := make([]string, len(phase.Steps))
			for j, step := range phase.Steps {
				stepStrs[j] = fmt.Sprintf("%s: %s", step.Name, formatDuration(step.Duration))
			}
			if details != "" {
				details = fmt.Sprintf("%s | %s", details, stepStrs[0])
				if len(stepStrs) > 1 {
					details = fmt.Sprintf("%s, %s", details, stepStrs[1])
				}
			} else {
				details = stepStrs[0]
				if len(stepStrs) > 1 {
					details = fmt.Sprintf("%s, %s", details, stepStrs[1])
				}
			}
		}

		// Truncate details if too long
		if len(details) > 45 {
			details = details[:42] + "..."
		}

		fmt.Printf("│ %d. %-25s │ %8s │ %-45s │\n", i+1, phase.Name, durStr, details)
	}

	fmt.Println("├──────────────────────────────┼──────────┼───────────────────────────────────────────────┤")
	fmt.Printf("│ %-28s │ %8s │ %-45s │\n", "Total", formatDuration(totalDuration), "Pipeline complete")
	fmt.Println("╰──────────────────────────────┴──────────┴───────────────────────────────────────────────╯")
}
