// Package render implements the hybrid Go+CUE render pipeline.
package render

import (
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
)

// Options configures the render pipeline.
type Options struct {
	Dir         string   // Module directory
	ValuesFiles []string // Values files to unify
	Verbose     bool     // Enable verbose output
}

// Job carries data to a worker thread via AST transport (thread-safe).
type Job struct {
	TransformerID string   // Transformer identifier (e.g., "deployment")
	ComponentName string   // Component name
	UnifiedAST    ast.Expr // Unified transformer + component + context as AST
}

// Result contains the output from a worker execution.
type Result struct {
	TransformerID string         // Source transformer
	ComponentName string         // Source component
	Output        map[string]any // Decoded K8s resource
	Error         error          // Transformation error (if any)
	Duration      time.Duration  // Worker execution time
}

// TransformerContext carries CLI-set fields for injection into CUE.
// The full context is built in CUE with hidden fields (#moduleMetadata, #componentMetadata).
type TransformerContext struct {
	Name      string `json:"name"`      // Release name
	Namespace string `json:"namespace"` // Target namespace
}

// Metadata holds extracted module/release information.
type Metadata struct {
	MatchingPlanVal   cue.Value // CUE-computed matching plan
	ModuleMetadataVal cue.Value // Module metadata for context injection
	ReleaseName       string    // Release name
	ReleaseNamespace  string    // Release namespace
	ModuleVersion     string    // Module version
}

// PhaseStep represents a timed sub-step within a phase.
type PhaseStep struct {
	Name     string
	Duration time.Duration
}

// PhaseRecord captures timing for an entire pipeline phase.
type PhaseRecord struct {
	Name     string
	Duration time.Duration
	Steps    []PhaseStep
	Details  string // Human-readable summary (e.g., "8 jobs from 7 components")
}

// Manifest represents a rendered Kubernetes resource with metadata.
type Manifest struct {
	Object        map[string]any // The K8s resource
	TransformerID string         // Source transformer
	ComponentName string         // Source component
}

// RenderResult contains the output from the render pipeline.
type RenderResult struct {
	Manifests []Manifest // Rendered K8s manifests

	// Metadata for labeling/tracking
	ModuleName       string
	ModuleVersion    string
	ReleaseNamespace string

	// Timing/debugging info
	Phases []PhaseRecord

	// Aggregated errors (fail-on-end)
	Errors []error
}
