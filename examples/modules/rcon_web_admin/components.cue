// Components defines the RCON Web Admin workload.
// Single stateless component with dual-port networking (HTTP + WebSocket) and optional HTTPRoute.
package rcon_web_admin

import (
	"strings"

	resources_workload "opmodel.dev/resources/workload@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
	traits_security "opmodel.dev/traits/security@v1"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// Admin - Stateless RCON Web Console
	/////////////////////////////////////////////////////////////////

	admin: {
		resources_workload.#Container
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_network.#Expose
		if #config.httpRoute != _|_ {
			traits_network.#HttpRoute
		}
		traits_security.#SecurityContext

		metadata: labels: "core.opmodel.dev/workload-type": "stateless"

		spec: {
			// Single replica - lightweight web UI
			scaling: count: 1

			restartPolicy: "Always"

			// === Security Context ===
			if #config.securityContext != _|_ {
				securityContext: #config.securityContext
			}
			if #config.securityContext == _|_ {
				securityContext: {
					runAsNonRoot:             true
					runAsUser:                1000
					runAsGroup:               3000
					readOnlyRootFilesystem:   true
					allowPrivilegeEscalation: false
					capabilities: drop: ["ALL"]
				}
			}

			// === Main Container: RCON Web Admin ===
			container: {
				name:  "rcon-web-admin"
				image: #config.admin.image

				ports: {
					http: {
						targetPort: #config.httpPort
						protocol:   "TCP"
					}
					ws: {
						targetPort: #config.wsPort
						protocol:   "TCP"
					}
				}

				env: {
					// Admin flag (itzg/rcon expects uppercase TRUE/FALSE)
					RWA_ADMIN: {
						name: "RWA_ADMIN"
						if #config.admin.isAdmin {
							value: "TRUE"
						}
						if !#config.admin.isAdmin {
							value: "FALSE"
						}
					}

					// Login credentials
					RWA_USERNAME: {
						name:  "RWA_USERNAME"
						value: #config.admin.username
					}
				RWA_PASSWORD: {
					name: "RWA_PASSWORD"
					from: #config.admin.password
				}

					// Game type
					RWA_GAME: {
						name:  "RWA_GAME"
						value: #config.admin.game
					}

					// Optional: Server display name
					if #config.admin.serverName != _|_ {
						RWA_SERVER_NAME: {
							name:  "RWA_SERVER_NAME"
							value: #config.admin.serverName
						}
					}

					// RCON connection details
					RWA_RCON_HOST: {
						name:  "RWA_RCON_HOST"
						value: #config.admin.rconHost
					}
					RWA_RCON_PORT: {
						name:  "RWA_RCON_PORT"
						value: "\(#config.admin.rconPort)"
					}
				RWA_RCON_PASSWORD: {
					name: "RWA_RCON_PASSWORD"
					from: #config.admin.rconPassword
				}

					// Optional: WebSocket-based RCON
					if #config.admin.websocketRcon != _|_ {
						RWA_WEBSOCKET: {
							name: "RWA_WEBSOCKET"
							if #config.admin.websocketRcon {
								value: "TRUE"
							}
							if !#config.admin.websocketRcon {
								value: "FALSE"
							}
						}
					}

					// Optional: Restrict commands (comma-separated)
					if #config.admin.restrictCommands != _|_ {
						RWA_RESTRICT_COMMANDS: {
							name:  "RWA_RESTRICT_COMMANDS"
							value: strings.Join(#config.admin.restrictCommands, ",")
						}
					}

					// Optional: Restrict widgets (comma-separated)
					if #config.admin.restrictWidgets != _|_ {
						RWA_RESTRICT_WIDGETS: {
							name:  "RWA_RESTRICT_WIDGETS"
							value: strings.Join(#config.admin.restrictWidgets, ",")
						}
					}

					// Optional: Read-only widget options
					if #config.admin.immutableWidgetOptions != _|_ {
						RWA_READ_ONLY_WIDGET_OPTIONS: {
							name: "RWA_READ_ONLY_WIDGET_OPTIONS"
							if #config.admin.immutableWidgetOptions {
								value: "TRUE"
							}
							if !#config.admin.immutableWidgetOptions {
								value: "FALSE"
							}
						}
					}
				}

				if #config.resources != _|_ {
					resources: #config.resources
				}
			}

			// === Network Exposure ===
			expose: {
				ports: {
					http: container.ports.http & {
						exposedPort: #config.httpPort
					}
					ws: container.ports.ws & {
						exposedPort: #config.wsPort
					}
				}
				type: #config.serviceType
			}

			// === HTTPRoute (optional, for Gateway API ingress) ===
			if #config.httpRoute != _|_ {
				httpRoute: {
					if #config.httpRoute.hostnames != _|_ {
						hostnames: #config.httpRoute.hostnames
					}
					rules: [{
						matches: [{
							path: {
								type:  "Prefix"
								value: "/"
							}
						}]
						backendPort: #config.httpPort
					}]
					if #config.httpRoute.gatewayRef != _|_ {
						parentRefs: [{
							name: #config.httpRoute.gatewayRef.name
							if #config.httpRoute.gatewayRef.namespace != _|_ {
								namespace: #config.httpRoute.gatewayRef.namespace
							}
						}]
					}
				}
			}
		}
	}
}
