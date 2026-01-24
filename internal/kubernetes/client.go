// Package kubernetes provides Kubernetes client operations for the OPM CLI.
package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// ErrNoKubeconfig is returned when kubeconfig cannot be found.
	ErrNoKubeconfig = errors.New("kubeconfig not found")
	// ErrClusterUnreachable is returned when the cluster cannot be reached.
	ErrClusterUnreachable = errors.New("cluster unreachable")
)

// Client provides Kubernetes operations.
type Client struct {
	// Dynamic is the dynamic client for unstructured resources.
	Dynamic dynamic.Interface

	// Discovery provides discovery of API resources.
	Discovery discovery.DiscoveryInterface

	// Mapper maps GVKs to GVRs.
	Mapper meta.RESTMapper

	// Config is the REST config.
	Config *rest.Config

	// DefaultNamespace is the namespace to use if not specified.
	DefaultNamespace string
}

// ClientOptions configures Client creation.
type ClientOptions struct {
	// Kubeconfig is the path to the kubeconfig file.
	// If empty, uses default kubeconfig locations.
	Kubeconfig string

	// Context is the kubeconfig context to use.
	// If empty, uses the current context.
	Context string

	// Namespace is the default namespace.
	// If empty, uses "default".
	Namespace string
}

// NewClient creates a new Kubernetes client.
func NewClient(opts ClientOptions) (*Client, error) {
	config, namespace, err := buildConfig(opts)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating discovery client: %w", err)
	}

	// Use cached discovery to reduce API calls
	cachedDiscovery := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscovery)

	return &Client{
		Dynamic:          dynamicClient,
		Discovery:        discoveryClient,
		Mapper:           mapper,
		Config:           config,
		DefaultNamespace: namespace,
	}, nil
}

// NewClientFromConfig creates a client from an existing REST config.
// Useful for testing with envtest.
func NewClientFromConfig(config *rest.Config, namespace string) (*Client, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating discovery client: %w", err)
	}

	cachedDiscovery := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscovery)

	if namespace == "" {
		namespace = "default"
	}

	return &Client{
		Dynamic:          dynamicClient,
		Discovery:        discoveryClient,
		Mapper:           mapper,
		Config:           config,
		DefaultNamespace: namespace,
	}, nil
}

// CheckConnection verifies connectivity to the cluster.
func (c *Client) CheckConnection(ctx context.Context) error {
	_, err := c.Discovery.ServerVersion()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrClusterUnreachable, err)
	}
	return nil
}

// buildConfig builds the REST config from options.
func buildConfig(opts ClientOptions) (*rest.Config, string, error) {
	kubeconfigPath := opts.Kubeconfig
	if kubeconfigPath == "" {
		kubeconfigPath = defaultKubeconfig()
	}

	if kubeconfigPath == "" {
		return nil, "", ErrNoKubeconfig
	}

	// Build config loading rules
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: kubeconfigPath,
	}

	// Build config overrides
	overrides := &clientcmd.ConfigOverrides{}
	if opts.Context != "" {
		overrides.CurrentContext = opts.Context
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		overrides,
	)

	// Get the namespace from context if not specified
	namespace := opts.Namespace
	if namespace == "" {
		var err error
		namespace, _, err = clientConfig.Namespace()
		if err != nil || namespace == "" {
			namespace = "default"
		}
	}

	// Build REST config
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("building client config: %w", err)
	}

	return config, namespace, nil
}

// defaultKubeconfig returns the default kubeconfig path.
func defaultKubeconfig() string {
	// Check KUBECONFIG environment variable
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	// Check ~/.kube/config
	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
