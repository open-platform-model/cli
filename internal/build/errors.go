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
