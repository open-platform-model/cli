package validatediag

// Invalid values for testing ValidateConfig diagnostics.
// Contains three deliberate violations:
//   - "test": extra top-level field (field not allowed)
//   - "invalidField": extra top-level field (field not allowed)
//   - settings.timeout: type conflict (string vs int → conflicting values)
values: {
	port: 9090
	name: "hello"
	settings: {
		debug:   false
		timeout: "slow" // wrong type: string vs int
	}
	test:         "bad"
	invalidField: "bad"
}
