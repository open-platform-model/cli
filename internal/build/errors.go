package build

import "fmt"

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
