package platform

import (
	"context"
	"fmt"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/materialize"
)

// Materialize synthesizes and materializes the resolved platform input on
// the invocation's kernel — the same SynthesizePlatform → Materialize chain
// the operator's PlatformReconciler runs. Materialize performs registry I/O;
// callers on offline paths decide when to invoke it.
func Materialize(ctx context.Context, k *kernel.Kernel, in synth.PlatformInput) (*materialize.MaterializedPlatform, error) {
	p, err := k.SynthesizePlatform(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("synthesizing platform %q: %w", in.Name, err)
	}
	mp, err := k.Materialize(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("materializing platform %q: %w", in.Name, err)
	}
	return mp, nil
}
