package config

import (
	cueerrors "cuelang.org/go/cue/errors"
)

// cueErrorUnder reports whether the chain of err holds a CUE error whose
// path starts with the selectors of prefix. A hint is picked from where an
// evaluation failed, never from the words of its message. An error with no
// CUE error in its chain, and a nil error, are under nothing.
func cueErrorUnder(err error, prefix ...string) bool {
	if err == nil || len(prefix) == 0 {
		return false
	}
	for _, e := range cueerrors.Errors(err) {
		if hasPrefix(e.Path(), prefix) {
			return true
		}
	}
	return false
}

// hasPrefix reports whether path starts with prefix.
func hasPrefix(path, prefix []string) bool {
	if len(path) < len(prefix) {
		return false
	}
	for i, sel := range prefix {
		if path[i] != sel {
			return false
		}
	}
	return true
}
