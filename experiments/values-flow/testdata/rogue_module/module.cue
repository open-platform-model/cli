// rogue_module/module.cue — pattern C: rogue values file present.
//
// This module contains values_forge.cue alongside values.cue.
// Approach C validation must detect values_forge.cue and return an error
// before any load.Instances call occurs.
package roguemodule

apiVersion: "opmodel.dev/core/v0"
kind:       "Module"

metadata: {
	apiVersion: "example.com/rogue-module@v0"
	name:       "rogue-module"
	version:    "1.0.0"
}

#config: {
	image:    string
	replicas: int & >=1
}

#components: {
	web: {
		metadata: name: "web"
		spec: {
			image:    #config.image
			replicas: #config.replicas
		}
	}
}
