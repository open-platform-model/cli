package engine

import (
	"sort"

	pmatch "github.com/opmodel/cli/internal/match"
)

type MatchResult = pmatch.MatchResult
type MatchPlan = pmatch.MatchPlan
type MatchedPair = pmatch.MatchedPair
type NonMatchedPair = pmatch.NonMatchedPair

func sortMatchedPairs(pairs []MatchedPair) {
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].ComponentName != pairs[j].ComponentName {
			return pairs[i].ComponentName < pairs[j].ComponentName
		}
		return pairs[i].TransformerFQN < pairs[j].TransformerFQN
	})
}
