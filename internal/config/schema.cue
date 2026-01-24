// Config schema for OPM CLI configuration validation.
package config

#Config: {
	// kubeconfig is the path to the kubeconfig file.
	kubeconfig?: string

	// context is the Kubernetes context to use.
	context?: string

	// namespace must be a valid Kubernetes namespace name.
	// RFC 1123 subdomain: lowercase alphanumeric with hyphens, max 63 chars.
	namespace?: string & =~"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"

	// registry is the default OCI registry URL.
	registry?: string

	// cacheDir is the local cache directory path.
	cacheDir?: string
}
