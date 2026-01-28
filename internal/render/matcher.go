package render

import (
	"fmt"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
)

// computeMatches performs Phase 3: CUE-computed component matching.
// Reads the matchingPlan computed by CUE and creates jobs for parallel execution.
func computeMatches(mainCtx *cue.Context, meta *Metadata, verbose bool) ([]Job, []string, PhaseRecord, error) {
	phase3Start := time.Now()

	// The matching plan is already computed by CUE (via #MatchTransformers)
	matchedTransformersVal := meta.MatchingPlanVal
	if err := matchedTransformersVal.Err(); err != nil {
		return nil, nil, PhaseRecord{}, fmt.Errorf("failed to read matchingPlan: %w", err)
	}

	// Build the base TransformerContext with CLI-set fields only
	baseContext := TransformerContext{
		Name:      meta.ReleaseName,
		Namespace: meta.ReleaseNamespace,
	}

	// Iterate the CUE-computed matchedTransformers map
	var jobs []Job
	var unmatchedComponents []string

	matchIter, _ := matchedTransformersVal.Fields()
	matchCount := 0
	for matchIter.Next() {
		transformerID := matchIter.Selector().Unquoted()
		matchVal := matchIter.Value()

		transformerVal := matchVal.LookupPath(cue.ParsePath("transformer"))
		componentsVal := matchVal.LookupPath(cue.ParsePath("components"))

		// Get the #transform function
		transformFuncVal := transformerVal.LookupPath(cue.ParsePath("#transform"))

		// Iterate components matched to this transformer
		compIter, _ := componentsVal.List()
		for compIter.Next() {
			compVal := compIter.Value()
			compMetadataVal := compVal.LookupPath(cue.ParsePath("metadata"))
			compName, _ := compMetadataVal.LookupPath(cue.ParsePath("name")).String()

			matchCount++
			if verbose {
				fmt.Printf("  [MATCH %d] '%s' -> %s\n", matchCount, compName, transformerID)
			}

			// Build context: encode CLI-set fields, then fill hidden CUE definitions
			contextVal := mainCtx.Encode(baseContext).
				FillPath(cue.ParsePath("#moduleMetadata"), meta.ModuleMetadataVal).
				FillPath(cue.ParsePath("#componentMetadata"), compMetadataVal)

			// IMPORTANT: Unify transformer with inputs IN THE MAIN CONTEXT
			transformInput := mainCtx.CompileString("{}").
				FillPath(cue.ParsePath("#component"), compVal).
				FillPath(cue.ParsePath("context"), contextVal)

			unified := transformFuncVal.Unify(transformInput)
			if err := unified.Err(); err != nil {
				fmt.Printf("  [ERROR] Unification failed for '%s' with %s: %v\n", compName, transformerID, err)
				unmatchedComponents = append(unmatchedComponents, compName)
				continue
			}

			// Export the unified result as AST for thread-safe transport
			unifiedAST := unified.Syntax(cue.Final(), cue.Concrete(true)).(ast.Expr)

			jobs = append(jobs, Job{
				TransformerID: transformerID,
				ComponentName: compName,
				UnifiedAST:    unifiedAST,
			})
		}
	}

	record := PhaseRecord{
		Name:     "Component Matching",
		Duration: time.Since(phase3Start),
		Details:  fmt.Sprintf("%d jobs created", len(jobs)),
	}

	return jobs, unmatchedComponents, record, nil
}
