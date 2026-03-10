package render

func WriteTransformerMatchesForTest(result *Result) {
	writeTransformerMatches(result)
}

func WriteVerboseMatchLogForTest(result *Result) {
	writeVerboseMatchLog(result)
}

func FormatFQNListForTest(fqns []string) string {
	return formatFQNList(fqns)
}
