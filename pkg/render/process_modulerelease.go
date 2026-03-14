package render

import (
	"context"
	"fmt"

	"github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/provider"
)

// ProcessModuleRelease renders a prepared release with the given provider.
// The release must already be fully prepared via module.ParseModuleRelease.
func ProcessModuleRelease(ctx context.Context, rel *module.Release, p *provider.Provider) (*ModuleResult, error) {
	schemaComponents := rel.MatchComponents()
	if !schemaComponents.Exists() {
		return nil, fmt.Errorf("release %q: no components field in release spec", rel.Metadata.Name)
	}

	dataComponents, err := finalizeValue(p.Data.Context(), schemaComponents)
	if err != nil {
		return nil, fmt.Errorf("finalizing components: %w", err)
	}

	plan, err := Match(schemaComponents, p)
	if err != nil {
		return nil, err
	}

	renderer := NewModule(p)
	return renderer.Execute(ctx, rel, schemaComponents, dataComponents, plan)
}
