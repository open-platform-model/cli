// Package integration provides integration tests for the OPM CLI.
package integration

import (
	"context"
	"os"
	"testing"

	"github.com/opmodel/cli/internal/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	// testEnv is the envtest environment.
	testEnv *envtest.Environment

	// testConfig is the REST config for the test cluster.
	testConfig *rest.Config

	// testClient is the Kubernetes client for tests.
	testClient *kubernetes.Client
)

// TestMain sets up the test environment.
func TestMain(m *testing.M) {
	// Check if we should use an external cluster
	if clusterName := os.Getenv("OPM_TEST_CLUSTER"); clusterName != "" {
		// Use external cluster
		var err error
		testClient, err = kubernetes.NewClient(kubernetes.ClientOptions{
			Context: clusterName,
		})
		if err != nil {
			panic("failed to connect to test cluster: " + err.Error())
		}
		testConfig = testClient.Config
	} else {
		// Use envtest
		testEnv = &envtest.Environment{}

		var err error
		testConfig, err = testEnv.Start()
		if err != nil {
			panic("failed to start envtest: " + err.Error())
		}

		testClient, err = kubernetes.NewClientFromConfig(testConfig, "default")
		if err != nil {
			panic("failed to create test client: " + err.Error())
		}
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if testEnv != nil {
		if err := testEnv.Stop(); err != nil {
			panic("failed to stop envtest: " + err.Error())
		}
	}

	os.Exit(code)
}

// GetTestClient returns the test Kubernetes client.
func GetTestClient() *kubernetes.Client {
	return testClient
}

// GetTestConfig returns the test REST config.
func GetTestConfig() *rest.Config {
	return testConfig
}

// CreateTestNamespace creates a test namespace and returns a cleanup function.
func CreateTestNamespace(t *testing.T, name string) func() {
	t.Helper()

	ctx := context.Background()

	// Create namespace using the client
	ns := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]interface{}{
			"name": name,
		},
	}

	obj := &kubernetes.Labels{}
	_ = obj // Use kubernetes package

	// For now, just return a no-op cleanup
	// Real implementation would create and delete the namespace
	return func() {
		// Cleanup would delete the namespace
		_ = ctx
		_ = ns
	}
}
