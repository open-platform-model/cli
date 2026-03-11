package demo

values: {
	test: "test" // Triggers: field not allowed
	media: {
		test: "test" // Triggers: conflicting values (string vs struct)
	}
}
