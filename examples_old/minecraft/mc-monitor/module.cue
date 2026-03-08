// Package main defines the mc-monitor module for Minecraft server metrics export.
// A stateless application using itzg/mc-monitor:
// - module.cue: metadata and config schema
// - components.cue: component definitions with monitor container
// - values.cue: default values
//
// Supports two export modes (exactly one must be chosen):
//   - Prometheus: HTTP /metrics endpoint for scraping
//   - OpenTelemetry: push metrics to an OTel Collector via gRPC
//
// Can monitor multiple Java and Bedrock servers from a single deployment.
// Config schema mirrors the itzg/mc-monitor CLI flag / env var surface area.
package main

import (
	"opmodel.dev/core@v1"
	schemas "opmodel.dev/schemas@v1"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	modulePath:       "example.com/modules"
	name:             "mc-monitor"
	version:          "0.1.0"
	description:      "Prometheus and OpenTelemetry metrics exporter for Minecraft server status"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// === Container Image ===
	// Container image for mc-monitor
	image: schemas.#Image & {
		repository: string | *"itzg/mc-monitor"
		tag:        string | *"0.16.1"
		digest:     string | *""
	}

	// === Server Targets ===

	// Java Edition servers to monitor.
	// Each entry maps to a host:port pair passed to --servers / EXPORT_SERVERS.
	javaServers!: [...{
		host: string
		port: _#portSchema | *25565
	}]

	// Bedrock Edition servers to monitor (optional).
	// Each entry maps to a host:port pair passed to --bedrock-servers / EXPORT_BEDROCK_SERVERS.
	bedrockServers?: [...{
		host: string
		port: _#portSchema | *19132
	}]

	// === Export Mode ===
	// Set exactly ONE of the following to select the metrics export backend.
	// The matchN(1, [...]) constraint enforces that only one is chosen.
	//
	// Example — Prometheus (HTTP scrape endpoint):
	//   prometheus: { port: 8080 }
	//
	// Example — OpenTelemetry (gRPC push):
	//   otel: { collectorEndpoint: "otel-collector.monitoring.svc:4317" }

	// PROMETHEUS — HTTP /metrics endpoint for Prometheus scraping
	prometheus?: {
		// HTTP port to serve /metrics on
		port: _#portSchema | *8080
	}

	// OTEL — push metrics to an OpenTelemetry Collector via gRPC
	otel?: {
		// gRPC endpoint of the OTel Collector
		collectorEndpoint: string | *"localhost:4317"

		// Timeout for OTel data export
		collectorTimeout: string | *"35s"

		// Collection interval
		interval: string | *"10s"
	}

	matchN(1, [
		{prometheus!: _},
		{otel!: _},
	])

	// === Shared Settings ===

	// Per-server check timeout
	timeout: string | *"1m0s"

	// === Networking ===
	// Service type for network exposure (only relevant for Prometheus mode)
	serviceType: *"ClusterIP" | "NodePort" | "LoadBalancer"

	// === Resource Limits (catalog-standard shape) ===
	resources?: schemas.#ResourceRequirementsSchema
}

_#portSchema: uint & >0 & <=65535
