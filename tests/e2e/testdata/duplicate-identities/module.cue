// duplicate-identities — a module whose two components render two objects
// under one Kubernetes apply identity, so every render of it must be refused.
// It is the only module in the tree that collides, and it is never published:
// it exists so the e2e suite can prove the refusal reaches a whole command,
// not just the helper call.
package dupidentities

import (
	"strings"

	m "opmodel.dev/core@v2"
	res "opmodel.dev/catalogs/opm/resources/v1beta1"

	id "test.example.com/dupidentities/identity"
)

m.#Module

// Module metadata — modulePath and version are the identity package's values,
// and name is the path's leaf (0010:D8, 0011:D12).
metadata: {
	_segments:   strings.Split(strings.SplitN(id.ModulePath, "@", 2)[0], "/")
	name:        _segments[len(_segments)-1]
	modulePath:  id.ModulePath
	version:     id.Version
	description: "Two components rendering one apply identity — refusal fixture"
}

#config: {
	// Container image shared by both colliding components.
	image: res.#Image & {repository: string | *"ghcr.io/stefanprodan/podinfo", tag: string | *"6.7.1", digest: string | *""}
}

debugValues: {
	image: {repository: "ghcr.io/stefanprodan/podinfo", tag: "6.7.1", digest: ""}
}
