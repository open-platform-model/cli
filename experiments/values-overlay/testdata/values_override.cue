// Override user values — intentionally overlaps with values_partial.cue on
// the "image" key. When applied AFTER values_partial.cue, this file should
// win (last-file-wins semantics). env and debug are set here but not in
// values_partial.cue, so they add rather than conflict when combined.
package main

values: {
	image: "app:release"
	debug: true
	env:   "staging"
}
