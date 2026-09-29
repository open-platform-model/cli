package render

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	liberrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/cmdutil/cmdutiltest"
)

// Instance acquisition decides by package kind: a module package handed to
// AcquireInstanceFromDir fails the shape gate with ErrWrongKind, which is
// what the instance commands key their module refusal on.
func TestAcquireInstanceFromDir_ModulePackageIsWrongKind(t *testing.T) {
	k := kernel.New(kernel.WithRegistry("127.0.0.1:1+insecure"))
	_, err := k.AcquireInstanceFromDir(context.Background(), cmdutiltest.WriteMinimalModule(t))
	require.ErrorIs(t, err, liberrors.ErrWrongKind)
}
