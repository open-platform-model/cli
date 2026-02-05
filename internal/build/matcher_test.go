package build

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
)

func TestMatcher_Match(t *testing.T) {
	matcher := NewMatcher()

	tests := []struct {
		name          string
		components    []*LoadedComponent
		transformers  []*LoadedTransformer
		wantMatched   int
		wantUnmatched int
	}{
		{
			name:          "empty inputs",
			components:    nil,
			transformers:  nil,
			wantMatched:   0,
			wantUnmatched: 0,
		},
		{
			name: "no transformers - all unmatched",
			components: []*LoadedComponent{
				{Name: "comp1", Labels: map[string]string{}, Resources: make(map[string]cue.Value), Traits: make(map[string]cue.Value)},
			},
			transformers:  nil,
			wantMatched:   0,
			wantUnmatched: 1,
		},
		{
			name: "matching by required labels",
			components: []*LoadedComponent{
				{
					Name:      "webapp",
					Labels:    map[string]string{"workload-type": "stateless"},
					Resources: make(map[string]cue.Value),
					Traits:    make(map[string]cue.Value),
				},
			},
			transformers: []*LoadedTransformer{
				{
					Name:           "DeploymentTransformer",
					FQN:            "test#DeploymentTransformer",
					RequiredLabels: map[string]string{"workload-type": "stateless"},
				},
			},
			wantMatched:   1,
			wantUnmatched: 0,
		},
		{
			name: "not matching - missing required label",
			components: []*LoadedComponent{
				{
					Name:      "webapp",
					Labels:    map[string]string{},
					Resources: make(map[string]cue.Value),
					Traits:    make(map[string]cue.Value),
				},
			},
			transformers: []*LoadedTransformer{
				{
					Name:           "DeploymentTransformer",
					FQN:            "test#DeploymentTransformer",
					RequiredLabels: map[string]string{"workload-type": "stateless"},
				},
			},
			wantMatched:   0,
			wantUnmatched: 1,
		},
		{
			name: "not matching - wrong label value",
			components: []*LoadedComponent{
				{
					Name:      "webapp",
					Labels:    map[string]string{"workload-type": "stateful"},
					Resources: make(map[string]cue.Value),
					Traits:    make(map[string]cue.Value),
				},
			},
			transformers: []*LoadedTransformer{
				{
					Name:           "DeploymentTransformer",
					FQN:            "test#DeploymentTransformer",
					RequiredLabels: map[string]string{"workload-type": "stateless"},
				},
			},
			wantMatched:   0,
			wantUnmatched: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.components, tt.transformers)

			// Count total matches
			totalMatches := 0
			for _, comps := range result.ByTransformer {
				totalMatches += len(comps)
			}

			assert.Equal(t, tt.wantMatched, totalMatches, "matched count")
			assert.Equal(t, tt.wantUnmatched, len(result.Unmatched), "unmatched count")
		})
	}
}

func TestMatchResult_ToMatchPlan(t *testing.T) {
	result := &MatchResult{
		ByTransformer: map[string][]*LoadedComponent{
			"test#DeploymentTransformer": {
				{Name: "webapp"},
			},
		},
		Unmatched: []*LoadedComponent{
			{Name: "unmatched-comp"},
		},
		Details: []MatchDetail{
			{
				ComponentName:  "webapp",
				TransformerFQN: "test#DeploymentTransformer",
				Matched:        true,
				Reason:         "Matched: requiredLabels[workload-type=stateless]",
			},
		},
	}

	plan := result.ToMatchPlan()

	assert.Len(t, plan.Matches, 1)
	assert.Len(t, plan.Matches["webapp"], 1)
	assert.Equal(t, "test#DeploymentTransformer", plan.Matches["webapp"][0].TransformerFQN)
	assert.Len(t, plan.Unmatched, 1)
	assert.Equal(t, "unmatched-comp", plan.Unmatched[0])
}

func TestMatcher_buildReason(t *testing.T) {
	matcher := NewMatcher()

	tests := []struct {
		name        string
		detail      MatchDetail
		transformer *LoadedTransformer
		wantPrefix  string
	}{
		{
			name: "matched with labels",
			detail: MatchDetail{
				Matched: true,
			},
			transformer: &LoadedTransformer{
				RequiredLabels: map[string]string{"type": "web"},
			},
			wantPrefix: "Matched:",
		},
		{
			name: "not matched - missing labels",
			detail: MatchDetail{
				Matched:       false,
				MissingLabels: []string{"type"},
			},
			transformer: &LoadedTransformer{},
			wantPrefix:  "Not matched:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := matcher.buildReason(tt.detail, tt.transformer)
			assert.Contains(t, reason, tt.wantPrefix)
		})
	}
}
