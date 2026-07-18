// Package schema defines the embedded CUE schemas for OPM CLI configuration.
// This file validates ~/.opm/platform.cue — the local default platform —
// and any file passed via --platform.
package schema

// #PlatformFile is the schema for the local platform file. It is a data-only
// projection of the Platform CR spec (opm-operator api/v1alpha1 PlatformSpec,
// itself projecting core #Platform): the same document that, wrapped in
// apiVersion/kind/metadata, is the cluster singleton Platform. The file MUST
// NOT contain CUE imports; the CLI rejects import-bearing platform files
// (enhancement 0006 D39).
#PlatformFile: {
	// name is the platform name (metadata.name of the CR form). The cluster
	// singleton is conventionally named "cluster".
	name!: string & =~"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"

	// type is the informational platform discriminator (core #Platform.type).
	// It does not affect matching.
	type!: string & !=""

	// registry is the set of catalog subscriptions keyed by catalog CUE
	// module path (e.g. "opmodel.dev/catalogs/opm").
	registry?: [string]: #Subscription
}

// #Subscription is a single catalog subscription, projecting core
// #Subscription / synth.SubscriptionSpec.
#Subscription: {
	// enable toggles the subscription. Omitted defers to the schema
	// default (true).
	enable?: bool

	// filter optionally constrains the subscribed versions.
	filter?: #SubscriptionFilter
}

// #SubscriptionFilter mirrors core #SubscriptionFilter.
#SubscriptionFilter: {
	// range is a SemVer constraint expression (e.g. ">=1.0.0-0 <2.0.0-0").
	range?: string

	// allow force-includes specific versions regardless of range.
	allow?: [...string]

	// deny force-excludes specific versions from the survivor set.
	deny?: [...string]
}
