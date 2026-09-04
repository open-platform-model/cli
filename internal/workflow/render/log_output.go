package render

import (
	"github.com/open-platform-model/cli/internal/output"
)

// writeTransformerMatches is the default render report: one line per matched
// (component, transformer) pair, in the build's order.
func writeTransformerMatches(result *Result) {
	instanceLog := output.InstanceLogger(result.Instance.Name)
	for _, pair := range result.Pairs {
		instanceLog.Info(output.FormatTransformerMatch(pair.Component, pair.Transformer))
	}
}

// writeVerboseMatchLog is the verbose render report: the instance identity,
// every matched pair, then one line per rendered resource. Unmatched
// components never reach a successful Result: the kernel's fail-closed gate
// refuses the render and the diagnostics print with the error instead.
func writeVerboseMatchLog(result *Result) {
	instanceLog := output.InstanceLogger(result.Instance.Name)
	instanceLog.Info("instance", "namespace", result.Instance.Namespace, "version", result.Module.Version)

	for _, pair := range result.Pairs {
		instanceLog.Info(output.FormatTransformerMatch(pair.Component, pair.Transformer))
	}

	for _, res := range result.Resources {
		instanceLog.Info(output.FormatResourceLine(res.GetKind(), res.GetNamespace(), res.GetName(), output.StatusValid))
	}
}
