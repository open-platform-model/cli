package render

import (
	"github.com/opmodel/cli/internal/output"
)

func showOutput(result *Result, opts ShowOutputOpts) {
	switch {
	case opts.Verbose:
		writeVerboseMatchLog(result)
	default:
		writeTransformerMatches(result)
	}

	instanceLog := output.InstanceLogger(result.Instance.Name)
	for _, w := range result.Warnings {
		instanceLog.Warn(w)
	}
}
