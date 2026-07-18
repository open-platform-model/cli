package demo

kind: "ModuleInstance"

metadata: {
	name:      "demo-instance"
	namespace: "default"
}

#module: {
	kind: "Module"
	metadata: {
		name:       "demo"
		modulePath: "test.example.com/demo"
		version:    "0.1.0"
	}
	#config: {
		media?: [Name=string]: {
			type: "pvc" | *"emptyDir"
		}
	}
}
