// rogue_module/values.cue — the legitimate default values file.
// Its presence alongside values_forge.cue is what makes this module "rogue":
// having both values.cue AND another values*.cue in the same directory is invalid.
package roguemodule

values: {
	image:    "nginx:latest"
	replicas: 1
}
