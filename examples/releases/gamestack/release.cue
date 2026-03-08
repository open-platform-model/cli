package gamestack

import (
	br "opmodel.dev/core/bundlerelease@v1"
	gs "opmodel.dev/examples/bundles/gamestack@v1"
)

// Bundle release definition
br.#BundleRelease

metadata: {
	name: "my-game-stack"
}

// Reference the game-stack bundle
#bundle: gs
