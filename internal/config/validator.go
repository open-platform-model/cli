package config

import (
	"embed"
	"fmt"
	"regexp"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

//go:embed schema.cue
var schemaFS embed.FS

// namespaceRegex validates Kubernetes namespace names per RFC 1123.
var namespaceRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

// Error implements the error interface.
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}

	var sb strings.Builder
	sb.WriteString("config validation failed:\n")
	for _, err := range e {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", err.Field, err.Message))
	}
	return sb.String()
}

// Validator validates configuration against the embedded CUE schema.
type Validator struct {
	ctx    *cue.Context
	schema cue.Value
}

// NewValidator creates a new configuration validator.
func NewValidator() (*Validator, error) {
	ctx := cuecontext.New()

	// Read the embedded schema
	schemaData, err := schemaFS.ReadFile("schema.cue")
	if err != nil {
		return nil, fmt.Errorf("reading embedded schema: %w", err)
	}

	// Compile the schema
	schema := ctx.CompileBytes(schemaData)
	if schema.Err() != nil {
		return nil, fmt.Errorf("compiling schema: %w", schema.Err())
	}

	return &Validator{
		ctx:    ctx,
		schema: schema,
	}, nil
}

// Validate validates the given configuration.
func (v *Validator) Validate(cfg *Config) error {
	var errs ValidationErrors

	// Validate namespace format
	if cfg.Namespace != "" {
		if !namespaceRegex.MatchString(cfg.Namespace) {
			errs = append(errs, ValidationError{
				Field:   "namespace",
				Message: "must be a valid Kubernetes namespace name (lowercase alphanumeric with hyphens)",
			})
		}
		if len(cfg.Namespace) > 63 {
			errs = append(errs, ValidationError{
				Field:   "namespace",
				Message: "must be at most 63 characters",
			})
		}
	}

	// Validate paths are not empty strings (if set, they should be valid paths)
	if cfg.Kubeconfig != "" && strings.TrimSpace(cfg.Kubeconfig) == "" {
		errs = append(errs, ValidationError{
			Field:   "kubeconfig",
			Message: "must not be empty or whitespace only",
		})
	}

	if cfg.CacheDir != "" && strings.TrimSpace(cfg.CacheDir) == "" {
		errs = append(errs, ValidationError{
			Field:   "cacheDir",
			Message: "must not be empty or whitespace only",
		})
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// ValidateFile validates a configuration file at the given path.
func (v *Validator) ValidateFile(path string) error {
	loader := NewLoader()
	cfg, err := loader.Load(path)
	if err != nil {
		return fmt.Errorf("loading config file: %w", err)
	}

	return v.Validate(cfg)
}

// ValidateNamespace checks if a namespace name is valid.
func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return nil
	}

	if !namespaceRegex.MatchString(namespace) {
		return &ValidationError{
			Field:   "namespace",
			Message: "must be a valid Kubernetes namespace name (lowercase alphanumeric with hyphens)",
		}
	}

	if len(namespace) > 63 {
		return &ValidationError{
			Field:   "namespace",
			Message: "must be at most 63 characters",
		}
	}

	return nil
}
