package render

import (
	"github.com/opmodel/cli/internal/cmdutil"
)

func printValidationError(err error) {
	if err == nil {
		return
	}
	cmdutil.PrintValidationError("render failed", err)
}
