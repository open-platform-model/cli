package render

import (
	"fmt"
	"strings"

	"github.com/opmodel/cli/internal/output"
)

func printValidationError(prefix string, err error) {
	if err == nil {
		return
	}
	for _, line := range strings.Split(err.Error(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		output.Error(fmt.Sprintf("%s: %s", prefix, line))
	}
}
