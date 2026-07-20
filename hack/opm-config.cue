// OPM CLI configuration for the local kind dev cluster.
// Used by: task cluster:operator (via --config), and available to any manual
// `opm ... --config hack/opm-config.cue` invocation against kind-opm-dev.
//
// This exists so the dev-cluster tooling is hermetic: it does not depend on a
// developer's personal ~/.opm being present, current, or in the post-D39
// data-only format. Data only — CUE imports are not allowed here.
package config

config: {
	// Host-side spelling of the local registry. The operator running inside the
	// cluster uses a different address for the same registry — see
	// KIND_CUE_REGISTRY in Taskfile.yml.
	registry: "testing.opmodel.dev=localhost:5000+insecure,opmodel.dev=localhost:5000+insecure,registry.cue.works"

	kubernetes: {
		kubeconfig: "~/.kube/config"
		context:    "kind-opm-dev"
		namespace:  "default"
	}

	log: {
		timestamps: true
		kubernetes: apiWarnings: "debug"
	}
}
