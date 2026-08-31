// Local default platform for the kind dev cluster tooling.
// Resolved as the sibling of hack/opm-config.cue (config.PlatformFilePath).
//
// This is the D21 precedence source 3 — used only by offline commands
// (`opm instance build`/`vet`) which never read the cluster. Cluster-facing
// commands resolve the Platform CR instead. The subscription pins are kept
// identical to hack/kind-platform.yaml and internal/config/templates.go
// (DefaultPlatformTemplate) so an offline build and an in-cluster render
// materialize the same catalog builds; a drift here would show up as a
// render-digest difference with no obvious cause.
//
// Data only — CUE imports are not allowed in this file.

name: "cluster"
type: "kubernetes"

registry: {
	"opmodel.dev/catalogs/opm@v4": {
		version: "4.0.1"
	}
	"opmodel.dev/catalogs/k8s@v1": {
		version: "1.0.0-alpha.2"
	}
}
