package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/open-platform-model/cli/internal/output"
)

func formatFQNList(fqns []string) string {
	if len(fqns) == 0 {
		return ""
	}
	formatted := make([]string, len(fqns))
	for i, fqn := range fqns {
		formatted[i] = output.FormatFQN(fqn)
	}
	return strings.Join(formatted, ", ")
}

func writeTransformerMatches(result *Result) {
	if result.MatchPlan == nil {
		return
	}
	instanceLog := output.InstanceLogger(result.Instance.Name)
	for _, pair := range result.MatchPlan.MatchedPairs() {
		instanceLog.Info(output.FormatTransformerMatch(pair.ComponentName, pair.TransformerFQN))
	}
	for _, comp := range result.MatchPlan.Unmatched {
		instanceLog.Warn(output.FormatTransformerUnmatched(comp))
	}
}

func writeVerboseMatchLog(result *Result) {
	if result.MatchPlan == nil {
		return
	}
	instanceLog := output.InstanceLogger(result.Instance.Name)
	instanceLog.Info("instance", "namespace", result.Instance.Namespace, "version", result.Module.Version)

	for _, comp := range result.Components {
		attrs := []any{}
		if resources := formatFQNList(comp.ResourceFQNs); resources != "" {
			attrs = append(attrs, "resources", resources)
		}
		if traits := formatFQNList(comp.TraitFQNs); traits != "" {
			attrs = append(attrs, "traits", traits)
		}
		for k, v := range comp.Labels {
			attrs = append(attrs, k, v)
		}
		instanceLog.Info(fmt.Sprintf("component: %s", comp.Name), attrs...)
	}

	type matchLine struct {
		compName string
		tfFQN    string
		matched  bool
		missing  struct{ labels []string }
	}
	var lines []matchLine
	for _, p := range result.MatchPlan.MatchedPairs() {
		lines = append(lines, matchLine{compName: p.ComponentName, tfFQN: p.TransformerFQN, matched: true})
	}
	for _, p := range result.MatchPlan.NonMatchedPairs() {
		l := matchLine{compName: p.ComponentName, tfFQN: p.TransformerFQN, matched: false}
		// The kernel's match model matches on labels; per-FQN missing
		// resource/trait diagnostics live in MatchPlan.Missing.
		l.missing.labels = p.MissingLabels
		lines = append(lines, l)
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].compName != lines[j].compName {
			return lines[i].compName < lines[j].compName
		}
		return lines[i].tfFQN < lines[j].tfFQN
	})
	for _, l := range lines {
		if l.matched {
			instanceLog.Info(output.FormatTransformerMatch(l.compName, l.tfFQN))
		} else {
			attrs := []any{}
			if len(l.missing.labels) > 0 {
				attrs = append(attrs, "missing-labels", strings.Join(l.missing.labels, ", "))
			}
			instanceLog.Debug(output.FormatTransformerSkipped(l.compName, l.tfFQN), attrs...)
		}
	}

	for _, res := range result.Resources {
		instanceLog.Info(output.FormatResourceLine(res.GetKind(), res.GetNamespace(), res.GetName(), output.StatusValid))
	}
}
