// OPM CLI configuration for the local kind dev cluster.
// Used by: task cluster:operator (via --config), and available to any manual
// `opm ... --config hack/opm-config.cue` invocation against kind-opm-dev.
//
// This exists so the dev-cluster tooling is hermetic: it does not depend on a
// developer's personal ~/.opm being present, current, or in the post-D39
// data-only format. Data only — CUE imports are not allowed here.
package config

config: {
	// Both domains resolve from GHCR: opmodel.dev carries core and the catalogs,
	// testing.opmodel.dev this repo's own fixture modules (published by
	// .github/workflows/publish-fixtures.yml). Matches the shipped default in
	// internal/config/templates.go. To point the kind flow at a local registry
	// instead, set KIND_CUE_REGISTRY — see Taskfile.yml.
	registry: "testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"

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
