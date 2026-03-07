// Values provide concrete configuration for the game-stack bundle.
// These satisfy the #config schema defined in bundle.cue.
package gamestack

// Concrete values — production-style game server
values: {
	// Number of concurrent players (proxy + server must match)
	maxPlayers: 50

	// Server list description shown to players
	motd: "Welcome to My Minecraft Server!"

	// Target namespace for all instances
	namespace: "game-stack"

	// Shared forwarding secret (proxy <-> server authentication)
	forwardingSecret: "change-me-in-production"

	// RCON password — resolved as a #SecretLiteral here.
	// Consumers with existing K8s secrets or ESO stores can use
	// the other variants instead:
	//   K8s ref: rconPassword: secretName: "my-secret", remoteKey: "rcon-pw"
	//   ESO ref: rconPassword: externalPath: "aws/secrets/mc", remoteKey: "rcon-password"
	rconPassword: value: "change-me-in-production"
}
