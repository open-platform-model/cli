package demo

metadata: {
	name:       "demo"
	modulePath: "test.example.com/demo@v0"
	version:    "0.1.0"
}

#config: {
	media?: [Name=string]: {
		type: "pvc" | *"emptyDir"
	}
}
