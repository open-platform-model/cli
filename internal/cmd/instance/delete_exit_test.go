package instance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"

	"github.com/open-platform-model/cli/internal/cmdutil"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	"github.com/open-platform-model/cli/internal/output"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
)

// A delete whose discovery request failed exits by the cause of that failure;
// every other per-resource failure keeps exit 1. The ModuleInstance is kept
// in each case, and an unserved kind gets the hint that names the way out.
func TestExecuteInstanceDelete_ExitCodeOfAnUnresolvedKind(t *testing.T) {
	widgetGVK := schema.GroupVersionKind{Group: "example.io", Version: "v1", Kind: "Widget"}
	entry := k8sinventory.Entry{Group: widgetGVK.Group, Version: "v1", Kind: "Widget", Namespace: "apps", Name: "main"}
	denied := apierrors.NewForbidden(schema.GroupResource{Group: widgetGVK.Group}, "", errors.New("no discovery"))
	discovery := func(cause error) error {
		return &kubernetes.DiscoveryError{GroupVersion: widgetGVK.GroupVersion(), Err: cause}
	}

	tests := map[string]struct {
		resolve  error
		wantCode int
		wantHint bool
	}{
		"discovery denied":      {resolve: discovery(denied), wantCode: opmexit.ExitPermissionDenied},
		"discovery unavailable": {resolve: discovery(apierrors.NewServiceUnavailable("down")), wantCode: opmexit.ExitConnectivityError},
		"discovery timeout":     {resolve: discovery(apierrors.NewServerTimeout(schema.GroupResource{}, "get", 1)), wantCode: opmexit.ExitConnectivityError},
		"discovery other":       {resolve: discovery(errors.New("broken pipe")), wantCode: opmexit.ExitGeneralError},
		"kind not served":       {resolve: &kubernetes.KindNotServedError{GVK: widgetGVK}, wantCode: opmexit.ExitGeneralError, wantHint: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			mi := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": inventory.APIVersionModuleInstance, "kind": inventory.KindModuleInstance,
				"metadata": map[string]any{"name": "demo", "namespace": "apps"},
			}}
			dyn := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
				map[schema.GroupVersionResource]string{inventory.ModuleInstanceGVR: "ModuleInstanceList"}, mi)
			client := &kubernetes.Client{
				Dynamic:   dyn,
				Resources: kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{widgetGVK: {Err: tt.resolve}}),
			}
			inv := &inventory.Record{Name: "demo", Namespace: "apps", Owner: inventory.OwnerCLI}
			inv.Inventory.Entries = []k8sinventory.Entry{entry}

			// Through the same discovery the command runs.
			live, missing, unreadable, err := inventory.DiscoverResourcesFromInventory(ctx, client, inv)
			require.NoError(t, err)
			require.Empty(t, live)
			require.Empty(t, missing, "the entry must not count as gone")

			var runErr error
			out := captureOutput(t, func() {
				runErr = executeInstanceDelete(ctx, client, &cmdutil.InstanceSelectorFlags{InstanceName: "demo"}, "apps", inv,
					live, unreadable, false, false, output.InstanceLogger("demo"))
			})

			var exitErr *opmexit.ExitError
			require.ErrorAs(t, runErr, &exitErr)
			assert.Equal(t, tt.wantCode, exitErr.Code)
			assert.NotContains(t, out, "Instance deleted")
			assert.Equal(t, tt.wantHint, strings.Contains(out, "Install the definition of that kind again"))
			_, miErr := dyn.Tracker().Get(inventory.ModuleInstanceGVR, "apps", "demo")
			assert.NoError(t, miErr, "the ModuleInstance is kept")
		})
	}
}
