package schemas

import (
	"crypto/sha256"
	"encoding/hex"
	"list"
	"strings"
)

/////////////////////////////////////////////////////////////////
//// Secret Contract Type
/////////////////////////////////////////////////////////////////

// #Secret is the contract type that module authors place on
// sensitive fields. It is a disjunction of fulfillment variants.
// Users provide values that resolve to one of these variants.
//
// The $opm discriminator enables auto-discovery via CUE comprehensions.
// The $secretName and $dataKey fields carry routing information:
//   $secretName -> K8s Secret resource name (grouping key)
//   $dataKey    -> data key within that K8s Secret
//
// These are set by the author in the schema declaration.
// Users never need to set them — CUE unification propagates them.
#Secret: #SecretLiteral | #SecretK8sRef | #SecretEsoRef

// #SecretLiteral: user provides the actual value.
// The transformer creates a K8s Secret with this data entry.
#SecretLiteral: {
	$opm:          "secret"
	$secretName!:  #NameType
	$dataKey!:     string
	$description?: string
	value!:        string
}

// #SecretK8sRef: points to a pre-existing K8s Secret in the cluster.
// OPM emits no resource — the Secret already exists.
// OPM only wires the secretKeyRef in env vars.
#SecretK8sRef: {
	$opm:          "secret"
	$secretName!:  #NameType
	$dataKey!:     string
	$description?: string
	secretName!:   string // pre-existing K8s Secret name
	remoteKey!:    string // key within that K8s Secret
}

// #SecretEsoRef: points to an external secret store via ESO.
// OPM emits an ExternalSecret CR that creates a K8s Secret at deploy time.
#SecretEsoRef: {
	$opm:          "secret"
	$secretName!:  #NameType
	$dataKey!:     string
	$description?: string
	externalPath:  string // path in the external secret store
	remoteKey:     string // key within the external secret
}

/////////////////////////////////////////////////////////////////
//// Config Schemas
/////////////////////////////////////////////////////////////////

// Secret specification for K8s Secret resources.
// data holds #Secret entries (not plain strings).
// name is auto-populated from the map key in resources/config/secret.cue.
#SecretSchema: {
	name!:     string
	type:      *"Opaque" | "kubernetes.io/service-account-token" | "kubernetes.io/dockercfg" | "kubernetes.io/dockerconfigjson" | "kubernetes.io/basic-auth" | "kubernetes.io/ssh-auth" | "kubernetes.io/tls" | "bootstrap.kubernetes.io/token"
	immutable: bool | *false
	data: [string]: #Secret
}

// ConfigMap specification.
// name is auto-populated from the map key in resources/config/configmap.cue.
#ConfigMapSchema: {
	name!:     string
	immutable: bool | *false
	data: [string]: string
}

/////////////////////////////////////////////////////////////////
//// Content Hash Helpers
////
//// These helpers use regular fields (not #-prefixed definitions)
//// for their inputs. Definition fields lose concrete values when
//// forwarded via `{#data: #data}` — only the constraint propagates.
//// Regular fields carry concrete values through unification chains.
/////////////////////////////////////////////////////////////////

// #ContentHash computes a deterministic 10-character hex hash of a string data map.
// Used by ConfigMapTransformer and as a building block for #SecretContentHash.
#ContentHash: {
	data: [string]: string

	// Step 1: Extract and sort keys for deterministic ordering
	let _keys = [for k, _ in data {k}]
	let _sorted = list.SortStrings(_keys)

	// Step 2: Concatenate sorted key=value pairs
	let _pairs = [for _, k in _sorted {"\(k)=\(data[k])"}]
	let _concat = strings.Join(_pairs, "\n")

	// Step 3: SHA256 and take first 5 bytes (10 hex characters)
	out: hex.Encode(sha256.Sum256(_concat)[:5])
}

// #SecretContentHash normalizes #Secret entries to strings, then delegates
// to #ContentHash. The normalization is variant-aware:
//   #SecretLiteral  -> key=<value>
//   #SecretK8sRef   -> key=k8sref:<secretName>:<remoteKey>
//   #SecretEsoRef   -> key=esoref:<externalPath>:<remoteKey>
#SecretContentHash: {
	data: [string]: #Secret

	// Normalize each #Secret entry to a string for hashing
	let _normalized = {
		for k, v in data {
			if (v & #SecretLiteral) != _|_ {
				"\(k)": v.value
			}
			if (v & #SecretK8sRef) != _|_ {
				"\(k)": "k8sref:\(v.secretName):\(v.remoteKey)"
			}
			if (v & #SecretEsoRef) != _|_ {
				"\(k)": "esoref:\(v.externalPath):\(v.remoteKey)"
			}
		}
	}

	out: (#ContentHash & {data: _normalized}).out
}

// #ImmutableName computes the K8s resource name for a ConfigMap.
// Appends a content-hash suffix when immutable.
//
// Note: `let _d = data` is required to capture concrete field values.
// CUE does not forward concrete entries through open patterns like
// `[string]: string` when passed via `{data: data}` in definitions.
#ImmutableName: {
	baseName: string
	data: [string]: string
	immutable: bool | *false

	let _d = data
	_hash: (#ContentHash & {data: _d}).out

	if immutable {
		out: "\(baseName)-\(_hash)"
	}
	if !immutable {
		out: baseName
	}
}

// #SecretImmutableName computes the K8s resource name for a Secret.
// Appends a content-hash suffix when immutable.
#SecretImmutableName: {
	baseName: string
	data: [string]: #Secret
	immutable: bool | *false

	let _d = data
	_hash: (#SecretContentHash & {data: _d}).out

	if immutable {
		out: "\(baseName)-\(_hash)"
	}
	if !immutable {
		out: baseName
	}
}

/////////////////////////////////////////////////////////////////
//// Secret Discovery Pipeline
/////////////////////////////////////////////////////////////////

// #DiscoverSecrets walks a resolved config (up to 3 levels deep)
// and collects all fields whose value is a #Secret.
//
// The detection checks for the presence of the $opm discriminator field:
//   v.$opm != _|_
// This succeeds only when $opm is already set on the value (concrete #Secret).
// Scalars, closed structs (e.g., #Image), and open structs without $opm
// are all correctly skipped.
//
// For recursion into nested structs, we check:
//   v.$opm == _|_   (not a secret itself)
//   (v & {...}) != _|_ (is a struct we can iterate into)
//
// The result is a flat map of all discovered secrets, keyed by
// their path (e.g., "dbUser", "database/password", "auth/tokens/api").
// The path keys are internal identifiers — grouping uses $secretName/$dataKey.
#DiscoverSecrets: {
	X=#in: {...}
	out: {
		// Level 1: direct fields
		for k1, v1 in X
		if (v1.$opm != _|_) {
			(k1): v1
		}

		// Level 2: one level of nesting
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm != _|_) {
				("\(k1)/\(k2)"): v2
			}
		}

		// Level 3: two levels of nesting
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm == _|_)
			if ((v2 & {...}) != _|_) {
				for k3, v3 in v2
				if (v3.$opm != _|_) {
					("\(k1)/\(k2)/\(k3)"): v3
				}
			}
		}
	}
}

// #GroupSecrets takes a flat map of discovered secrets and groups
// them by $secretName, keyed by $dataKey.
// The result structure mirrors the K8s Secret resource layout:
//   { "db-creds": { username: #Secret, password: #Secret }, ... }
#GroupSecrets: {
	X=#in: {...}
	out: {
		for _k, v in X {
			(v.$secretName): (v.$dataKey): v
		}
	}
}

// #AutoSecrets discovers all #Secret instances from a resolved config
// and groups them by $secretName/$dataKey in one step.
// Output is ready to use as spec.secrets data entries.
//
#AutoSecrets: {
	X=#in: {...}
	let _discovered = (#DiscoverSecrets & {#in: X}).out
	out: (#GroupSecrets & {#in: _discovered}).out
}
