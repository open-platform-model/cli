package nometadata

// Module without literal metadata name (uses a let binding)
_baseName: "computed"
metadata: {
	modulePath: "example.com/modules"
	name:       _baseName + "-module"
	version:    "1.0.0"
	fqn:        "example.com/modules/" + (_baseName + "-module") + ":1.0.0"
}

#config: {
	image: {
		repository: string | *"nginx"
		tag:        string | *"latest"
		digest:     string | *""
	}
}

#components: {}
