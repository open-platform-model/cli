package apply

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	"github.com/open-platform-model/cli/internal/output"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
)

// A prune that a failed discovery request stopped fails the command with the
// code of that failure; a prune with a failed delete keeps exit 1.
func TestPruneStale_ExitCode(t *testing.T) {
	cmGVK := schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}
	stale := []k8sinventory.Entry{{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "stale"}}
	denied := apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "stale", errors.New("denied"))
	discovery := func(cause error) error {
		return &kubernetes.DiscoveryError{GroupVersion: cmGVK.GroupVersion(), Err: cause}
	}

	tests := map[string]struct {
		resolve   error
		deleteErr error
		want      int
	}{
		"discovery denied":      {resolve: discovery(denied), want: opmexit.ExitPermissionDenied},
		"discovery unavailable": {resolve: discovery(apierrors.NewServiceUnavailable("down")), want: opmexit.ExitConnectivityError},
		"discovery other":       {resolve: discovery(errors.New("broken pipe")), want: opmexit.ExitGeneralError},
		"kind not served":       {resolve: &kubernetes.KindNotServedError{GVK: cmGVK}, want: opmexit.ExitGeneralError},
		"delete denied":         {deleteErr: denied, want: opmexit.ExitGeneralError},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// The stale object is the instance's own, so the verdict allows the delete.
			dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), renderedConfigMap(stale[0].Name))
			if tt.deleteErr != nil {
				dyn.PrependReactor("delete", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, tt.deleteErr
				})
			}
			resources := kubetest.Resources()
			if tt.resolve != nil {
				resources.Set(cmGVK, kubetest.Outcome{Err: tt.resolve})
			}
			client := &kubernetes.Client{Dynamic: dyn, Resources: resources}

			notPruned, code, err := pruneStale(context.Background(), client, stale, "", output.InstanceLogger("demo"))

			require.NoError(t, err)
			assert.Equal(t, stale, notPruned, "the entry stays in the record")
			assert.Equal(t, tt.want, code)
		})
	}
}
