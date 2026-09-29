// Platform module for the kind dev cluster tooling and the offline tests
// (module form, 0019:D5), passed explicitly with --platform. Nothing
// resolves it implicitly: renders resolve --platform, else the cluster
// Platform (hack/kind-platform.yaml), else their own dependency pins.
//
// Catalog builds are pinned in cue.mod/module.cue, never here. Those pins
// MIRROR hack/kind-platform.yaml on purpose, so an offline build and an
// in-cluster render evaluate the same catalog builds; a drift would show up
// as a render-digest difference with no obvious cause. cue.mod is kept in
// `cue mod tidy`'s canonical form (transitive pins included, no comments)
// because the root `task deps:update` tidies it.
package platform

import (
	core "opmodel.dev/core@v2"
	opm "opmodel.dev/catalogs/opm@v4"
	k8s "opmodel.dev/catalogs/k8s@v1"
)

core.#Platform

metadata: name: "cluster"
type: "kubernetes"

#registry: {
	"opmodel.dev/catalogs/opm@v4": #catalog: opm
	"opmodel.dev/catalogs/k8s@v1": #catalog: k8s
}
