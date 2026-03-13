package validatediag

// Minimal closed config schema for testing ValidateConfig diagnostics.
// Deliberately simple — no external deps required.
#config: {
	port: int | *8080
	name: string | *"myapp"
	settings: {
		debug:   bool | *false
		timeout: int
	}
}
