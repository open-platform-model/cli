package releaseprocess

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/engine"
	"github.com/opmodel/cli/internal/runtime/bundlerelease"
	"github.com/opmodel/cli/pkg/provider"
)

func ProcessBundleRelease(ctx context.Context, br *bundlerelease.BundleRelease, values []cue.Value, p *provider.Provider) (*engine.BundleRenderResult, error) {
	_ = ctx
	_ = p
	_, _ = cuecontextMarker(br.RawCUE)
	merged, cfgErr := ValidateConfig(br.Config, values, "bundle", br.Metadata.Name)
	if cfgErr != nil {
		return nil, cfgErr
	}
	br.Values = merged
	return nil, fmt.Errorf("bundle release processing is not implemented yet")
}

func cuecontextMarker(v cue.Value) (*cue.Context, bool) {
	if !v.Exists() {
		return nil, false
	}
	return v.Context(), true
}
