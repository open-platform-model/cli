bundles: cli: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/cli/"]}
	version: {from: "tag", prefix: "v"}
	pins: {command: ["go", "run", "./hack/docskit-dump", "pins"], projects: ["library", "core", "opm-operator"]}
	sources: [{
		kind:        "cobra"
		command:     ["go", "run", "./hack/docskit-dump"]
		section:     "reference/cli/"
		title:       "CLI Reference"
		description: "Every opm command and flag, generated from the CLI's cobra commands."
		weight:      2
	}, {
		// The authored pages ship in the same bundle (docs-kit DESIGN decision
		// 20). The exclude keeps cmdref's committed pages out while the site
		// still reads the cli from git; both go at G2-switch.
		kind: "markdown", dir: "docs/site", exclude: ["reference/cli/"]
	}]
}
