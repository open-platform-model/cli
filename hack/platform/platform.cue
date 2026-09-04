// Local default platform for the kind dev cluster tooling (module form,
// 0019 D5), resolved as the sibling platform/ of hack/opm-config.cue
// (config.PlatformDir). This is the D21 precedence source 3 — used only by
// offline commands (`opm instance build`/`vet`) which never read the
// cluster; cluster-facing commands resolve the Platform CR
// (hack/kind-platform.yaml) instead.
//
// Catalog builds are pinned in cue.mod/module.cue, never here. Those pins
// MIRROR internal/config/templates.go's seeded module (DefaultCorePin,
// DefaultCatalogPins) and hack/kind-platform.yaml on purpose, so an offline
// build and an in-cluster render evaluate the same catalog builds; a drift
// would show up as a render-digest difference with no obvious cause
// (TestHackPlatformMirror_PinsMatchSeed guards it). cue.mod is kept in
// `cue mod tidy`'s canonical form (transitive pins included, no comments)
// because the root `task deps:update` tidies it in the same pass as the
// seed.
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
