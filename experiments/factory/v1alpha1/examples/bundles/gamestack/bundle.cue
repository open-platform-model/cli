// Package gamestack defines the game-stack bundle.
//
// Bundles a dynamic Minecraft server fleet with optional Velocity proxy networking,
// automatic TCP hostname routing (mc-router), and unified metrics export (mc-monitor).
//
// ## Two deployment modes
//
// Standalone servers (`servers` map)
//   Each server is independently accessible via a dedicated hostname:
//     {serverName}.{domain}  →  mc-router  →  Minecraft server
//
// Proxied network (`network` block)
//   A single hostname fronts a Velocity proxy that connects backend servers.
//   Players join at {network.hostname}.{domain} and switch worlds in-game:
//     {network.hostname}.{domain}  →  mc-router  →  Velocity  →  backend servers
//
// Both modes can be used simultaneously in the same release.
//
// ## Auto-wiring
//
// Adding a server to either map automatically:
//   - Adds a --mapping entry to mc-router
//   - Adds a javaServers entry to mc-monitor
//
// mc-router and mc-monitor instances are always present.
// The Velocity proxy instance is created only when `network` is set.
//
// ## Service DNS convention
//
// This bundle uses a `releaseName` config field to compute K8s service DNS names.
// The BundleRelease produces ModuleRelease names of the form:
//   {releaseName}-{instanceName}
// The K8s Service for each server is reachable at:
//   {releaseName}-{serverName}.{namespace}.svc
// The Velocity proxy service:
//   {releaseName}-proxy.{namespace}.svc
//
// Set `releaseName` to exactly match the BundleRelease `metadata.name`.
package gamestack

import (
	"list"

	bundle  "opmodel.dev/core/bundle@v1"
	schemas "opmodel.dev/schemas@v1"
	mc      "opmodel.dev/examples/modules/minecraft@v1"
	vel     "opmodel.dev/examples/modules/velocity@v1"
	rtr     "opmodel.dev/examples/modules/mc-router@v1:mcrouter"
	mon     "opmodel.dev/examples/modules/mc-monitor@v1:mcmonitor"
)

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

// Bundle definition
bundle.#Bundle

// Bundle metadata
metadata: {
	modulePath:  "example.com/bundles"
	name:        "gamestack"
	version:     "v1"
	description: "Dynamic Minecraft server fleet with mc-router, optional Velocity proxy, and mc-monitor"
}

// Bundle-level config schema.
// Consumer fills these in when creating a BundleRelease.
//
// The C= alias captures bundle #config at package scope so it can be
// referenced inside #instances blocks without colliding with module-level #config.
C=#config: {
	// === Routing ===

	// Base domain for all server hostnames.
	// Example: "mc.example.com"
	//   → standalone "lobby" server: lobby.mc.example.com
	//   → network proxy "play":      play.mc.example.com
	domain: string

	// Must match the BundleRelease metadata.name exactly.
	// Used to compute K8s service DNS names:
	//   {releaseName}-{serverName}.{namespace}.svc
	releaseName: string

	// Kubernetes namespace for all module instances
	namespace: string | *"game-stack"

	// === Standalone Servers ===
	// Each entry produces one Minecraft server accessible at {name}.{domain}.
	// Players connect directly — no proxy in the path.
	// Can be combined with `network` in the same release.
	servers?: [string]: _#serverConfig

	// === Proxied Network ===
	// Optional Velocity proxy fronting a group of backend servers.
	// Players join at {network.hostname}.{domain} and switch worlds in-game.
	// Omit to use standalone-only mode.
	network?: {
		// Public hostname prefix for the Velocity proxy.
		// Full hostname: {hostname}.{domain}  (e.g. "play.mc.example.com")
		hostname: string | *"play"

		// Velocity proxy settings
		proxy: {
			// Message of the day shown on the Velocity server list
			motd: string | *"A Minecraft Network"

			// Maximum total players across the proxy
			maxPlayers: uint & >0 & <=10000 | *500

			// Player info forwarding mode to backend servers.
			// MODERN:  Velocity native forwarding (recommended).
			// LEGACY:  BungeeCord-compatible (requires Spigot/Paper backend).
			// NONE:    No forwarding.
			forwardingMode: *"MODERN" | "LEGACY" | "NONE"

			// Forwarding secret — shared between Velocity and all backend servers.
			// Required when forwardingMode is MODERN.
			forwardingSecret: string | *"changeme"
		}

		// Backend Minecraft servers behind this proxy.
		// Each entry produces one Minecraft server instance.
		// In MODERN forwarding mode, `onlineMode` is automatically set to false on
		// backends so Velocity handles authentication instead.
		servers: [string]: _#serverConfig
	}

	// === Shared Secret ===
	// RCON password shared across all Minecraft server instances.
	// Embedding mc.#config.rcon.password inherits $secretName/$dataKey routing
	// metadata from the module — the bundle never duplicates them.
	rconPassword: schemas.#Secret & mc.#config.rcon.password
}

// _#serverConfig — full per-server configuration.
// Copied from the minecraft module's #config so the bundle can define its own
// defaults independently. The rcon.password field is always overridden at the
// instance level by the bundle-level C.rconPassword.
_#serverConfig: {
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
	// The matchN(<=1, [...]) constraint enforces that at most one is chosen.
	// Defaults to vanilla behaviour when none is set.
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
	// Note: password is absent here — always injected from C.rconPassword at instance level.
	rcon: {
		enabled: bool | *true
		port:    _#portSchema | *25575
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

// debugValues exercises the full #config surface for local cue vet / cue eval.
// Both modes active: standalone servers + proxied network.
debugValues: {
	domain:      "mc.example.com"
	releaseName: "debug-stack"
	namespace:   "debug"

	// Standalone servers — direct hostname per server
	servers: {
		creative: {
			motd:       "Creative Mode"
			maxPlayers: 10
			mode:       "creative"
			pvp:        false
			difficulty: "peaceful"
			paper: {}
		}
		minigames: {
			motd:       "Minigames!"
			maxPlayers: 20
			fabric: {
				loaderVersion: "0.15.11"
				mods: {
					modrinth: {
						projects: ["some-minigame-mod"]
					}
				}
			}
		}
	}

	// Proxied network — Velocity fronting lobby + survival
	network: {
		hostname: "play"
		proxy: {
			motd:             "Debug Network"
			maxPlayers:       100
			forwardingMode:   "MODERN"
			forwardingSecret: "debug-forwarding-secret"
		}
		servers: {
			lobby: {
				motd:       "Hub"
				maxPlayers: 10
				mode:       "adventure"
				pvp:        false
				difficulty: "peaceful"
				paper: {}
			}
			survival: {
				maxPlayers: 10
				difficulty: "hard"
			}
		}
	}

	rconPassword: value: "debug-rcon-password"
}

// Bundle instances.
//
// Dynamic generation pattern — instances are derived from config maps:
//   - C.servers:         one mc instance per entry (standalone)
//   - C.network.servers: one mc instance per entry (proxied backend)
//   - proxy:             one velocity instance (only when C.network is set)
//   - router:            one mc-router (always; mappings auto-built from server maps)
//   - monitor:           one mc-monitor (always; javaServers auto-built from server maps)
//
// See authoring-guide.md §"Dynamic Instance Generation" for the full pattern.
#instances: {
	// Pre-computed shared bindings — avoids repeating C.xxx in every comprehension
	// and ensures string interpolation has concrete values.
	let _domain  = C.domain
	let _relName = C.releaseName
	let _ns      = C.namespace

	// ── Standalone Minecraft servers ──────────────────────────────────────────
	// One ModuleRelease per entry in C.servers.
	// `*C.servers | {}` falls back to empty struct when servers is absent — safe for `for`.
	// The `*` default marker disambiguates the disjunction when servers IS present.
	for _name, _cfg in (*C.servers | {}) {
		let _c = _cfg
		"\(_name)": {
			module: mc
			metadata: namespace: _ns
			// Embed the full per-server config and override the bundle-level RCON secret.
			// Standalone servers always authenticate directly against Mojang (onlineMode
			// defaults to true in _#serverConfig and is not overridden here).
			values: _c & {
				rcon: password: C.rconPassword
			}
		}
	}

	// ── Proxied network: backend servers + Velocity proxy ─────────────────────
	// Only generated when C.network is set.
	if C.network != _|_ {
		let _network = C.network

		// Backend servers — onlineMode forced false for MODERN forwarding so
		// Velocity handles authentication and forwards player identity.
		for _name, _cfg in _network.servers {
			let _c          = _cfg
			let _onlineMode = _network.proxy.forwardingMode != "MODERN"
			"\(_name)": {
				module: mc
				metadata: namespace: _ns
				// Embed the full per-server config, then apply bundle-level overrides:
				//   - rcon.password: shared bundle secret
				//   - server.onlineMode: driven by proxy forwarding mode
				values: _c & {
					server: onlineMode: _onlineMode
					rcon:   password:   C.rconPassword
				}
			}
		}

		// Velocity proxy — fronts all network backend servers
		proxy: {
			module: vel
			metadata: namespace: _ns
			values: {
				motd:             _network.proxy.motd
				maxPlayers:       _network.proxy.maxPlayers
				forwardingMode:   _network.proxy.forwardingMode
				forwardingSecret: _network.proxy.forwardingSecret
			}
		}
	}

	// ── mc-router ─────────────────────────────────────────────────────────────
	// Always present. Static --mapping args auto-built from both server maps:
	//   standalone: {name}.{domain}     → {releaseName}-{name}.{namespace}.svc
	//   network:    {hostname}.{domain} → {releaseName}-proxy.{namespace}.svc
	let _standaloneMappings = [ for _name, _cfg in (*C.servers | {}) {
		{
			externalHostname: "\(_name).\(_domain)"
			host:             "\(_relName)-\(_name).\(_ns).svc"
			port:             _cfg.port
		}
	}]

	// Conditional single-element list: [{...}] when network is set, [] otherwise.
	let _proxyMapping = [
		if C.network != _|_ {
			{
				externalHostname: "\(C.network.hostname).\(_domain)"
				host:             "\(_relName)-proxy.\(_ns).svc"
				port:             25577
			}
		},
	]

	router: {
		module: rtr
		metadata: namespace: _ns
		values: {
			router: mappings: list.Concat([_standaloneMappings, _proxyMapping])
			port:        25565
			serviceType: "LoadBalancer"
		}
	}

	// ── mc-monitor ────────────────────────────────────────────────────────────
	// Always present. Monitors ALL Minecraft servers directly (standalone + backends).
	// mc-monitor bypasses the proxy so each server is checked individually.
	//
	// `C.network.servers | {}` safely returns {} when C.network is absent.
	let _standaloneTargets = [ for _name, _cfg in (*C.servers | {}) {
		{
			host: "\(_relName)-\(_name).\(_ns).svc"
			port: _cfg.port
		}
	}]

	let _networkTargets = [ for _name, _cfg in (*C.network.servers | {}) {
		{
			host: "\(_relName)-\(_name).\(_ns).svc"
			port: _cfg.port
		}
	}]

	monitor: {
		module: mon
		metadata: namespace: _ns
		values: {
			javaServers: list.Concat([_standaloneTargets, _networkTargets])
			prometheus: {}
			serviceType: "ClusterIP"
		}
	}
}
