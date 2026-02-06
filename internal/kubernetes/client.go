// Package kubernetes provides Kubernetes cluster operations for OPM modules.
// It handles server-side apply, label-based resource discovery, and deletion.
package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	oerrors "github.com/opmodel/cli/internal/errors"
)

// ClientOptions configures Kubernetes client creation.
type ClientOptions struct {
	// Kubeconfig is the path to the kubeconfig file.
	// Precedence: this field > OPM_KUBECONFIG env > KUBECONFIG env > ~/.kube/config
	Kubeconfig string

	// Context is the Kubernetes context to use.
	// If empty, uses the current-context from kubeconfig.
	Context string
}

// Client wraps Kubernetes API clients for OPM operations.
type Client struct {
	// Dynamic is used for server-side apply and resource operations.
	Dynamic dynamic.Interface

	// Clientset is used for discovery and API group listing.
	Clientset kubernetes.Interface

	// RestConfig is the underlying REST configuration.
	RestConfig *rest.Config
}

// cachedClient stores the singleton client for reuse within a command.
var (
	cachedClient *Client
	clientMu     sync.Mutex
)

// NewClient creates a Kubernetes client with the given options.
// The client is cached for reuse within the same command invocation.
func NewClient(opts ClientOptions) (*Client, error) {
	clientMu.Lock()
	defer clientMu.Unlock()

	if cachedClient != nil {
		return cachedClient, nil
	}

	restConfig, err := buildRestConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("building kubernetes config: %w",
			oerrors.Wrap(oerrors.ErrConnectivity, err.Error()))
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w",
			oerrors.Wrap(oerrors.ErrConnectivity, err.Error()))
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating clientset: %w",
			oerrors.Wrap(oerrors.ErrConnectivity, err.Error()))
	}

	cachedClient = &Client{
		Dynamic:    dynamicClient,
		Clientset:  clientset,
		RestConfig: restConfig,
	}

	return cachedClient, nil
}

// ResetClient clears the cached client. Used for testing.
func ResetClient() {
	clientMu.Lock()
	defer clientMu.Unlock()
	cachedClient = nil
}

// buildRestConfig resolves kubeconfig with precedence:
// flag > OPM_KUBECONFIG env > KUBECONFIG env > ~/.kube/config
func buildRestConfig(opts ClientOptions) (*rest.Config, error) {
	kubeconfigPath := resolveKubeconfig(opts.Kubeconfig)

	loadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: kubeconfigPath,
	}

	overrides := &clientcmd.ConfigOverrides{}
	if opts.Context != "" {
		overrides.CurrentContext = opts.Context
	}

	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		overrides,
	)

	return config.ClientConfig()
}

// resolveKubeconfig resolves kubeconfig path with precedence:
// flag > OPM_KUBECONFIG > KUBECONFIG > ~/.kube/config
func resolveKubeconfig(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if v := os.Getenv("OPM_KUBECONFIG"); v != "" {
		return v
	}

	if v := os.Getenv("KUBECONFIG"); v != "" {
		return v
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}
