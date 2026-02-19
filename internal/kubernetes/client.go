// Package kubernetes provides Kubernetes cluster operations for OPM modules.
// It handles server-side apply, label-based resource discovery, and deletion.
package kubernetes

import (
	"context"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	oerrors "github.com/opmodel/cli/internal/errors"
)

// ClientOptions configures Kubernetes client creation.
// All fields must be pre-resolved by the caller (via config.ResolveKubernetes).
// No further precedence resolution is performed inside the client.
type ClientOptions struct {
	// Kubeconfig is the pre-resolved path to the kubeconfig file.
	// Empty string means use the default kubeconfig discovery (KUBECONFIG env / ~/.kube/config).
	Kubeconfig string

	// Context is the pre-resolved Kubernetes context name.
	// Empty string means use the current-context from kubeconfig.
	Context string

	// APIWarnings controls how K8s API warnings are handled.
	// Valid values: "warn", "debug", "suppress". Default: "warn"
	APIWarnings string
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

	// Set custom warning handler based on config
	warningLevel := opts.APIWarnings
	if warningLevel == "" {
		warningLevel = "warn" // default
	}
	restConfig.WarningHandler = &opmWarningHandler{level: warningLevel, logger: outputWarningLogger{}}

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

// EnsureNamespace checks if a namespace exists and creates it if missing.
// Returns true if the namespace was created, false if it already existed.
// When dryRun is true, the namespace is not actually created.
func (c *Client) EnsureNamespace(ctx context.Context, name string, dryRun bool) (bool, error) {
	_, err := c.Clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return false, nil // already exists
	}

	if !apierrors.IsNotFound(err) {
		return false, fmt.Errorf("checking namespace %q: %w", name, err)
	}

	if dryRun {
		return true, nil
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err = c.Clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		// Handle race condition: another process may have created it
		if apierrors.IsAlreadyExists(err) {
			return false, nil
		}
		return false, fmt.Errorf("creating namespace %q: %w", name, err)
	}

	return true, nil
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

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		overrides,
	)

	return clientConfig.ClientConfig()
}

// resolveKubeconfig resolves kubeconfig path with precedence:
// flag > OPM_KUBECONFIG > KUBECONFIG > ~/.kube/config
func resolveKubeconfig(flagValue string) string {
	var path string

	if flagValue != "" {
		path = flagValue
	} else if v := os.Getenv("OPM_KUBECONFIG"); v != "" {
		path = v
	} else if v := os.Getenv("KUBECONFIG"); v != "" {
		path = v
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, ".kube", "config")
	}

	// Expand tilde in all paths (defensive, in case config resolver didn't)
	return config.ExpandTilde(path)
}
