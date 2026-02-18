// Package main defines the Minecraft server module with automated backup support.
// A stateful application using itzg/minecraft-server and itzg/mc-backup:
// - module.cue: metadata and config schema
// - components.cue: component definitions with server + backup sidecar
// - values.cue: default values
package main

import (
	"opmodel.dev/core@v0"
	schemas "opmodel.dev/schemas@v0"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	apiVersion:       "example.com/minecraft@v0"
	name:             "minecraft"
	version:          "0.1.0"
	description:      string | *"Minecraft Java Edition server with automated backup support"
	defaultNamespace: "minecraft"
}

// Schema only - constraints for users, no defaults
#config: {
	// === Server Configuration ===
	server: {
		// Container image for Minecraft server
		image: string

		// Server type determines which server software to run
		type: "VANILLA" | "PAPER" | "FORGE" | "FABRIC" | "SPIGOT" | "BUKKIT" | "PURPUR" | "MAGMA"

		// Minecraft version (e.g., "1.20.4", "LATEST", "SNAPSHOT")
		version: string

		// EULA acceptance (required by Mojang/Microsoft)
		eula: bool

		// Optional: Server message of the day
		motd?: string

		// Maximum number of concurrent players
		maxPlayers: int & >0 & <=1000

		// Game difficulty
		difficulty: "peaceful" | "easy" | "normal" | "hard"

		// Game mode
		mode: "survival" | "creative" | "adventure" | "spectator"

		// Enable PvP combat
		pvp: bool

		// Enable command blocks
		enableCommandBlock: bool

		// Optional: List of server operator usernames
		ops?: [...string]

		// Optional: List of whitelisted usernames
		whitelist?: [...string]

		// Optional: Seed for world generation
		seed?: string

		// Optional: Maximum world size (radius in blocks)
		maxWorldSize?: int & >0

		// Optional: View distance (in chunks)
		viewDistance?: int & >0 & <=32

		// RCON configuration for backup coordination
		rcon: {
			password: string
			port:     int & >0 & <=65535
		}
	}

	// === Storage Configuration ===
	storage: {
		// Game data volume (worlds, configs, plugins)
		data: {
			type: "pvc" | "hostPath" | "emptyDir"

			// For PVC
			size?:         string
			storageClass?: string

			// For hostPath
			path?:         string
			hostPathType?: "Directory" | "DirectoryOrCreate"
		}

		// Backup storage volume
		backups: {
			type: "pvc" | "hostPath" | "emptyDir"

			// For PVC
			size?:         string
			storageClass?: string

			// For hostPath
			path?:         string
			hostPathType?: "Directory" | "DirectoryOrCreate"
		}
	}

	// === Backup Configuration ===
	backup?: {
		// Enable backup sidecar container
		enabled: bool

		// Backup container image
		image: string

		// Backup method determines how backups are stored
		method: "tar" | "rsync" | "restic" | "rclone"

		// Backup interval (e.g., "24h", "2h 30m", "6h")
		interval: string

		// Initial delay before first backup
		initialDelay: string

		// Optional: Delete backups older than N days
		pruneBackupsDays?: int & >0

		// Optional: Pause backups if no players online
		pauseIfNoPlayers?: bool

		// Tar-specific configuration
		if method == "tar" {
			tar?: {
				compressMethod: "gzip" | "bzip2" | "lzip" | "lzma" | "lzop" | "xz" | "zstd"
				linkLatest:     bool
			}
		}

		// Restic-specific configuration
		if method == "restic" {
			restic?: {
				repository: string
				password:   string
				retention?: string
				hostname?:  string
				verbose?:   bool
			}
		}

		// Rclone-specific configuration
		if method == "rclone" {
			rclone?: {
				remote:         string
				destDir:        string
				compressMethod: "gzip" | "bzip2" | "lzip" | "lzma" | "lzop" | "xz" | "zstd"
			}
		}
	}

	// === Resource Limits ===
	resources?: schemas.#ResourceRequirementsSchema

	// === Networking ===
	// Minecraft server port
	port: int & >0 & <=65535

	// Service type for network exposure
	serviceType: "ClusterIP" | "LoadBalancer" | "NodePort"
}

// Values must satisfy #config - concrete values in values.cue
values: #config
