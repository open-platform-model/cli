package nometadata

// Module without literal metadata name (uses a let binding)
_baseName: "computed"
metadata: {
	name:    _baseName + "-module"
	version: "1.0.0"
}

#config: {
	image: string
}

#components: {}
