package render

import (
	"github.com/opmodel/cli/internal/cmdutil"
)

func printValidationError(prefix string, err error) {
	if err == nil {
		return
	}
	cmdutil.PrintValidationError(prefix, err)
}
