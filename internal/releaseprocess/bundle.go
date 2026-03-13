package releaseprocess

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/provider"
	"github.com/opmodel/cli/pkg/render"
)

func ProcessBundleRelease(ctx context.Context, br *render.BundleRelease, values []cue.Value, p *provider.Provider) (*render.BundleResult, error) {
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
