package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cueerrors "cuelang.org/go/cue/errors"
	"github.com/charmbracelet/lipgloss"

	"github.com/opmodel/cli/internal/output"
)

// Lipgloss styles for CUE error output.
var (
	errStylePath     = lipgloss.NewStyle().Foreground(output.ColorCyan)
	errStyleDim      = lipgloss.NewStyle().Faint(true)
	errStylePosition = lipgloss.NewStyle().Foreground(output.ColorYellow)
)

// formatCUEDetails formats a CUE error into a multi-line, lipgloss-colorized string.
func formatCUEDetails(err error) string {
	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		cwd = ""
	}

	errs := cueerrors.Errors(err)
	if len(errs) == 0 {
		return err.Error()
	}

	errs = deduplicateCUEErrors(errs)

	var b strings.Builder
	for i, e := range errs {
		if i > 0 {
			b.WriteByte('\n')
		}

		if path := strings.Join(e.Path(), "."); path != "" {
			b.WriteString(errStylePath.Render(path))
			b.WriteString(": ")
		}

		b.WriteString(cueErrorMessage(e))

		positions := cueerrors.Positions(e)
		if len(positions) > 0 {
			b.WriteString(":")
		}
		for _, p := range positions {
			b.WriteByte('\n')
			pos := p.Position()
			filePath := cueRelPath(pos.Filename, cwd)

			b.WriteString("    ")
			b.WriteString(errStyleDim.Render("→"))
			b.WriteByte(' ')
			b.WriteString(errStyleDim.Render(filePath))
			if pos.IsValid() {
				if filePath != "" {
					b.WriteString(errStyleDim.Render(":"))
				}
				b.WriteString(errStylePosition.Render(fmt.Sprintf("%d:%d", pos.Line, pos.Column)))
			}
		}
	}

	return b.String()
}

func cueErrorMessage(e cueerrors.Error) string {
	var parts []string
	var current error = e

	for current != nil {
		cueErr, ok := current.(cueerrors.Error) //nolint:errorlint // intentional: loop manually unwraps CUE error chain
		if !ok {
			parts = append(parts, current.Error())
			break
		}

		format, args := cueErr.Msg()
		if format != "" {
			parts = append(parts, fmt.Sprintf(format, args...))
		}

		current = cueerrors.Unwrap(current)
	}

	return strings.Join(parts, ": ")
}

func cueRelPath(path, cwd string) string {
	if cwd == "" || path == "" {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	if !strings.HasPrefix(rel, ".") {
		rel = "." + string(filepath.Separator) + rel
	}
	return rel
}

func deduplicateCUEErrors(errs []cueerrors.Error) []cueerrors.Error {
	if len(errs) <= 1 {
		return errs
	}

	result := make([]cueerrors.Error, len(errs))
	copy(result, errs)

	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && compareCUEErrors(result[j-1], result[j]) > 0; j-- {
			result[j-1], result[j] = result[j], result[j-1]
		}
	}

	deduped := result[:1]
	for _, e := range result[1:] {
		prev := deduped[len(deduped)-1]
		if !approximateEqualCUE(prev, e) {
			deduped = append(deduped, e)
		}
	}

	return deduped
}

func compareCUEErrors(a, b cueerrors.Error) int {
	aPos := a.Position()
	bPos := b.Position()

	if !aPos.IsValid() && bPos.IsValid() {
		return -1
	}
	if aPos.IsValid() && !bPos.IsValid() {
		return 1
	}
	if aPos.IsValid() && bPos.IsValid() {
		if c := aPos.Compare(bPos); c != 0 {
			return c
		}
	}

	aPath := a.Path()
	bPath := b.Path()
	minLen := len(aPath)
	if len(bPath) < minLen {
		minLen = len(bPath)
	}
	for i := 0; i < minLen; i++ {
		if aPath[i] < bPath[i] {
			return -1
		}
		if aPath[i] > bPath[i] {
			return 1
		}
	}
	if len(aPath) != len(bPath) {
		if len(aPath) < len(bPath) {
			return -1
		}
		return 1
	}

	if a.Error() < b.Error() {
		return -1
	}
	if a.Error() > b.Error() {
		return 1
	}
	return 0
}

func approximateEqualCUE(a, b cueerrors.Error) bool {
	aPos := a.Position()
	bPos := b.Position()
	if !aPos.IsValid() || !bPos.IsValid() {
		return a.Error() == b.Error()
	}
	aPath := a.Path()
	bPath := b.Path()
	if len(aPath) != len(bPath) {
		return false
	}
	for i := range aPath {
		if aPath[i] != bPath[i] {
			return false
		}
	}
	return aPos.Compare(bPos) == 0
}
