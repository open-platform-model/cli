package build

import "github.com/opmodel/cli/internal/build/transform"

// LoadedTransformer re-exports transform.LoadedTransformer for backward compatibility.
type LoadedTransformer = transform.LoadedTransformer

// LoadedProvider re-exports transform.LoadedProvider for backward compatibility.
type LoadedProvider = transform.LoadedProvider

// ProviderLoader re-exports transform.ProviderLoader for backward compatibility.
type ProviderLoader = transform.ProviderLoader

// Matcher re-exports transform.Matcher for backward compatibility.
type Matcher = transform.Matcher

// Executor re-exports transform.Executor for backward compatibility.
type Executor = transform.Executor

// TransformerContext re-exports transform.TransformerContext for backward compatibility.
type TransformerContext = transform.TransformerContext

// TransformError re-exports transform.TransformError for backward compatibility.
// Used by external packages (e.g., cmdutil) to detect transform failures.
type TransformError = transform.TransformError

// TransformerSummary re-exports transform.TransformerSummary for backward compatibility.
// Used by UnmatchedComponentError to describe available transformers.
type TransformerSummary = transform.TransformerSummary

// MatchResult re-exports transform.MatchResult for backward compatibility.
// Used internally by pipeline.go and collectWarnings.
type MatchResult = transform.MatchResult

// NewProviderLoader re-exports transform.NewProviderLoader for backward compatibility.
var NewProviderLoader = transform.NewProviderLoader

// NewMatcher re-exports transform.NewMatcher for backward compatibility.
var NewMatcher = transform.NewMatcher

// NewExecutor re-exports transform.NewExecutor for backward compatibility.
var NewExecutor = transform.NewExecutor

// BuildFQN re-exports transform.BuildFQN for backward compatibility.
var BuildFQN = transform.BuildFQN
