package render

import (
	"context"
	"fmt"

	"github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/provider"
)

// ProcessModuleInstance renders a prepared instance with the given provider.
// The instance must already be fully prepared via module.ParseModuleInstance.
// runtimeName identifies the runtime executing this render (e.g. "opm-cli");
// it is stamped onto every rendered resource as app.kubernetes.io/managed-by
// and MUST be non-empty.
//
// Was: ProcessModuleRelease (enhancement 0002 D8 hard-rename).
func ProcessModuleInstance(ctx context.Context, rel *module.Instance, p *provider.Provider, runtimeName string) (*ModuleResult, error) {
	if runtimeName == "" {
		return nil, fmt.Errorf("runtimeName must be non-empty")
	}
	schemaComponents := rel.MatchComponents()
	if !schemaComponents.Exists() {
		return nil, fmt.Errorf("instance %q: no components field in instance spec", rel.Metadata.Name)
	}

	dataComponents, err := finalizeValue(p.Data.Context(), schemaComponents)
	if err != nil {
		return nil, fmt.Errorf("finalizing components: %w", err)
	}

	plan, err := Match(schemaComponents, p)
	if err != nil {
		return nil, err
	}

	return NewModule(p, runtimeName).Execute(ctx, rel, schemaComponents, dataComponents, plan)
}
