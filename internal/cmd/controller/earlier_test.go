package controllercmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	ctrllib "github.com/open-platform-model/cli/internal/controller"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
)

const earlierNoticeStart = "the cluster still runs the controller of the earlier install"

func earlierCluster(withDeployment bool, extra ...*unstructured.Unstructured) *kubernetes.Client {
	var objs []runtime.Object
	if withDeployment {
		objs = append(objs, &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]any{"name": "opm-operator-controller-manager", "namespace": "opm-operator-system"},
		}})
	}
	for _, o := range extra {
		objs = append(objs, o)
	}
	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}: "CustomResourceDefinitionList",
		{Group: "opmodel.dev", Version: "v1alpha1", Resource: "moduleinstances"}:              "ModuleInstanceList",
	}
	return &kubernetes.Client{
		Resources: kubetest.Resources(),
		Dynamic:   fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds, objs...),
		Clientset: k8sfake.NewClientset(),
	}
}

// runCRDsOnly runs the --crds-only install command against client.
func runCRDsOnly(t *testing.T, client *kubernetes.Client) error {
	t.Helper()
	prev := newClusterClient
	newClusterClient = func(*config.ResolvedKubernetesConfig, string) (*kubernetes.Client, error) { return client, nil }
	t.Cleanup(func() { newClusterClient = prev })
	kubeconfig := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfig, []byte("apiVersion: v1\nkind: Config\n"), 0o600))
	return runControllerInstall(context.Background(), &config.GlobalConfig{}, &cmdutil.K8sFlags{Kubeconfig: kubeconfig}, installFlags{crdsOnly: true, timeout: 0})
}

// The command prints the hint after a guard refusal, last, and exits 2; with no
// earlier Deployment it prints none.
func TestRunControllerInstall_CRDsOnlyRefusalHint(t *testing.T) {
	foreignCRD := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1", "kind": "CustomResourceDefinition",
		"metadata": map[string]any{"name": ctrllib.CRDNames()[0]},
	}}
	cases := []struct {
		name       string
		deployment bool
	}{
		{"earlier Deployment present", true},
		{"no earlier Deployment", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := runCRDsOnly(t, earlierCluster(c.deployment, foreignCRD))
			var exitErr *opmexit.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
			var guardErr *ctrllib.GuardError
			require.ErrorAs(t, err, &guardErr)
			assert.Contains(t, err.Error(), "exists and is not managed by OPM")
			if c.deployment {
				assert.Contains(t, err.Error(), "\nhint: "+earlierNoticeStart)
			} else {
				assert.NotContains(t, err.Error(), "hint:")
			}
		})
	}
}

// When the checks pass the command prints one warning before its first write,
// and a failure of the writes that follow keeps its own exit code.
func TestRunControllerInstall_CRDsOnlyWarnsBeforeTheFirstWrite(t *testing.T) {
	logs := captureLogs(t)
	client := earlierCluster(true)
	var warnedAtFirstWrite *bool
	client.Dynamic.(*fakedynamic.FakeDynamicClient).PrependReactor("*", "*", func(a k8stesting.Action) (bool, runtime.Object, error) {
		switch a.GetVerb() {
		case "get", "list":
		default:
			if warnedAtFirstWrite == nil {
				warned := strings.Contains(logs.String(), earlierNoticeStart)
				warnedAtFirstWrite = &warned
			}
		}
		return false, nil, nil
	})

	err := runCRDsOnly(t, client)

	assert.Contains(t, logs.String(), earlierNoticeStart, "the warning is printed")
	require.NotNil(t, warnedAtFirstWrite, "the install went on to write")
	assert.True(t, *warnedAtFirstWrite, "the warning came before the first write")
	assert.Equal(t, 1, strings.Count(logs.String(), earlierNoticeStart))
	var guardErr *ctrllib.GuardError
	assert.NotErrorAs(t, err, &guardErr, "the checks passed")
	if err != nil {
		var exitErr *opmexit.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.NotContains(t, err.Error(), earlierNoticeStart, "the warning is not part of the error")
	}
}
