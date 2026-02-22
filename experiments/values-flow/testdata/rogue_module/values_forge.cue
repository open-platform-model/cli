// rogue_module/values_forge.cue — the rogue file.
//
// This file should never be loaded. Its mere presence in the module directory
// must trigger an error from validateFileList() before any CUE load occurs.
// Module authors must move environment-specific values files outside the module
// directory and reference them via --values.
package roguemodule

values: {
	image:    "forge:1.20"
	replicas: 3
}
