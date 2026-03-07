// Package gamestack defines the game-stack bundle.
// Combines a Minecraft Java server and a Velocity proxy into a single deployable unit.
//
// Bundle-level config exposes a shared surface that consumers fill in.
// The #BundleRelease comprehension wires bundle values into each module's #config
// at deployment time.
package gamestack

import (
	bundle  "opmodel.dev/core/bundle@v1"
	schemas "opmodel.dev/schemas@v1"
	mc      "opmodel.dev/examples/modules/minecraft@v1"
	vel     "opmodel.dev/examples/modules/velocity@v1"
)

// Bundle definition
bundle.#Bundle

// Bundle metadata
metadata: {
	modulePath:  "example.com/bundles"
	name:        "gamestack"
	version:     "v1"
	description: "Minecraft Java server + Velocity proxy, bundled for easy deployment"
}

// Bundle-level config schema.
// Consumer sets these when creating a BundleRelease.
// The BundleRelease comprehension maps these into each module's #config.
//
// The C alias captures this config at package scope so it can be
// referenced inside module unification blocks (where #config would resolve
// to the module's own #config instead).
C=#config: {
	// Max players — applied to both the Minecraft server and the Velocity proxy
	maxPlayers: uint & >0 & <=10000 | *20

	// Message of the day — shown on the Velocity proxy's server list
	motd: string | *"A Game-Stack Server"

	// Namespace to deploy all instances into
	namespace: string | *"game-stack"

	// Forwarding secret — shared between Velocity and the backend Minecraft server.
	// MODERN forwarding requires this on both proxy and server.
	forwardingSecret: string | *"changeme"

	// RCON password — exposed as a #Secret so consumers can choose how to fulfill it:
	//   Literal:  values: rconPassword: value: "my-password"
	//   K8s ref:  values: rconPassword: secretName: "existing-k8s-secret", remoteKey: "rcon-pw"
	//
	// Embedding mc.#config.rcon.password inherits the module's $secretName and
	// $dataKey routing metadata via CUE unification, keeping the module as the
	// single source of truth for secret routing. The bundle author never needs
	// to duplicate routing metadata — it flows in from the module definition.
	rconPassword: schemas.#Secret & mc.#config.rcon.password
}

// Bundle instances.
// Each entry is a module instance with metadata and explicit values wiring.
// The `values` field sends bundle-level #config fields into each module's #config.
// metadata.name is auto-derived from the map key by the #Bundle constraint.
#instances: {
	// Backend Minecraft server
	server: {
		module: mc
		metadata: namespace: C.namespace
		values: {
			server: {
				maxPlayers: C.maxPlayers
				motd:       C.motd
			}
			rcon: password: C.rconPassword
		}
	}

	// Frontend Velocity proxy
	proxy: {
		module: vel
		metadata: namespace: C.namespace
		values: {
			maxPlayers:       C.maxPlayers
			motd:             C.motd
			forwardingSecret: C.forwardingSecret
		}
	}
}
