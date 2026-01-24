package hello_world

// Module metadata - using let to avoid cycles
let _name = "hello-world"
let _version = "1.0.0"

metadata: {
	name:    _name
	version: _version
}

values: {
	replicas: int | *1
	image:    string | *"nginx:1.25"
	port:     int | *80
}

manifests: [
	{
		apiVersion: "v1"
		kind:       "ConfigMap"
		metadata: {
			name: _name + "-config"
		}
		data: {
			"index.html": """
				<!DOCTYPE html>
				<html>
				<head><title>Hello World</title></head>
				<body><h1>Hello from OPM!</h1></body>
				</html>
				"""
		}
	},
	{
		apiVersion: "apps/v1"
		kind:       "Deployment"
		metadata: {
			name: _name
		}
		spec: {
			replicas: values.replicas
			selector: matchLabels: app: _name
			template: {
				metadata: labels: app: _name
				spec: {
					containers: [{
						name:  "nginx"
						image: values.image
						ports: [{
							containerPort: values.port
						}]
						volumeMounts: [{
							name:      "html"
							mountPath: "/usr/share/nginx/html"
						}]
					}]
					volumes: [{
						name: "html"
						configMap: name: _name + "-config"
					}]
				}
			}
		}
	},
	{
		apiVersion: "v1"
		kind:       "Service"
		metadata: {
			name: _name
		}
		spec: {
			selector: app: _name
			ports: [{
				port:       values.port
				targetPort: values.port
			}]
		}
	},
]
