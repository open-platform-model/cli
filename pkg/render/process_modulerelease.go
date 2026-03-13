package render

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/provider"
)

func ProcessModuleRelease(ctx context.Context, mr *module.Release, values []cue.Value, p *provider.Provider) (*ModuleResult, error) {
	merged, cfgErr := ValidateConfig(mr.Config, values, "module", mr.Metadata.Name)
	if cfgErr != nil {
		return nil, cfgErr
	}
	mr.Values = merged
	mr.RawCUE = mr.RawCUE.FillPath(cue.ParsePath("values"), merged)
	if err := mr.RawCUE.Err(); err != nil {
		return nil, fmt.Errorf("filling values into raw release: %w", err)
	}
	if err := mr.RawCUE.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("release %q: not fully concrete: %w", mr.Metadata.Name, err)
	}

	schemaComponents := mr.MatchComponents()
	if !schemaComponents.Exists() {
		return nil, fmt.Errorf("release %q: no components field in raw release", mr.Metadata.Name)
	}
	dataComponents, err := finalizeValue(p.Data.Context(), schemaComponents)
	if err != nil {
		return nil, fmt.Errorf("finalizing components: %w", err)
	}
	mr.DataComponents = dataComponents

	plan, err := Match(schemaComponents, p)
	if err != nil {
		return nil, err
	}

	renderer := NewModule(p)
	return renderer.Execute(ctx, mr, plan)
}
