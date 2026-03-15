// MetalLB — bare metal load-balancer for Kubernetes.
// Deploys the MetalLB controller (Deployment), speaker (DaemonSet), all CRD definitions,
// and the required ClusterRoles + ClusterRoleBindings.
//
// https://metallb.io  |  https://github.com/metallb/metallb
package metallb

import (
	m       "opmodel.dev/core/module@v1"
	schemas "opmodel.dev/schemas@v1"
)

m.#Module

metadata: {
	modulePath:       "opmodel.dev/modules"
	name:             "metallb"
	version:          "0.1.0"
	description:      "MetalLB bare metal load-balancer for Kubernetes — deploys controller, speaker, CRDs, and RBAC"
	defaultNamespace: "metallb-system"
	labels: {
		"app.kubernetes.io/component": "load-balancer"
	}
}

#config: {
	// Image configuration — tag is shared across all MetalLB components (controller, speaker).
	image: {
		// MetalLB release tag (e.g., "v0.15.3"). See https://github.com/metallb/metallb/releases.
		tag: string | *"v0.15.3"
		// Image pull policy applied to both controller and speaker.
		pullPolicy: "Always" | *"IfNotPresent" | "Never"
	}

	// Controller configuration — handles IP address assignment for LoadBalancer Services.
	controller: {
		// Structured log level for the controller.
		logLevel: "debug" | *"info" | "warn" | "error"
		// Number of controller replicas.
		replicas: int & >=1 | *1
		// Resource requests and limits (optional — omit to use cluster defaults).
		resources?: schemas.#ResourceRequirementsSchema
		// Reference to the webhook TLS Secret.
		// OPM does not create this Secret — it must be pre-created as an empty secret
		// (Helm creates it as an empty stub; the controller then writes the cert into it):
		//   kubectl create secret generic metallb-webhook-cert -n metallb-system \
		//     --from-literal=ca.crt="" --from-literal=tls.crt="" --from-literal=tls.key=""
		webhookCertSecret: schemas.#Secret | *{
			$opm:         "secret"
			$secretName:  "metallb-webhook-cert"
			$dataKey:     "tls.crt"
			$description: "MetalLB webhook TLS certificate (written by the controller at startup)"
			secretName:   "metallb-webhook-cert"
			remoteKey:    "tls.crt"
		}
	}

	// Speaker configuration — runs on every node and announces LoadBalancer IPs via L2/BGP.
	speaker: {
		// Structured log level for the speaker.
		logLevel: "debug" | *"info" | "warn" | "error"
		// Resource requests and limits (optional — omit to use cluster defaults).
		resources?: schemas.#ResourceRequirementsSchema
		// Reference to the pre-created memberlist Secret containing the gossip encryption key.
		// OPM does not create this Secret — the operator must pre-create it:
		//   kubectl create secret generic metallb-memberlist -n metallb-system \
		//     --from-literal=secretkey=$(head -c 128 /dev/urandom | base64 | tr -d '\n')
		memberlistSecret: schemas.#Secret | *{
			$opm:         "secret"
			$secretName:  "metallb-memberlist"
			$dataKey:     "secretkey"
			$description: "MetalLB memberlist gossip encryption key"
			secretName:   "metallb-memberlist"
			remoteKey:    "secretkey"
		}
	}
}

// debugValues exercises the full #config surface for local `cue vet` / `cue eval`.
debugValues: {
	image: {
		tag:        "v0.15.3"
		pullPolicy: "IfNotPresent"
	}
	controller: {
		logLevel: "info"
		replicas: 1
		webhookCertSecret: {
			$opm:        "secret"
			$secretName: "metallb-webhook-cert"
			$dataKey:    "tls.crt"
			secretName:  "metallb-webhook-cert"
			remoteKey:   "tls.crt"
		}
		resources: {
			requests: {
				cpu:    "100m"
				memory: "64Mi"
			}
			limits: {
				cpu:    "200m"
				memory: "128Mi"
			}
		}
	}
	speaker: {
		logLevel: "info"
		memberlistSecret: {
			$opm:         "secret"
			$secretName:  "metallb-memberlist"
			$dataKey:     "secretkey"
			secretName:   "metallb-memberlist"
			remoteKey:    "secretkey"
		}
		resources: {
			requests: {
				cpu:    "100m"
				memory: "64Mi"
			}
			limits: {
				cpu:    "200m"
				memory: "128Mi"
			}
		}
	}
}
