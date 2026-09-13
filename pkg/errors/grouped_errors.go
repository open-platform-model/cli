package errors

import (
	"fmt"
	"path/filepath"
	"strings"

	cueerrors "cuelang.org/go/cue/errors"
)

// GroupedErrorsFromError extracts the CUE errors carried by any error
// (including wrapped ones such as fmt.Errorf("...: %w", cueErr)) and groups
// them by message. Each GroupedError holds the message and all distinct
// source positions (primary + contributing) that report it, so conflicts
// between multiple files appear as a single entry with multiple locations.
// The kernel's values validation returns exactly such a tree.
//
// Returns nil if no CUE error information can be extracted.
func GroupedErrorsFromError(err error) []GroupedError {
	if err == nil {
		return nil
	}
	return groupCUEErrors(err)
}

// groupCUEErrors walks the CUE error tree obtained from err and groups
// errors by message, collecting all source positions (primary +
// contributing via InputPositions) per group.
func groupCUEErrors(err error) []GroupedError {
	cueErrs := cueerrors.Errors(err)
	if len(cueErrs) == 0 {
		return nil
	}

	// groupOrder preserves insertion order of first-seen message/path pairs.
	type groupKey struct {
		msg  string
		path string
	}
	var groupOrder []groupKey
	groupMap := make(map[groupKey]*GroupedError)

	for _, ce := range cueErrs {
		path := normalizeCUEPath(ce.Path())

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

		key := groupKey{msg: msg, path: path}
		ge, exists := groupMap[key]
		if !exists {
			ge = &GroupedError{Message: msg}
			groupMap[key] = ge
			groupOrder = append(groupOrder, key)
		}

		// Collect all positions: primary + contributing (e.g. both sides of a
		// conflict). cueerrors.Positions returns Position() + InputPositions()
		// deduped and sorted.
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

func normalizeCUEPath(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	path := strings.Join(parts, ".")
	path = strings.TrimPrefix(path, "#module.#config.")
	path = strings.TrimPrefix(path, "#module.#config")
	path = strings.TrimPrefix(path, "#config.")
	path = strings.TrimPrefix(path, "#config")
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return "values"
	}
	if strings.HasPrefix(path, "values.") || path == "values" {
		return path
	}
	return "values." + path
}
