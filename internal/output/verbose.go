package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// VerboseOptions controls verbose output.
type VerboseOptions struct {
	// JSON outputs structured JSON instead of human-readable text
	JSON bool
	// Writer is the output destination
	Writer io.Writer
}

// verboseResult is the structured verbose output.
type verboseResult struct {
	Module    verboseModule     `json:"module"`
	MatchPlan verboseMatchPlan  `json:"matchPlan"`
	Resources []verboseResource `json:"resources"`
	Errors    []string          `json:"errors,omitempty"`
	Warnings  []string          `json:"warnings,omitempty"`
}

// verboseModule contains module metadata for verbose output.
type verboseModule struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Version    string            `json:"version"`
	Components []string          `json:"components"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// verboseMatchPlan contains matching details for verbose output.
type verboseMatchPlan struct {
	Matches   map[string][]verboseMatch `json:"matches"`
	Unmatched []string                  `json:"unmatched,omitempty"`
	Details   []verboseMatchDetail      `json:"details,omitempty"`
}

// verboseMatch describes a transformer match.
type verboseMatch struct {
	Transformer string `json:"transformer"`
	Reason      string `json:"reason"`
}

// verboseMatchDetail provides detailed matching information.
type verboseMatchDetail struct {
	Component        string   `json:"component"`
	Transformer      string   `json:"transformer"`
	Matched          bool     `json:"matched"`
	MissingLabels    []string `json:"missingLabels,omitempty"`
	MissingResources []string `json:"missingResources,omitempty"`
	MissingTraits    []string `json:"missingTraits,omitempty"`
	Reason           string   `json:"reason"`
}

// verboseResource describes a rendered resource.
type verboseResource struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	Component   string `json:"component"`
	Transformer string `json:"transformer"`
}

// RenderResultInfo provides access to render result data without importing build.
type RenderResultInfo struct {
	ModuleName       string
	ModuleNamespace  string
	ModuleVersion    string
	ModuleComponents []string
	ModuleLabels     map[string]string
	Matches          map[string][]TransformerMatchInfo
	Unmatched        []string
	Resources        []ResourceInfo
	Errors           []error
	Warnings         []string
}

// TransformerMatchInfo provides transformer match data.
type TransformerMatchInfo struct {
	TransformerFQN string
	Reason         string
}

// matchDetailInfo provides match detail data.
type matchDetailInfo struct {
	ComponentName    string
	TransformerFQN   string
	Matched          bool
	MissingLabels    []string
	MissingResources []string
	MissingTraits    []string
	Reason           string
}

// WriteVerboseResult writes verbose output from a RenderResultInfo.
func WriteVerboseResult(result *RenderResultInfo, details []matchDetailInfo, opts VerboseOptions) error {
	verboseResult := buildVerboseResultFromInfo(result, details)

	if opts.JSON {
		return writeVerboseJSON(verboseResult, opts.Writer)
	}
	return writeVerboseHuman(verboseResult, opts.Writer)
}

// buildVerboseResultFromInfo constructs verbose result from info.
func buildVerboseResultFromInfo(result *RenderResultInfo, details []matchDetailInfo) *verboseResult {
	vr := &verboseResult{
		Module: verboseModule{
			Name:       result.ModuleName,
			Namespace:  result.ModuleNamespace,
			Version:    result.ModuleVersion,
			Components: result.ModuleComponents,
			Labels:     result.ModuleLabels,
		},
		MatchPlan: verboseMatchPlan{
			Matches:   make(map[string][]verboseMatch),
			Unmatched: result.Unmatched,
		},
		Resources: make([]verboseResource, 0, len(result.Resources)),
		Warnings:  result.Warnings,
	}

	// Convert matches
	for compName, matches := range result.Matches {
		for _, m := range matches {
			vr.MatchPlan.Matches[compName] = append(vr.MatchPlan.Matches[compName], verboseMatch{
				Transformer: m.TransformerFQN,
				Reason:      m.Reason,
			})
		}
	}

	// Convert match details
	for i := range details {
		d := &details[i]
		vr.MatchPlan.Details = append(vr.MatchPlan.Details, verboseMatchDetail{
			Component:        d.ComponentName,
			Transformer:      d.TransformerFQN,
			Matched:          d.Matched,
			MissingLabels:    d.MissingLabels,
			MissingResources: d.MissingResources,
			MissingTraits:    d.MissingTraits,
			Reason:           d.Reason,
		})
	}

	// Convert resources
	for _, res := range result.Resources {
		vr.Resources = append(vr.Resources, verboseResource{
			Kind:        res.GetKind(),
			Name:        res.GetName(),
			Namespace:   res.GetNamespace(),
			Component:   res.GetComponent(),
			Transformer: res.GetTransformer(),
		})
	}

	// Convert errors
	for _, err := range result.Errors {
		vr.Errors = append(vr.Errors, err.Error())
	}

	return vr
}

// writeVerboseJSON writes verbose output as JSON.
func writeVerboseJSON(result *verboseResult, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// writeVerboseHuman writes verbose output in human-readable format.
func writeVerboseHuman(result *verboseResult, w io.Writer) error {
	var sb strings.Builder

	// Module info
	sb.WriteString("Module:\n")
	sb.WriteString(fmt.Sprintf("  Name:      %s\n", result.Module.Name))
	sb.WriteString(fmt.Sprintf("  Namespace: %s\n", result.Module.Namespace))
	sb.WriteString(fmt.Sprintf("  Version:   %s\n", result.Module.Version))
	if len(result.Module.Components) > 0 {
		sb.WriteString(fmt.Sprintf("  Components: %s\n", strings.Join(result.Module.Components, ", ")))
	}
	sb.WriteString("\n")

	// Matching decisions
	sb.WriteString("Transformer Matching:\n")
	for compName, matches := range result.MatchPlan.Matches {
		sb.WriteString(fmt.Sprintf("  %s:\n", compName))
		for _, m := range matches {
			sb.WriteString(fmt.Sprintf("    ✓ %s\n", m.Transformer))
			if m.Reason != "" {
				sb.WriteString(fmt.Sprintf("      %s\n", m.Reason))
			}
		}
	}

	if len(result.MatchPlan.Unmatched) > 0 {
		sb.WriteString("  Unmatched components:\n")
		for _, comp := range result.MatchPlan.Unmatched {
			sb.WriteString(fmt.Sprintf("    ✗ %s\n", comp))
		}
	}
	sb.WriteString("\n")

	// Resources
	if len(result.Resources) > 0 {
		sb.WriteString("Generated Resources:\n")
		for _, res := range result.Resources {
			line := FormatResourceLine(res.Kind, res.Namespace, res.Name, StatusValid)
			sb.WriteString("  " + line + "\n")
		}
		sb.WriteString("\n")
	}

	// Warnings
	if len(result.Warnings) > 0 {
		sb.WriteString("Warnings:\n")
		for _, w := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  ⚠ %s\n", w))
		}
		sb.WriteString("\n")
	}

	// Errors
	if len(result.Errors) > 0 {
		sb.WriteString("Errors:\n")
		for _, e := range result.Errors {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", e))
		}
	}

	_, err := w.Write([]byte(sb.String()))
	return err
}
