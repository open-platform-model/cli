// Package main defines the Minecraft Bedrock Edition server module.
// A stateful application using itzg/minecraft-bedrock-server:
// - module.cue: metadata and config schema
// - components.cue: component definitions with server container
// - values.cue: default values
//
// Config schema mirrors the itzg/minecraft-bedrock-server environment variable surface area.
// Bedrock Edition uses UDP port 19132 and does not support RCON or backup sidecars.
package main

import (
	"opmodel.dev/core@v1"
	schemas "opmodel.dev/schemas@v1"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	modulePath:       "example.com/modules"
	name:             "minecraft-bedrock"
	version:          "0.1.0"
	description:      "Minecraft Bedrock Edition dedicated server"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// === Server Configuration ===
	server: {
		// Container image for Minecraft Bedrock server
		image: schemas.#Image

		// Bedrock server version (e.g., "LATEST", "1.20.51.01")
		version: string

		// EULA acceptance (required by Mojang/Microsoft)
		eula: bool

		// Game difficulty
		difficulty: "peaceful" | "easy" | "normal" | "hard"

		// Game mode
		gameMode: "survival" | "creative" | "adventure" | "spectator"

		// Maximum number of concurrent players
		maxPlayers: int & >0 & <=1000

		// Optional: View distance (in chunks)
		viewDistance?: int & >0 & <=32

		// Optional: Tick distance (4-12 for Bedrock)
		tickDistance?: int & >0

		// Default permission level for new players
		defaultPermission: "visitor" | "member" | "operator"

		// Optional: Idle timeout in minutes (0 = disabled)
		playerIdleTimeout?: int & >=0

		// Optional: Level type
		levelType?: "DEFAULT" | "FLAT" | "LEGACY"

		// Optional: Level name
		levelName?: string

		// Optional: Level seed
		levelSeed?: string

		// Optional: Server name displayed in server list
		serverName?: string

		// Optional: Require resource packs
		texturepackRequired?: bool

		// Optional: Verify player identity with Xbox Live
		onlineMode?: bool

		// Optional: Maximum server threads
		maxThreads?: int & >0

		// Optional: Allow cheats
		cheats?: bool

		// Optional: Emit server telemetry
		emitServerTelemetry?: bool

		// Optional: Enable LAN visibility
		enableLanVisibility?: bool

		// Optional: Enable whitelist
		whitelist?: bool

		// Optional: Whitelisted usernames (comma-separated)
		whitelistUsers?: string

		// Optional: Operator XUIDs (comma-separated)
		ops?: string

		// Optional: Member XUIDs (comma-separated)
		members?: string

		// Optional: Visitor XUIDs (comma-separated)
		visitors?: string

		// Optional: Enable SSH for remote access
		enableSSH?: bool

		// Server port (UDP)
		serverPort: int & >0 & <=65535
	}

	// === Storage Configuration ===
	storage: {
		// Game data volume (worlds, configs)
		data: {
			type: "pvc" | "hostPath" | "emptyDir"

			// For PVC
			size?:         string
			storageClass?: string

			// For hostPath
			path?:         string
			hostPathType?: "Directory" | "DirectoryOrCreate"
		}
	}

	// === Resource Limits (catalog-standard shape) ===
	resources?: schemas.#ResourceRequirementsSchema

	// === Security Context ===
	securityContext?: schemas.#SecurityContextSchema

	// === Networking ===
	// Service type for network exposure
	serviceType: "ClusterIP" | "LoadBalancer" | "NodePort"
}
