package render

import (
	"fmt"
	"path/filepath"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

// loadModule performs Phase 1: CUE module loading and validation.
func loadModule(dir string) (cue.Value, *cue.Context, PhaseRecord, error) {
	phase1Start := time.Now()
	var phase1Steps []PhaseStep

	mainCtx := cuecontext.New()
	expDir, err := filepath.Abs(dir)
	if err != nil {
		return cue.Value{}, nil, PhaseRecord{}, fmt.Errorf("failed to resolve directory: %w", err)
	}

	cfg := &load.Config{
		ModuleRoot: expDir,
		Dir:        expDir,
	}

	loadStart := time.Now()
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) != 1 {
		return cue.Value{}, nil, PhaseRecord{}, fmt.Errorf("expected 1 instance, got %d", len(instances))
	}
	if instances[0].Err != nil {
		return cue.Value{}, nil, PhaseRecord{}, fmt.Errorf("failed to load CUE instance: %w", instances[0].Err)
	}
	phase1Steps = append(phase1Steps, PhaseStep{Name: "load.Instances", Duration: time.Since(loadStart)})

	buildStart := time.Now()
	rootVal := mainCtx.BuildInstance(instances[0])
	// NOTE: We intentionally do not check rootVal.Err() here.
	// The CUE instance contains abstract definitions that are not concrete at the top level.
	phase1Steps = append(phase1Steps, PhaseStep{Name: "BuildInstance", Duration: time.Since(buildStart)})

	record := PhaseRecord{
		Name:     "Module Loading",
		Duration: time.Since(phase1Start),
		Steps:    phase1Steps,
		Details:  "CUE module loaded",
	}

	return rootVal, mainCtx, record, nil
}

// extractMetadata performs Phase 2: Provider loading and metadata extraction.
func extractMetadata(rootVal cue.Value) (*Metadata, PhaseRecord, error) {
	phase2Start := time.Now()
	var phase2Steps []PhaseStep

	// Load the matching plan (computed by CUE using #MatchTransformers)
	lookupStart := time.Now()
	matchingPlanVal := rootVal.LookupPath(cue.ParsePath("matchingPlan"))
	if err := matchingPlanVal.Err(); err != nil {
		return nil, PhaseRecord{}, fmt.Errorf("failed to load matchingPlan: %w", err)
	}
	phase2Steps = append(phase2Steps, PhaseStep{Name: "LookupPath(matchingPlan)", Duration: time.Since(lookupStart)})

	// Extract module release metadata for TransformerContext
	metadataStart := time.Now()

	// Look for ModuleRelease - try common naming patterns
	var moduleReleaseVal cue.Value
	releasePatterns := []string{
		"moduleRelease",
		"release",
		"#moduleRelease",
	}

	for _, pattern := range releasePatterns {
		testVal := rootVal.LookupPath(cue.ParsePath(pattern))
		if testVal.Exists() && testVal.Err() == nil {
			moduleReleaseVal = testVal
			break
		}
	}

	if !moduleReleaseVal.Exists() {
		return nil, PhaseRecord{}, fmt.Errorf("no module release found (tried: %v)", releasePatterns)
	}

	releaseMetadataVal := moduleReleaseVal.LookupPath(cue.ParsePath("metadata"))
	if err := releaseMetadataVal.Err(); err != nil {
		return nil, PhaseRecord{}, fmt.Errorf("failed to load release metadata: %w", err)
	}

	releaseName, _ := releaseMetadataVal.LookupPath(cue.ParsePath("name")).String()
	releaseNamespace, _ := releaseMetadataVal.LookupPath(cue.ParsePath("namespace")).String()

	// Get the module metadata for transformer context injection
	moduleVal := moduleReleaseVal.LookupPath(cue.ParsePath("#module"))
	moduleMetadataVal := moduleVal.LookupPath(cue.ParsePath("metadata"))
	if err := moduleMetadataVal.Err(); err != nil {
		return nil, PhaseRecord{}, fmt.Errorf("failed to load module metadata: %w", err)
	}

	moduleVersion, _ := moduleMetadataVal.LookupPath(cue.ParsePath("version")).String()

	phase2Steps = append(phase2Steps, PhaseStep{Name: "Extract metadata", Duration: time.Since(metadataStart)})

	record := PhaseRecord{
		Name:     "Provider Loading",
		Duration: time.Since(phase2Start),
		Steps:    phase2Steps,
		Details:  fmt.Sprintf("Release: %s", releaseName),
	}

	meta := &Metadata{
		MatchingPlanVal:   matchingPlanVal,
		ModuleMetadataVal: moduleMetadataVal,
		ReleaseName:       releaseName,
		ReleaseNamespace:  releaseNamespace,
		ModuleVersion:     moduleVersion,
	}

	return meta, record, nil
}
