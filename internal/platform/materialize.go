package platform

import (
	"context"
	"errors"
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

// WrapClusterMaterializeError adds the legacy-CR hint to a synthesis or
// materialize failure of a cluster-CR-sourced platform: a missing
// subscription version usually means a legacy filter-shaped CR stored before
// the scalar-version shape. Permanent behavior, not transitional — stored CRs
// keep their old shape in etcd until their next spec write. Any other error
// (and a nil error) passes through unchanged.
func WrapClusterMaterializeError(err error) error {
	if err != nil && errors.Is(err, synth.ErrSubscriptionMissingVersion) {
		return fmt.Errorf("%w — the cluster Platform may predate the scalar-version subscription shape (legacy filter-shaped CR); re-apply its spec with version set on every subscription", err)
	}
	return err
}
