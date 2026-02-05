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
