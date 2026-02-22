// External values file — lives OUTSIDE the module directory.
// Simulates a file passed via --values / -f flag.
// Package name does not matter here; ctx.CompileBytes handles it standalone.
package main

values: {
	serverType: "FABRIC"
	port:       25565
	maxPlayers: 10
}
