// Forge-specific values — conflicts with values.cue (serverType: "PAPER" vs "FORGE").
// This file existing in the module directory is what causes the unification conflict.
package main

values: {
	serverType: "FORGE"
	port:       25565
	maxPlayers: 50
}
