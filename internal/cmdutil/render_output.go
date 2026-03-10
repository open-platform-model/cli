package cmdutil

import "github.com/opmodel/cli/internal/output"

// ShowOutputOpts controls how ShowRenderOutput displays results.
type ShowOutputOpts struct {
	Verbose bool
}

// ShowRenderOutput shows transformer match output and logs warnings.
func ShowRenderOutput(result *RenderResult, opts ShowOutputOpts) {
	switch {
	case opts.Verbose:
		WriteVerboseMatchLog(result)
	default:
		WriteTransformerMatches(result)
	}

	releaseLog := output.ReleaseLogger(result.Release.Name)
	for _, w := range result.Warnings {
		releaseLog.Warn(w)
	}
}
