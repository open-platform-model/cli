// Package main defines the Minecraft Java Edition server module with automated backup support.
// A stateful application using itzg/minecraft-server and itzg/mc-backup:
// - module.cue: metadata and config schema
// - components.cue: component definitions with server + backup sidecar
// - values.cue: default values
//
// Config schema mirrors the itzg/minecraft-server Helm chart values.yaml surface area.
package minecraft

import (
	m "opmodel.dev/core/module@v1"
	schemas "opmodel.dev/schemas@v1"
)

// Module definition
m.#Module

// Module metadata
metadata: {
	modulePath:       "example.com/modules"
	name:             "minecraft-java"
	version:          "0.3.0"
	description:      "Minecraft Java Edition server with automated backup support"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// === Container Image ===
	// Container image for the Minecraft server
	image: schemas.#Image & {
		repository: string | *"itzg/minecraft-server"
		tag:        string | *"java21"
		digest:     string | *""
	}

	// === Game Version ===
	// Minecraft version (e.g., "1.20.4", "LATEST", "SNAPSHOT")
	version: string | *"LATEST"

	// === EULA ===
	// EULA acceptance (required by Mojang/Microsoft)
	eula: bool | *true

	// === Server Type ===
	// Set exactly ONE of the following to select your server software.
	// The matchN(1, [...]) constraint enforces that only one is chosen.
	//
	// Example — Paper (auto-downloads latest):
	//   paper: {}
	//
	// Example — Forge with specific version:
	//   forge: { version: "47.2.0" }
	//
	// Example — Vanilla:
	//   vanilla: {}

	// VANILLA — unmodified Mojang server
	vanilla?: {}

	// PAPER — Paper server (high-performance Spigot fork)
	paper?: {
		downloadUrl?: string
		plugins?:     _#pluginsConfig
	}

	// FORGE — Minecraft Forge modded server
	forge?: {
		version:       string
		installerUrl?: string
		mods?:         _#modsConfig
	}

	// FABRIC — Fabric modded server
	fabric?: {
		loaderVersion: string
		installerUrl?: string
		mods?:         _#modsConfig
	}

	// SPIGOT — Spigot server
	spigot?: {
		downloadUrl?: string
		plugins?:     _#pluginsConfig
	}

	// BUKKIT — CraftBukkit server
	bukkit?: {
		downloadUrl?: string
		plugins?:     _#pluginsConfig
	}

	// SPONGEVANILLA — SpongeVanilla server
	sponge?: {
		version: string
	}

	// PURPUR — Purpur server (Paper fork with extra features)
	purpur?: {
		plugins?: _#pluginsConfig
	}

	// MAGMA — Magma server (Forge + Bukkit/Spigot API hybrid)
	magma?: {
		mods?:    _#modsConfig
		plugins?: _#pluginsConfig
	}

	// FTBA — Feed The Beast modpack server
	ftba?: {
		mods?: _#modsConfig
	}

	// AUTO_CURSEFORGE — CurseForge modpack server
	autoCurseForge?: {
		// CurseForge API key — stored in a K8s Secret
		apiKey: schemas.#Secret & {
			$secretName: "server-secrets"
			$dataKey:    "cf-api-key"
		}
		pageUrl?:         string
		slug?:            string
		fileId?:          string
		filenameMatcher?: string
		excludeMods?: [...string]
		includeMods?: [...string]
		forceSynchronize?: bool
		parallelDownloads: uint | *1
	}

	matchN(<=1, [
		{vanilla!: _},
		{paper!: _},
		{forge!: _},
		{fabric!: _},
		{spigot!: _},
		{bukkit!: _},
		{sponge!: _},
		{purpur!: _},
		{magma!: _},
		{ftba!: _},
		{autoCurseForge!: _},
	])

	// === JVM Configuration ===
	jvm: {
		// JVM memory allocation (e.g., "1024M", "2G")
		memory: string | *"2G"

		// General JVM options
		opts?: string

		// JVM -XX options (precede general options)
		xxOpts?: string

		// Enable Aikar's optimized GC flags for Minecraft
		// See: https://aikar.co/2018/07/02/tuning-the-jvm-g1gc-garbage-collector-flags-for-minecraft/
		useAikarFlags: bool | *true
	}

	// === Server Properties ===
	// Fields that map directly to Minecraft server.properties.
	server: {
		// Optional: Server message of the day
		motd?: string | *"OPM Minecraft Java Server"

		// Maximum number of concurrent players
		maxPlayers: uint & >0 & <=1000 | *10

		// Game difficulty
		difficulty: "peaceful" | "easy" | *"normal" | "hard"

		// Game mode
		mode: *"survival" | "creative" | "adventure" | "spectator"

		// Enable PvP combat
		pvp: bool | *true

		// Enable command blocks
		enableCommandBlock: bool | *false

		// Optional: List of server operator usernames
		ops?: [...string]

		// Optional: List of blocked usernames
		blocklist?: [...string]

		// Optional: Seed for world generation
		seed?: string

		// Optional: Maximum world size (radius in blocks)
		maxWorldSize?: uint

		// Optional: View distance (in chunks)
		viewDistance: uint & <=32 | *10

		// === World Settings ===
		// Allows players to travel to the Nether
		allowNether: bool | *true

		// Allows server to announce when a player gets an achievement
		announcePlayerAchievements: bool | *true

		// If true, players will always join in the default gameMode
		forceGameMode: bool | *false

		// Defines whether structures (such as villages) will be generated
		generateStructures: bool | *true

		// If set to true, players will be set to spectator mode if they die
		hardcore: bool | *false

		// The maximum height in which building is allowed
		maxBuildHeight: uint | *256

		// Maximum number of milliseconds a single tick may take (-1 to disable)
		maxTickTime: int | *60000

		// Determines if animals will be able to spawn
		spawnAnimals: bool | *true

		// Determines if monsters will be spawned
		spawnMonsters: bool | *true

		// Determines if villagers will be spawned
		spawnNPCs: bool | *true

		// Sets the area that non-ops can not edit (0 to disable)
		spawnProtection: int & >=0 | *0

		// World type: DEFAULT, FLAT, LARGEBIOMES, AMPLIFIED, CUSTOMIZED
		levelType: *"DEFAULT" | "FLAT" | "LARGEBIOMES" | "AMPLIFIED" | "CUSTOMIZED"

		// Name of the world save
		worldSaveName: string | *"world"

		// Check accounts against Minecraft account service
		onlineMode: bool | *true

		// Require public key to be signed by Mojang to join
		enforceSecureProfile: bool | *true

		// Override server.properties even if they already exist
		overrideServerProperties: bool | *true

		// Resource pack
		resourcePackUrl?:     string
		resourcePackSha?:     string
		resourcePackEnforce?: bool

		// VanillaTweaks share codes for datapacks, crafting tweaks, and resource packs
		vanillaTweaksShareCodes?: [...string]
	}

	// === RCON Configuration ===
	rcon: {
		enabled: bool | *true
		// RCON password — stored in a K8s Secret, never in plaintext
		password: schemas.#Secret & {
			$secretName: "server-secrets"
			$dataKey:    "rcon-password"
		}
		port: _#portSchema | *25575
	}

	// === Query Port ===
	// Optional query port for server status queries (e.g., for server lists)
	// If you enable this, your server will be "published" to Gamespy
	query: {
		enabled: bool | *false
		port:    _#portSchema | *25565
	}

	// === World Data ===
	// URL to download a world at startup (any server type)
	downloadWorldUrl?: string

	// === Networking ===
	// Minecraft server port
	port: _#portSchema | *25565

	// Service type for network exposure
	serviceType: *"ClusterIP" | "LoadBalancer" | "NodePort"

	// Extra ports (for plugins like dynmap, bluemap, etc.)
	extraPorts?: [...{
		name:          string
		containerPort: _#portSchema
		protocol:      *"TCP" | "UDP" | "SCTP"
	}]

	// === Storage Configuration ===
	storage: {
		// Game data volume (worlds, configs, plugins)
		data: {
			type: *"pvc" | "hostPath" | "emptyDir"

			// For PVC
			size:          string | *"10Gi"
			storageClass?: string

			// For hostPath
			path?:         string
			hostPathType?: "Directory" | "DirectoryOrCreate"
		}

		// Backup storage volume
		backups: {
			type: *"pvc" | "hostPath" | "emptyDir"

			// For PVC
			size:          string | *"10Gi"
			storageClass?: string

			// For hostPath
			path?:         string
			hostPathType?: "Directory" | "DirectoryOrCreate"
		}
	}

	// === Backup Configuration ===
	backup: {
		// Enable backup sidecar container
		enabled: bool | *false

		// Backup container image
		image: schemas.#Image & {
			repository: string | *"itzg/mc-backup"
			tag:        string | *"latest"
			digest:     string | *""
		}

		// Backup method determines how backups are stored
		method: *"tar" | "rsync" | "restic" | "rclone"

		// Backup interval (e.g., "24h", "2h 30m", "6h")
		interval: string | *"24h"

		// Initial delay before first backup
		initialDelay: string | *"5m"

		// Optional: Delete backups older than N days
		pruneBackupsDays: uint | *7

		// Pause backups if no players online
		pauseIfNoPlayers: bool | *true

		// === Backup Method ===
		// Set exactly ONE of the following to select the backup method.
		// The matchN(1, [...]) constraint enforces that only one is chosen.
		//
		// Example — Tar (compressed archives, default):
		//   tar: {}
		//
		// Example — Rsync (incremental file-level):
		//   rsync: {}
		//
		// Example — Restic (encrypted, deduplicated, to S3/rclone/etc.):
		//   restic: { repository: "s3:s3.amazonaws.com/my-bucket", password: value: "secret" }
		//
		// Example — Rclone (tar + upload to remote):
		//   rclone: { remote: "myremote", destDir: "minecraft-backups" }

		// TAR — compressed tarballs written to DEST_DIR
		tar?: {
			compressMethod:      "gzip" | "bzip2" | "lzip" | "lzma" | "lzop" | *"xz" | "zstd"
			compressParameters?: string
			linkLatest:          bool | *false
		}

		// RSYNC — incremental file-level backup to DEST_DIR
		rsync?: {
			linkLatest: bool | *false
		}

		// RESTIC — encrypted, deduplicated backup to any restic backend
		restic?: {
			// Restic repository URL (e.g. "s3:s3.amazonaws.com/bucket", "rclone:remote:path")
			repository: string
			// Restic repo password — stored in a K8s Secret
			password: schemas.#Secret & {
				$secretName: "backup-secrets"
				$dataKey:    "restic-password"
			}
			retention?:      string
			hostname?:       string
			verbose?:        bool
			additionalTags?: string
			limitUpload?:    uint
			retryLock?:      string
		}

		// RCLONE — tar + upload via rclone to a configured remote
		rclone?: {
			// Name of the rclone remote (from rclone.conf)
			remote: string
			// Directory on the remote to upload backups to
			destDir:        string
			compressMethod: "gzip" | "bzip2" | "lzip" | "lzma" | "lzop" | "xz" | "zstd"
		}

	if enabled {
		matchN(1, [
			{tar!: _},
			{rsync!: _},
			{restic!: _},
			{rclone!: _},
		])
	}
	}

	// === Resource Limits (catalog-standard shape) ===
	resources?: schemas.#ResourceRequirementsSchema

	// === Security Context ===
	securityContext?: schemas.#SecurityContextSchema
}

_#portSchema: uint & >0 & <=65535

// Shared Modrinth auto-download config (works for both mods and plugins)
_#modrinthConfig: {
	projects: [...string]
	downloadDependencies?: "none" | "required" | "optional"
	allowedVersionType?:   "release" | "beta" | "alpha"
}

// Mods config — for mod-based server types (Forge, Fabric, FTB)
_#modsConfig: {
	// List of URLs to mod jar files
	urls?: [...string]

	// Modrinth project auto-download
	modrinth?: _#modrinthConfig

	// URL to a modpack zip to download at startup
	modpackUrl?: string

	// Remove old mods before installing new ones
	removeOldMods: bool | *false
}

// Plugins config — for plugin-based server types (Paper, Spigot, Bukkit, Purpur)
_#pluginsConfig: {
	// List of URLs to plugin jar files
	urls?: [...string]

	// Spigot resource/plugin IDs for auto-download via Spiget
	spigetResources?: [...int]

	// Modrinth project auto-download
	modrinth?: _#modrinthConfig

	// Remove old plugins before installing new ones
	removeOldMods: bool | *false
}

debugValues: {
	// Example config exercising the full schema for local cue vet / cue eval.
	// Demonstrates multiple server types, backup config, mods/plugins, and more.
	version: "1.20.4"
	eula:    true

	paper: {
		plugins: {
			urls: [
				"https://example.com/plugins/EssentialsX.jar",
				"https://example.com/plugins/LuckPerms.jar"
			]
			modrinth: {
				projects: ["some-paper-plugin"]
			}
			removeOldMods: true
		}
	}

	storage: {
		data: {
			type: "pvc"
			size: "20Gi"
		}
		backups: {
			type: "pvc"
			size: "20Gi"
		}
	}

	rcon: {
		password: value: "debug-rcon-password"
	}

	backup: {
		enabled: true
		method:  "restic"
		interval: "12h"
		initialDelay: "10m"
		pruneBackupsDays: 14
		pauseIfNoPlayers: true

		restic: {
			repository: "s3:s3.amazonaws.com/my-minecraft-backups"
			password: value: "supersecretresticpassword"
			retention: "30d"
			additionalTags: "env=prod,type=minecraft-backup"
			retryLock: "10m"
		}
	}
}