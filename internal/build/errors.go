package build

import (
	"fmt"
	"os"
	"strings"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
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

// UnhandledTraitError indicates a trait was not handled by any transformer.
type UnhandledTraitError struct {
	// ComponentName is the component with the unhandled trait.
	ComponentName string

	// TraitFQN is the fully qualified trait name.
	TraitFQN string

	// Strict indicates if this was treated as an error (strict mode)
	// or warning (normal mode).
	Strict bool
}

func (e *UnhandledTraitError) Error() string {
	return fmt.Sprintf("component %q: unhandled trait %q", e.ComponentName, e.TraitFQN)
}

func (e *UnhandledTraitError) Component() string {
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

// formatCUEDetails formats a CUE error into a multi-line string matching
// the output style of `cue vet`. Each error includes its CUE path, message,
// and source positions (file:line:col). Errors are deduplicated and sorted.
//
// Example output:
//
//	values.media.test: conflicting values "test" and {mountPath:string,...}:
//	    ./values.cue:12:5
//	    ./module.cue:15:10
func formatCUEDetails(err error) string {
	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		cwd = ""
	}
	return strings.TrimSpace(cueerrors.Details(err, &cueerrors.Config{
		Cwd: cwd,
	}))
}

// collectAllCUEErrors runs Validate() on the full value tree.
// Returns a combined error containing all discovered errors, or nil if clean.
func collectAllCUEErrors(v cue.Value) error {
	return v.Validate()
}

// validateValuesAgainstConfig validates each top-level field of the values
// struct against the #config definition independently.
//
// This works around a CUE engine limitation where closedness errors
// ("field not allowed") are suppressed when other error types (e.g., type
// mismatches) exist in the same struct. CUE's evaluator skips close-checking
// on structs that already have a child error (typocheck.go early return).
//
// By validating each field in its own isolated unification, both type errors
// and closedness errors are detected regardless of each other.
//
// Returns a combined error with all validation errors, or nil if clean.
func validateValuesAgainstConfig(cueCtx *cue.Context, configDef, valuesVal cue.Value) error {
	var combined cueerrors.Error

	// Only iterate regular fields (not optional, hidden, or definitions).
	// These are the concrete values provided by the user.
	iter, err := valuesVal.Fields()
	if err != nil {
		// If we can't iterate fields, the value itself may be an error.
		return err
	}

	for iter.Next() {
		sel := iter.Selector()
		fieldVal := iter.Value()

		// Build a single-field struct and unify with #config independently.
		// This isolates each field's validation so errors don't suppress each other.
		// Use cue.Path with the selector directly to avoid string-parsing issues.
		fieldPath := cue.MakePath(sel)
		singleField := cueCtx.CompileString("{}")
		singleField = singleField.FillPath(fieldPath, fieldVal)
		result := configDef.Unify(singleField)

		if err := result.Validate(); err != nil {
			combined = appendCUEError(combined, err)
		}
	}

	if combined == nil {
		return nil
	}
	return combined
}

// appendCUEError appends a Go error to a CUE error list, handling type
// promotion from plain errors to cueerrors.Error.
func appendCUEError(list cueerrors.Error, err error) cueerrors.Error {
	var cueErr cueerrors.Error
	if ok := cueerrors.As(err, &cueErr); ok {
		return cueerrors.Append(list, cueErr)
	}
	return cueerrors.Append(list, cueerrors.Promote(err, ""))
}
