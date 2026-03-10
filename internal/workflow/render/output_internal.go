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

	releaseLog := output.ReleaseLogger(result.Release.Name)
	for _, w := range result.Warnings {
		releaseLog.Warn(w)
	}
}
