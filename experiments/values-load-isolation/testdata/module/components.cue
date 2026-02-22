// components.cue — a second module package file.
// Proves that filtering out values*.cue does not accidentally drop
// other legitimate package files like this one.
package main

#components: {
	server: {
		metadata: {
			name: "server"
		}
		spec: {
			serverType: values.serverType
			port:       values.port
			maxPlayers: values.maxPlayers
		}
	}
	proxy: {
		metadata: {
			name: "proxy"
		}
		spec: {
			// Proxy always listens on a fixed port, independent of values
			port:   25577
			target: "server:\(values.port)"
		}
	}
}
