package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"
	"github.com/charmbracelet/lipgloss"

	"github.com/opmodel/cli/internal/output"
)

// RenderError is a base interface for render errors.
// All render-specific errors implement this interface.
type RenderError interface {
	error

	// Component returns the component name where the error occurred.
	Component() string
}

// UnmatchedComponentError indicates no transformer matched a component.
type UnmatchedComponentError struct {
	// ComponentName is the name of the unmatched component.
	ComponentName string

	// Available lists transformers and their requirements.
	// Helps users understand what's needed to match.
	Available []TransformerSummary
}

func (e *UnmatchedComponentError) Error() string {
	return fmt.Sprintf("component %q: no matching transformer", e.ComponentName)
}

func (e *UnmatchedComponentError) Component() string {
	return e.ComponentName
}

// TransformError indicates transformer execution failed.
type TransformError struct {
	// ComponentName is the component being transformed.
	ComponentName string

	// TransformerFQN is the transformer that failed.
	TransformerFQN string

	// Cause is the underlying error.
	Cause error
}

func (e *TransformError) Error() string {
	return fmt.Sprintf("component %q, transformer %q: %v",
		e.ComponentName, e.TransformerFQN, e.Cause)
}

func (e *TransformError) Component() string {
	return e.ComponentName
}

func (e *TransformError) Unwrap() error {
	return e.Cause
}

// TransformerSummary provides guidance on transformer requirements.
// Used in error messages to help users understand matching.
type TransformerSummary struct {
	// FQN is the fully qualified transformer name.
	FQN string

	// RequiredLabels that components must have.
	RequiredLabels map[string]string

	// RequiredResources (FQNs) that components must have.
	RequiredResources []string

	// RequiredTraits (FQNs) that components must have.
	RequiredTraits []string
}

// NamespaceRequiredError indicates namespace was not provided and module has no default.
type NamespaceRequiredError struct {
	// ModuleName is the module being loaded.
	ModuleName string
}

func (e *NamespaceRequiredError) Error() string {
	return fmt.Sprintf("namespace required for module %q: set --namespace flag or define metadata.defaultNamespace in module", e.ModuleName)
}

// Component returns empty string as this is not a component error.
func (e *NamespaceRequiredError) Component() string {
	return ""
}

// ModuleValidationError indicates the module failed CUE validation.
// This typically happens when required fields are missing or constraints are violated.
type ModuleValidationError struct {
	// Message describes what validation failed.
	Message string

	// ComponentName is the component with the error (if applicable).
	ComponentName string

	// FieldPath is the path to the field with the error (if known).
	FieldPath string

	// Cause is the underlying CUE error.
	Cause error
}

func (e *ModuleValidationError) Error() string {
	if e.ComponentName != "" {
		return fmt.Sprintf("module validation failed for component %q: %s", e.ComponentName, e.Message)
	}
	return fmt.Sprintf("module validation failed: %s", e.Message)
}

func (e *ModuleValidationError) Component() string {
	return e.ComponentName
}

func (e *ModuleValidationError) Unwrap() error {
	return e.Cause
}

// ReleaseValidationError indicates the release failed validation.
// This typically happens when values are incomplete or non-concrete.
type ReleaseValidationError struct {
	// Message describes what validation failed.
	Message string

	// Cause is the underlying error.
	Cause error

	// Details contains the formatted CUE error output with all individual
	// errors, their CUE paths, and source positions. Formatted using the
	// same style as `cue vet` (via cuelang.org/go/cue/errors.Details).
	// Empty when the error is not a CUE validation error.
	Details string
}

func (e *ReleaseValidationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("release validation failed: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("release validation failed: %s", e.Message)
}

func (e *ReleaseValidationError) Unwrap() error {
	return e.Cause
}

// Lipgloss styles for CUE error output.
var (
	// errStylePath styles CUE paths (e.g. "values.media.test") — cyan.
	errStylePath = lipgloss.NewStyle().Foreground(output.ColorCyan)

	// errStyleDim styles structural chrome (arrows, file paths) — faint.
	errStyleDim = lipgloss.NewStyle().Faint(true)

	// errStylePosition styles line:col numbers — yellow.
	errStylePosition = lipgloss.NewStyle().Foreground(output.ColorYellow)
)

// formatCUEDetails formats a CUE error into a multi-line, lipgloss-colorized
// string. Each error includes its CUE path (cyan), message (default),
// and source positions with a dim arrow prefix, dim file path, and yellow
// line:col numbers.
//
// Example output (conceptual, without ANSI codes):
//
//	values.media.test: conflicting values "test" and {mountPath:string,...}:
//	    → ./values.cue:12:5
//	    → ./module.cue:15:10
func formatCUEDetails(err error) string {
	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		cwd = ""
	}

	errs := cueerrors.Errors(err)
	if len(errs) == 0 {
		// Not a CUE error — fall back to plain text.
		return err.Error()
	}

	// Deduplicate and sort: replicate CUE's sanitize logic.
	errs = deduplicateCUEErrors(errs)

	var b strings.Builder
	for i, e := range errs {
		if i > 0 {
			b.WriteByte('\n')
		}

		// Path prefix (cyan).
		if path := strings.Join(e.Path(), "."); path != "" {
			b.WriteString(errStylePath.Render(path))
			b.WriteString(": ")
		}

		// Message text (default terminal color).
		b.WriteString(cueErrorMessage(e))

		// Positions — each on its own line with arrow prefix.
		positions := cueerrors.Positions(e)
		if len(positions) > 0 {
			b.WriteString(":")
		}
		for _, p := range positions {
			b.WriteByte('\n')
			pos := p.Position()
			filePath := cueRelPath(pos.Filename, cwd)

			// "    → ./file.cue:12:5"
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

// cueErrorMessage walks the wrapped error chain of a CUE error and
// concatenates messages with ": " separators, replicating the logic of
// CUE's internal writeErr without the path prefix.
func cueErrorMessage(e cueerrors.Error) string {
	var parts []string
	var current error = e

	for current != nil {
		cueErr, ok := current.(cueerrors.Error)
		if !ok {
			// Non-CUE error at the end of the chain.
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

// cueRelPath converts an absolute file path to a relative one based on cwd.
// If cwd is empty or Rel fails, returns the original path.
// Adds "./" prefix when the result doesn't start with "." (matching CUE convention).
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

// deduplicateCUEErrors sorts and deduplicates CUE errors.
// Two errors are considered duplicates if they have the same position and path.
// This replicates CUE's unexported sanitize logic.
func deduplicateCUEErrors(errs []cueerrors.Error) []cueerrors.Error {
	if len(errs) <= 1 {
		return errs
	}

	// Sort by position, then path, then message.
	result := make([]cueerrors.Error, len(errs))
	copy(result, errs)

	// Sort.
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && compareCUEErrors(result[j-1], result[j]) > 0; j-- {
			result[j-1], result[j] = result[j], result[j-1]
		}
	}

	// Deduplicate: keep first of each group with same position+path.
	deduped := result[:1]
	for _, e := range result[1:] {
		prev := deduped[len(deduped)-1]
		if !approximateEqualCUE(prev, e) {
			deduped = append(deduped, e)
		}
	}

	return deduped
}

// compareCUEErrors compares two CUE errors for sorting.
func compareCUEErrors(a, b cueerrors.Error) int {
	aPos := a.Position()
	bPos := b.Position()

	// Invalid positions sort first.
	if !aPos.IsValid() && bPos.IsValid() {
		return -1
	}
	if aPos.IsValid() && !bPos.IsValid() {
		return 1
	}
	if aPos.IsValid() && bPos.IsValid() {
		if c := token.Pos.Compare(aPos, bPos); c != 0 {
			return c
		}
	}

	// Then by path.
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

	// Then by error message.
	if a.Error() < b.Error() {
		return -1
	}
	if a.Error() > b.Error() {
		return 1
	}
	return 0
}

// approximateEqualCUE checks if two CUE errors are duplicates.
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

// collectAllCUEErrors runs Validate() on the full value tree.
// Returns a combined error containing all discovered errors, or nil if clean.
func collectAllCUEErrors(v cue.Value) error {
	return v.Validate()
}

// pathRewrittenError wraps a cueerrors.Error and overrides Path() to return
// a custom path. All other methods delegate to the inner error.
//
// This is used to rewrite CUE error paths from #config-rooted (e.g.,
// #config.media.test) to values-rooted (e.g., values.media.test) so that
// error messages reflect the user's input, not CUE internals.
type pathRewrittenError struct {
	inner   cueerrors.Error
	newPath []string
}

func (e *pathRewrittenError) Error() string {
	return e.inner.Error()
}

func (e *pathRewrittenError) Position() token.Pos {
	return e.inner.Position()
}

func (e *pathRewrittenError) InputPositions() []token.Pos {
	return e.inner.InputPositions()
}

func (e *pathRewrittenError) Path() []string {
	return e.newPath
}

func (e *pathRewrittenError) Msg() (format string, args []interface{}) {
	return e.inner.Msg()
}

// rewriteErrorPath wraps a CUE error with a new path that prepends basePath
// to the error's existing relative path segments.
func rewriteErrorPath(e cueerrors.Error, basePath []string) cueerrors.Error {
	errPath := e.Path()
	newPath := make([]string, 0, len(basePath)+len(errPath))
	newPath = append(newPath, basePath...)
	newPath = append(newPath, errPath...)

	return &pathRewrittenError{
		inner:   e,
		newPath: newPath,
	}
}

// findSourcePosition extracts a source file position from a CUE value.
//
// For single-source values, v.Pos() returns the position directly.
// For unified values (multiple sources), Pos() returns the last conjunct's
// position which may not be useful. In that case, Expr() decomposes the value
// into its constituent conjuncts, each retaining its original source position.
//
// Returns token.NoPos if no valid position can be found.
func findSourcePosition(v cue.Value) token.Pos {
	// Try direct position first (works for single-source values).
	if pos := v.Pos(); pos.IsValid() {
		return pos
	}

	// Decompose unified values via Expr() to find a valid position.
	op, parts := v.Expr()
	if op == cue.AndOp {
		for _, part := range parts {
			if pos := part.Pos(); pos.IsValid() {
				return pos
			}
		}
	}

	return token.NoPos
}

// validateValuesAgainstConfig validates user-provided values against the #config
// definition by recursively walking every field and checking each against the
// corresponding schema node.
//
// This replaces the previous per-field isolation approach. Instead of relying on
// CUE's internal closedness checker (which produces sparse position info and has
// a suppression bug), we:
//
//   - Check closedness manually via Value.Allows() — giving us full control over
//     the error message, path, and source position.
//   - Validate type constraints by unifying each allowed field with its resolved
//     schema counterpart — CUE's type errors naturally include positions from both sides.
//   - Handle pattern constraints ([Name=string]: {...}) via Str(key).Optional()
//     resolution for recursive descent into pattern-matched fields.
//   - Build error paths rooted at "values" (the user's perspective), not "#config".
//
// Returns a combined error with all validation errors, or nil if clean.
func validateValuesAgainstConfig(configDef, valuesVal cue.Value) error {
	combined := validateFieldsRecursive(configDef, valuesVal, []string{"values"}, nil)
	if combined == nil {
		return nil
	}
	return combined
}

// validateFieldsRecursive walks every field in data recursively and validates
// each against the corresponding schema node.
//
// At each level:
//  1. Check if the field is allowed by the schema via Value.Allows().
//     If not → emit "field not allowed" with source position, don't recurse.
//  2. Resolve the schema field: try literal lookup first, then pattern constraint
//     resolution via Str(key).Optional().
//  3. Unify the data field with the resolved schema field and check for errors.
//     If errors → emit with rewritten path, don't recurse.
//  4. If the field is a struct and passes validation → recurse into children.
func validateFieldsRecursive(schema, data cue.Value, path []string, errs cueerrors.Error) cueerrors.Error {
	iter, err := data.Fields()
	if err != nil {
		return errs
	}

	for iter.Next() {
		sel := iter.Selector()
		fieldVal := iter.Value()
		fieldName := sel.Unquoted()

		// Build the full path for this field using Selector.String() which
		// handles quoting for non-identifier names (e.g., "test-field").
		fieldPath := make([]string, len(path), len(path)+1)
		copy(fieldPath, path)
		fieldPath = append(fieldPath, sel.String())

		// Phase 1: Closedness check — is this field allowed by the schema?
		if !schema.Allows(cue.Str(fieldName)) {
			pos := findSourcePosition(fieldVal)
			fieldNotAllowed := cueerrors.Newf(pos, "field not allowed")
			errs = cueerrors.Append(errs, &pathRewrittenError{
				inner:   fieldNotAllowed,
				newPath: fieldPath,
			})
			continue // Don't recurse into disallowed fields.
		}

		// Phase 2: Resolve the schema field for this data field.
		// Try literal field first, then pattern constraint resolution.
		schemaField := schema.LookupPath(cue.MakePath(sel))
		if !schemaField.Exists() {
			// Try pattern constraint resolution (e.g., [Name=string]: {...}).
			schemaField = schema.LookupPath(cue.MakePath(cue.Str(fieldName).Optional()))
		}
		if !schemaField.Exists() {
			// Schema allows it (Allows() returned true) but no constraint
			// to validate against — skip validation for this field.
			continue
		}

		// Phase 3: For struct-kind values, recurse into children rather than
		// unifying at this level. This prevents nested errors from being reported
		// at the parent level and ensures each error is attributed to its exact
		// field with the correct path.
		if fieldVal.IncompleteKind() == cue.StructKind {
			errs = validateFieldsRecursive(schemaField, fieldVal, fieldPath, errs)
			continue
		}

		// Phase 4: Type validation for leaf values — unify data with schema.
		unified := schemaField.Unify(fieldVal)
		if fieldErr := unified.Validate(); fieldErr != nil {
			// Replace each CUE error's path with our values-rooted path.
			// CUE errors from unification carry #config-rooted paths which
			// we discard entirely — fieldPath already captures the correct
			// location from the user's perspective.
			for _, e := range cueerrors.Errors(fieldErr) {
				errs = cueerrors.Append(errs, &pathRewrittenError{
					inner:   e,
					newPath: fieldPath,
				})
			}
		}
	}
	return errs
}
