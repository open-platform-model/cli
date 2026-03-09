package errors

import (
	"fmt"
	"path/filepath"
	"strings"

	cueerrors "cuelang.org/go/cue/errors"
)

// ConfigError is a structured validation error produced by the Bundle Gate or
// Module Gate when supplied values do not satisfy a #config schema.
//
// It carries the raw CUE error tree so callers can obtain either a human-readable
// summary (Error()) or structured per-field diagnostics (FieldErrors()).
type ConfigError struct {
	// Context is "bundle" or "module" — identifies which gate produced the error.
	Context string

	// Name is the release/bundle name for display (e.g. "my-game-stack", "server").
	Name string

	// RawError is the original CUE unification or concreteness error.
	// Preserved so FieldErrors() can walk cue/errors.Errors() for per-field output.
	RawError error
}

// Error implements the error interface.
// Produces a human-readable summary: one line per unique CUE error position.
func (e *ConfigError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %q: values do not satisfy #config:\n", e.Context, e.Name)

	for _, ce := range cueerrors.Errors(e.RawError) {
		pos := ce.Position()
		msg := cueerrors.Details(ce, nil)
		if pos.IsValid() {
			fmt.Fprintf(&sb, "  - %s: %s\n", pos, strings.TrimSpace(msg))
		} else {
			fmt.Fprintf(&sb, "  - %s\n", strings.TrimSpace(msg))
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// Unwrap returns the underlying CUE error for errors.Is/As compatibility.
func (e *ConfigError) Unwrap() error { return e.RawError }

// FieldErrors walks the raw CUE error tree and returns structured per-field
// diagnostics. Each entry contains the source file, line, column, field path,
// and a human-readable message — suitable for rich terminal output.
//
// Returns nil if RawError is nil or produces no parseable positions.
func (e *ConfigError) FieldErrors() []FieldError {
	if e.RawError == nil {
		return nil
	}

	var out []FieldError
	for _, ce := range cueerrors.Errors(e.RawError) {
		pos := ce.Position()
		file := ""
		if pos.IsValid() {
			file = filepath.Base(pos.Filename())
		}

		format, args := ce.Msg()
		var msg string
		if len(args) == 0 {
			msg = format
		} else {
			msg = fmt.Sprintf(format, args...)
		}

		// Skip disjunction summary lines — they add noise without actionable info.
		if strings.Contains(msg, "errors in empty disjunction") {
			continue
		}

		out = append(out, FieldError{
			File:    file,
			Line:    pos.Line(),
			Column:  pos.Column(),
			Path:    strings.Join(ce.Path(), "."),
			Message: msg,
		})
	}
	return out
}

// GroupedErrors walks the raw CUE error tree and returns errors grouped by
// message. Each GroupedError holds the message and all distinct source
// positions (primary + contributing) that report it, so conflicts between
// multiple files appear as a single entry with multiple locations.
//
// Returns nil if RawError is nil or produces no parseable errors.
func (e *ConfigError) GroupedErrors() []GroupedError {
	if e.RawError == nil {
		return nil
	}

	// groupOrder preserves insertion order of first-seen messages.
	type groupKey struct{ msg string }
	var groupOrder []groupKey
	groupMap := make(map[groupKey]*GroupedError)

	for _, ce := range cueerrors.Errors(e.RawError) {
		format, args := ce.Msg()
		var msg string
		if len(args) == 0 {
			msg = format
		} else {
			msg = fmt.Sprintf(format, args...)
		}

		// Skip disjunction summary lines — they add noise without actionable info.
		if strings.Contains(msg, "errors in empty disjunction") {
			continue
		}

		key := groupKey{msg: msg}
		ge, exists := groupMap[key]
		if !exists {
			ge = &GroupedError{Message: msg}
			groupMap[key] = ge
			groupOrder = append(groupOrder, key)
		}

		// Collect all positions: primary + contributing (e.g. both sides of a conflict).
		path := strings.Join(ce.Path(), ".")
		seen := make(map[string]bool, len(ge.Locations))
		for _, loc := range ge.Locations {
			seen[fmt.Sprintf("%s:%d:%d", loc.File, loc.Line, loc.Column)] = true
		}
		for _, pos := range cueerrors.Positions(ce) {
			if !pos.IsValid() {
				continue
			}
			file := filepath.Base(pos.Filename())
			locKey := fmt.Sprintf("%s:%d:%d", file, pos.Line(), pos.Column())
			if seen[locKey] {
				continue
			}
			seen[locKey] = true
			ge.Locations = append(ge.Locations, ErrorLocation{
				File:   file,
				Line:   pos.Line(),
				Column: pos.Column(),
				Path:   path,
			})
		}

		// If no valid position existed, record a position-less location so the
		// error message is still surfaced.
		if len(ge.Locations) == 0 {
			ge.Locations = append(ge.Locations, ErrorLocation{Path: path})
		}
	}

	out := make([]GroupedError, 0, len(groupOrder))
	for _, key := range groupOrder {
		out = append(out, *groupMap[key])
	}
	return out
}
