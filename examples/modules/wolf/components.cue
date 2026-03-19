// Components defines the Wolf game streaming server workload.
//
// A single StatefulSet (`wolf`) is deployed containing:
//
//   initContainer: config-seed
//     Writes /etc/wolf/cfg/config.toml on first start (skips if already present).
//     Wolf reads this file at startup and writes paired_clients back into it,
//     so it must be on the persistent config PVC, not a read-only ConfigMap.
//
//   sidecar: dind
//     Docker-in-Docker daemon providing the Docker API Wolf uses to spawn
//     per-session app containers. Shares the config PVC and XDG socket volume
//     so that Wolf can pass correct bind-mount paths to those containers.
//
//   main: wolf
//     The Wolf streaming server. Talks to DinD via a shared Unix socket.
//     Mounts /dev and /run/udev from the host for GPU and virtual input access.
//
// Volume layout:
//
//   wolf-config   PVC/hostPath/NFS   → /etc/wolf       (wolf + dind)
//   docker-data   PVC/emptyDir       → /var/lib/docker  (dind only)
//   docker-socket emptyDir           → /run/dind        (wolf + dind)  ← shared socket
//   wolf-api      emptyDir           → /run/wolf        (wolf only)    ← wolf.sock
//   xdg-sockets   emptyDir           → /tmp/wolf-sockets (wolf + dind) ← PulseAudio/Wayland
//   dev           hostPath /dev      → /dev             (wolf + dind)
//   udev          hostPath /run/udev → /run/udev        (wolf only)
//   nvidia-driver hostPath (nvidia)  → /usr/nvidia      (wolf only, nvidia only)
//
// Security note:
//   DinD requires privileged: true at the pod level.
//   Wolf requires capabilities: [NET_RAW, MKNOD, NET_ADMIN, SYS_ADMIN, SYS_NICE]
//   and device cgroup rule "c 13:* rmw" for virtual input device (uinput/uhid) support.
//   These must be configured via the K8s provider / ModuleRelease annotations or
//   a cluster-level mutating webhook — the current OPM SecurityContextSchema does
//   not yet model container-level privileged or device cgroup rules.
package wolf

import (
	"list"

	resources_workload "opmodel.dev/resources/workload@v1"
	resources_storage  "opmodel.dev/resources/storage@v1"
	traits_workload    "opmodel.dev/traits/workload@v1"
	traits_network     "opmodel.dev/traits/network@v1"
)

// #components contains component definitions resolved at build time.
#components: {
	wolf: {
		resources_workload.#Container
		resources_storage.#Volumes
		traits_workload.#InitContainers
		traits_workload.#SidecarContainers
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_workload.#GracefulShutdown
		traits_network.#Expose

		// StatefulSet — Wolf stores paired client state on the config PVC
		metadata: labels: "core.opmodel.dev/workload-type": "stateful"

		spec: {
			scaling: count: 1

			restartPolicy: "Always"

			// Recreate strategy — only one Wolf instance can use the GPU + uinput
			// devices safely at a time
			updateStrategy: type: "Recreate"

			// Allow in-flight streams and DinD containers to wind down gracefully
			gracefulShutdown: terminationGracePeriodSeconds: 60

			// ── Init Container: config-seed ────────────────────────────────────
			// Seeds /etc/wolf/cfg/config.toml on first start only.
			// Wolf writes paired_clients back to this file, so we must not
			// overwrite it on subsequent starts.
			initContainers: [{
				name: "config-seed"
				image: {
					repository: "busybox"
					tag:        "latest"
					digest:     ""
				}
				command: ["sh", "-c"]
				args: ["""
					CONFIG_DIR="/etc/wolf/cfg"
					CONFIG_FILE="$CONFIG_DIR/config.toml"

					if [ -f "$CONFIG_FILE" ]; then
					  echo "config-seed: $CONFIG_FILE already exists, skipping."
					  exit 0
					fi

					mkdir -p "$CONFIG_DIR"
					cat > "$CONFIG_FILE" << TOML
					hostname = "\(#config.wolf.hostname)"
					support_hevc = \(#config.wolf.supportHevc)
					config_version = 2

					paired_clients = []
					profiles = []
					gstreamer = {}
					TOML

					echo "config-seed: wrote $CONFIG_FILE"
					"""]

				// Reference volumes from spec.volumes so the source type is included,
				// satisfying the #VolumeMountSchema matchN(1, [...]) constraint.
				volumeMounts: {
					"wolf-config": volumes["wolf-config"] & {
						mountPath: "/etc/wolf"
					}
				}
			}]

			// ── Sidecar: dind ─────────────────────────────────────────────────
			// Docker-in-Docker daemon providing the Docker API for Wolf to spawn
			// per-session app containers. Listens on a shared Unix socket so Wolf
			// does not need host Docker access.
			//
			// Required env:
			//   DOCKER_TLS_CERTDIR=""  — disables TLS so Wolf can connect without certs
			//
			// Required args passed to dockerd:
			//   --host unix:///run/dind/docker.sock — write socket to shared emptyDir
			//   --tls=false                          — no TLS on the socket
			//
			// Security: DinD requires privileged: true. Apply at the K8s provider level.
			//
			// The config PVC is mounted at /etc/wolf in DinD so that when Wolf tells
			// DinD to create an app container with "-v /etc/wolf/profile_data/...:/home/retro",
			// DinD can resolve that bind mount path from its own filesystem.
			//
			// The XDG socket emptyDir is mounted at /tmp/wolf-sockets in DinD so
			// PulseAudio containers spawned by DinD can bind-mount the same path.
			let _dindSidecar = [{
				name:  "dind"
				image: #config.dind.image
				args: [
					"--host", "unix:///run/dind/docker.sock",
					"--tls=false",
				]
				env: {
					// Empty string disables TLS certificate generation at startup
					DOCKER_TLS_CERTDIR: {
						name:  "DOCKER_TLS_CERTDIR"
						value: ""
					}
				}
				volumeMounts: {
					"wolf-config": volumes["wolf-config"] & {
						mountPath: "/etc/wolf"
					}
					"docker-data": volumes["docker-data"] & {
						mountPath: "/var/lib/docker"
					}
					"docker-socket": volumes["docker-socket"] & {
						mountPath: "/run/dind"
					}
					"xdg-sockets": volumes["xdg-sockets"] & {
						mountPath: "/tmp/wolf-sockets"
					}
					dev: volumes.dev & {
						mountPath: "/dev"
					}
				}

				if #config.dind.resources != _|_ {
					resources: #config.dind.resources
				}
			}]

			// ── Optional API proxy sidecar: nginx ──────────────────────────────
			// nginx reverse proxy to expose the Wolf REST API (Unix socket) over TCP.
			// Enabled only when #config.api is defined and api.enabled is true.
			let _apiProxySidecar = [if #config.api != _|_ if #config.api.enabled {
				{
					name:  "api-proxy"
					image: #config.api.image
					command: ["sh", "-c"]
					args: ["""
						cat > /etc/nginx/conf.d/wolf.conf << 'EOF'
						server {
						    listen \(#config.api.port);
						    location / {
						        proxy_pass http://unix:/run/wolf/wolf.sock;
						        proxy_http_version 1.0;
						        proxy_set_header Host $host;
						        proxy_set_header X-Real-IP $remote_addr;
						    }
						}
						EOF
						nginx -g 'daemon off;'
						"""]
					ports: {
						api: {
							targetPort: #config.api.port
							protocol:   "TCP"
						}
					}
					volumeMounts: {
						"wolf-api": volumes["wolf-api"] & {
							mountPath: "/run/wolf"
							readOnly:  true
						}
					}
					if #config.api.resources != _|_ {
						resources: #config.api.resources
					}
				}
			}]

			sidecarContainers: list.Concat([_dindSidecar, _apiProxySidecar])

			// ── Main container: wolf ───────────────────────────────────────────
			container: {
				name:  "wolf"
				image: #config.image

				ports: {
					// TCP ports — Moonlight pairing and RTSP stream setup
					https: {
						targetPort: #config.networking.httpsPort
						protocol:   "TCP"
					}
					http: {
						targetPort: #config.networking.httpPort
						protocol:   "TCP"
					}
					rtsp: {
						targetPort: #config.networking.rtspPort
						protocol:   "TCP"
					}
					// UDP ports — control, video, audio streams
					control: {
						targetPort: #config.networking.controlPort
						protocol:   "UDP"
					}
					video: {
						targetPort: #config.networking.videoPort
						protocol:   "UDP"
					}
					audio: {
						targetPort: #config.networking.audioPort
						protocol:   "UDP"
					}
					if #config.api != _|_ if #config.api.enabled {
						"api-proxy": {
							targetPort: #config.api.port
							protocol:   "TCP"
						}
					}
				}

				env: {
					// ── Wolf daemon settings ───────────────────────────────────
					WOLF_LOG_LEVEL: {
						name:  "WOLF_LOG_LEVEL"
						value: #config.wolf.logLevel
					}
					WOLF_CFG_FILE: {
						name:  "WOLF_CFG_FILE"
						value: "/etc/wolf/cfg/config.toml"
					}
					WOLF_STOP_CONTAINER_ON_EXIT: {
						name:  "WOLF_STOP_CONTAINER_ON_EXIT"
						value: "\(#config.wolf.stopContainerOnExit)"
					}

					// ── GPU ────────────────────────────────────────────────────
					WOLF_RENDER_NODE: {
						name:  "WOLF_RENDER_NODE"
						value: #config.gpu.renderNode
					}

					// ── Docker socket ──────────────────────────────────────────
					// Point Wolf at the DinD socket on the shared emptyDir volume
					WOLF_DOCKER_SOCKET: {
						name:  "WOLF_DOCKER_SOCKET"
						value: "/run/dind/docker.sock"
					}

					// ── Paths ──────────────────────────────────────────────────
					// HOST_APPS_STATE_FOLDER tells Wolf the base path under which
					// it will store per-user per-app persistent state. Wolf passes
					// this path to DinD as a bind mount source for app containers,
					// so it must match the mountPath of the wolf-config volume.
					HOST_APPS_STATE_FOLDER: {
						name:  "HOST_APPS_STATE_FOLDER"
						value: "/etc/wolf"
					}
					// XDG_RUNTIME_DIR is used by Wolf for PulseAudio and Wayland
					// compositor Unix sockets shared with app containers.
					XDG_RUNTIME_DIR: {
						name:  "XDG_RUNTIME_DIR"
						value: "/tmp/wolf-sockets"
					}
					// Wolf REST API Unix socket path (used by Wolf UI internally)
					WOLF_SOCKET_PATH: {
						name:  "WOLF_SOCKET_PATH"
						value: "/run/wolf/wolf.sock"
					}

					// ── Port overrides (only emit when non-default) ────────────
					if #config.networking.httpsPort != 47984 {
						WOLF_HTTPS_PORT: {
							name:  "WOLF_HTTPS_PORT"
							value: "\(#config.networking.httpsPort)"
						}
					}
					if #config.networking.httpPort != 47989 {
						WOLF_HTTP_PORT: {
							name:  "WOLF_HTTP_PORT"
							value: "\(#config.networking.httpPort)"
						}
					}
					if #config.networking.controlPort != 47999 {
						WOLF_CONTROL_PORT: {
							name:  "WOLF_CONTROL_PORT"
							value: "\(#config.networking.controlPort)"
						}
					}
					if #config.networking.rtspPort != 48010 {
						WOLF_RTSP_SETUP_PORT: {
							name:  "WOLF_RTSP_SETUP_PORT"
							value: "\(#config.networking.rtspPort)"
						}
					}
					if #config.networking.videoPort != 48100 {
						WOLF_VIDEO_PING_PORT: {
							name:  "WOLF_VIDEO_PING_PORT"
							value: "\(#config.networking.videoPort)"
						}
					}
					if #config.networking.audioPort != 48200 {
						WOLF_AUDIO_PING_PORT: {
							name:  "WOLF_AUDIO_PING_PORT"
							value: "\(#config.networking.audioPort)"
						}
					}

					// ── GStreamer debug (optional) ──────────────────────────────
					if #config.wolf.gstDebug != _|_ {
						GST_DEBUG: {
							name:  "GST_DEBUG"
							value: #config.wolf.gstDebug
						}
					}

					// ── NVIDIA-specific ────────────────────────────────────────
					if #config.gpu.type == "nvidia" if #config.gpu.nvidia != _|_ {
						NVIDIA_DRIVER_VOLUME_NAME: {
							name:  "NVIDIA_DRIVER_VOLUME_NAME"
							value: "nvidia-driver-vol"
						}
						NVIDIA_DRIVER_CAPABILITIES: {
							name:  "NVIDIA_DRIVER_CAPABILITIES"
							value: "all"
						}
						NVIDIA_VISIBLE_DEVICES: {
							name:  "NVIDIA_VISIBLE_DEVICES"
							value: "all"
						}
					}
				}

				// Reference volumes from spec.volumes so the source type is included,
				// satisfying the #VolumeMountSchema matchN(1, [...]) constraint.
				volumeMounts: {
					// Wolf configuration, paired clients, and app state
					"wolf-config": volumes["wolf-config"] & {
						mountPath: "/etc/wolf"
					}
					// Shared Docker socket — talks to the DinD daemon
					"docker-socket": volumes["docker-socket"] & {
						mountPath: "/run/dind"
					}
					// Wolf REST API Unix socket
					"wolf-api": volumes["wolf-api"] & {
						mountPath: "/run/wolf"
					}
					// PulseAudio and Wayland compositor sockets
					"xdg-sockets": volumes["xdg-sockets"] & {
						mountPath: "/tmp/wolf-sockets"
					}
					// Host /dev — GPU, uinput, uhid device access
					dev: volumes.dev & {
						mountPath: "/dev"
					}
					// Host /run/udev — udev event socket for virtual device detection
					udev: volumes.udev & {
						mountPath: "/run/udev"
					}
					// NVIDIA driver libraries (nvidia only)
					if #config.gpu.type == "nvidia" if #config.gpu.nvidia != _|_ {
						"nvidia-driver": volumes["nvidia-driver"] & {
							mountPath: "/usr/nvidia"
						}
					}
				}

				if #config.resources != _|_ {
					resources: #config.resources
				}
			}

			// ── Network exposure ───────────────────────────────────────────────
			// Expose all streaming ports via a Kubernetes Service.
			// For host-network pods (hostNetwork: true), this Service is still
			// useful as a stable DNS endpoint for management/observability — the
			// actual streaming traffic goes directly over the node IP.
			expose: {
				ports: {
					https: {
						targetPort:  #config.networking.httpsPort
						protocol:    "TCP"
						exposedPort: #config.networking.httpsPort
					}
					http: {
						targetPort:  #config.networking.httpPort
						protocol:    "TCP"
						exposedPort: #config.networking.httpPort
					}
					rtsp: {
						targetPort:  #config.networking.rtspPort
						protocol:    "TCP"
						exposedPort: #config.networking.rtspPort
					}
					control: {
						targetPort:  #config.networking.controlPort
						protocol:    "UDP"
						exposedPort: #config.networking.controlPort
					}
					video: {
						targetPort:  #config.networking.videoPort
						protocol:    "UDP"
						exposedPort: #config.networking.videoPort
					}
					audio: {
						targetPort:  #config.networking.audioPort
						protocol:    "UDP"
						exposedPort: #config.networking.audioPort
					}
					if #config.api != _|_ if #config.api.enabled {
						"api-proxy": {
							targetPort:  #config.api.port
							protocol:    "TCP"
							exposedPort: #config.api.port
						}
					}
				}
				type: #config.networking.serviceType
			}

			// ── Volumes ────────────────────────────────────────────────────────
			volumes: {
				// Wolf configuration, paired clients, and per-user app state
				"wolf-config": {
					name: "wolf-config"
					if #config.storage.config.type == "pvc" {
						persistentClaim: {
							size: #config.storage.config.size
							if #config.storage.config.storageClass != _|_ {
								storageClass: #config.storage.config.storageClass
							}
						}
					}
					if #config.storage.config.type == "hostPath" {
						hostPath: {
							path: #config.storage.config.path
							type: #config.storage.config.hostPathType
						}
					}
					if #config.storage.config.type == "nfs" {
						nfs: {
							server: #config.storage.config.nfsServer
							path:   #config.storage.config.nfsPath
						}
					}
				}

				// Docker image layers for the DinD daemon
				"docker-data": {
					name: "docker-data"
					if #config.dind.storage.type == "pvc" {
						persistentClaim: {
							size: #config.dind.storage.size
							if #config.dind.storage.storageClass != _|_ {
								storageClass: #config.dind.storage.storageClass
							}
						}
					}
					if #config.dind.storage.type == "emptyDir" {
						emptyDir: {}
					}
				}

				// Shared Unix socket between DinD and Wolf
				// DinD writes: /run/dind/docker.sock
				// Wolf reads:  /run/dind/docker.sock (via WOLF_DOCKER_SOCKET)
				"docker-socket": {
					name:     "docker-socket"
					emptyDir: {}
				}

				// Wolf REST API Unix socket (wolf.sock)
				// Consumed by Wolf UI and optional nginx API proxy sidecar
				"wolf-api": {
					name:     "wolf-api"
					emptyDir: {}
				}

				// PulseAudio and Wayland compositor Unix sockets (XDG_RUNTIME_DIR)
				// Wolf creates sockets here; app containers spawned by DinD bind-mount
				// this same path to access the audio and display streams.
				"xdg-sockets": {
					name:     "xdg-sockets"
					emptyDir: {}
				}

				// Host /dev — grants Wolf and DinD access to:
				//   /dev/dri/*     — GPU render nodes (Intel/AMD/NVIDIA)
				//   /dev/uinput    — virtual joypad creation (requires udev rules)
				//   /dev/uhid      — DualSense emulation
				//   /dev/nvidia*   — NVIDIA device nodes (nvidia only)
				dev: {
					name: "dev"
					hostPath: {
						path: "/dev"
						type: "Directory"
					}
				}

				// Host /run/udev — udev socket and database for virtual device hotplug
				// Wolf uses fake-udev to send events into app container namespaces
				udev: {
					name: "udev"
					hostPath: {
						path: "/run/udev"
						type: "Directory"
					}
				}

				// NVIDIA driver libraries (mounted at /usr/nvidia inside Wolf)
				// Created from a gow/nvidia-driver Docker image — see README.
				if #config.gpu.type == "nvidia" if #config.gpu.nvidia != _|_ {
					"nvidia-driver": {
						name: "nvidia-driver"
						hostPath: {
							path: #config.gpu.nvidia.driverPath
							type: #config.gpu.nvidia.hostPathType
						}
					}
				}
			}
		}
	}
}
