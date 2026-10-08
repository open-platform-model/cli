package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// CheckOutputFormat refuses a --output value that is not one of valid, with
// the message and the exit code the other commands use for that mistake.
func CheckOutputFormat(value string, valid ...string) error {
	if slices.Contains(valid, value) {
		return nil
	}
	return &opmexit.ExitError{
		Code: opmexit.ExitGeneralError,
		Err:  fmt.Errorf("invalid output format %q (valid: %s)", value, strings.Join(valid, ", ")),
	}
}

// WriteStructured writes v to w as indented JSON or as YAML. format is
// "json" or "yaml"; the caller validated it with CheckOutputFormat.
func WriteStructured(w io.Writer, format string, v any) error {
	var (
		data []byte
		err  error
	)
	switch format {
	case "json":
		data, err = json.MarshalIndent(v, "", "  ")
		data = append(data, '\n')
	case "yaml":
		data, err = yaml.Marshal(v)
	default:
		return fmt.Errorf("no structured writer for output format %q", format)
	}
	if err != nil {
		return fmt.Errorf("encoding %s output: %w", format, err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}
