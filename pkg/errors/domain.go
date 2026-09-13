package errors

// ValidationError indicates the instance failed validation.
type ValidationError struct {
	// Message describes what validation failed.
	Message string

	// Cause is the underlying error.
	Cause error

	// Details contains the formatted CUE error output.
	Details string
}

func (e *ValidationError) Error() string {
	if e.Cause != nil {
		return "instance validation failed: " + e.Message + ": " + e.Cause.Error()
	}
	return "instance validation failed: " + e.Message
}

func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// ErrorLocation is a source position paired with its CUE field path.
// Used in GroupedError to record every location where an error message appears.
type ErrorLocation struct {
	// File is the values file name (basename only).
	File string

	// Line is the 1-based line number. Zero means no position information.
	Line int

	// Column is the 1-based column number.
	Column int

	// Path is the dot-joined CUE field path at this position (e.g. "values.db.port").
	// May be empty when the error has no associated path.
	Path string
}

// GroupedError collects all source locations where the same error message
// appears. CUE can report the same logical error at multiple positions (e.g.
// both sides of a value conflict), so grouping by message collapses duplicates
// and makes conflicts immediately readable without a separate section.
type GroupedError struct {
	// Message is the human-readable error description shared by all locations.
	Message string

	// Locations holds one entry per distinct source position reporting this error.
	Locations []ErrorLocation
}
