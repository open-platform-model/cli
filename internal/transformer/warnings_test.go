package transformer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/transformer"
)

func TestCollectWarnings(t *testing.T) {
	tests := []struct {
		name     string
		matches  []*core.TransformerMatch
		wantWarn []string // nil means expect no warnings
	}{
		{
			name: "trait handled by at least one matched transformer — no warning",
			// Two transformers match the same component. One handles "expose", the other doesn't.
			// Because one handles it, it must NOT be warned.
			matches: []*core.TransformerMatch{
				{
					Matched: true,
					Detail: &core.TransformerMatchDetail{
						ComponentName:   "web",
						TransformerFQN:  "k8s#deployment",
						UnhandledTraits: []string{}, // deployment handles expose
					},
				},
				{
					Matched: true,
					Detail: &core.TransformerMatchDetail{
						ComponentName:   "web",
						TransformerFQN:  "k8s#service",
						UnhandledTraits: []string{"expose"}, // service doesn't, but deployment does
					},
				},
			},
			wantWarn: nil,
		},
		{
			name: "trait unhandled by all matched transformers — warning emitted",
			// Both matched transformers report "logging" as unhandled.
			matches: []*core.TransformerMatch{
				{
					Matched: true,
					Detail: &core.TransformerMatchDetail{
						ComponentName:   "api",
						TransformerFQN:  "k8s#deployment",
						UnhandledTraits: []string{"logging"},
					},
				},
				{
					Matched: true,
					Detail: &core.TransformerMatchDetail{
						ComponentName:   "api",
						TransformerFQN:  "k8s#service",
						UnhandledTraits: []string{"logging"},
					},
				},
			},
			wantWarn: []string{"component api: unhandled trait logging"},
		},
		{
			name: "component with no matched transformers — no warnings emitted",
			matches: []*core.TransformerMatch{
				{
					Matched: false,
					Detail: &core.TransformerMatchDetail{
						ComponentName:  "orphan",
						TransformerFQN: "k8s#deployment",
					},
				},
			},
			wantWarn: nil,
		},
		{
			name: "multiple components, mixed handled/unhandled traits — only truly unhandled warned",
			// Component "web": "expose" handled by one transformer → no warning.
			// Component "api": "logging" unhandled by both → warning.
			matches: []*core.TransformerMatch{
				// web — expose handled by deployment
				{
					Matched: true,
					Detail: &core.TransformerMatchDetail{
						ComponentName:   "web",
						TransformerFQN:  "k8s#deployment",
						UnhandledTraits: []string{},
					},
				},
				{
					Matched: true,
					Detail: &core.TransformerMatchDetail{
						ComponentName:   "web",
						TransformerFQN:  "k8s#service",
						UnhandledTraits: []string{"expose"},
					},
				},
				// api — logging unhandled by both
				{
					Matched: true,
					Detail: &core.TransformerMatchDetail{
						ComponentName:   "api",
						TransformerFQN:  "k8s#deployment",
						UnhandledTraits: []string{"logging"},
					},
				},
				{
					Matched: true,
					Detail: &core.TransformerMatchDetail{
						ComponentName:   "api",
						TransformerFQN:  "k8s#service",
						UnhandledTraits: []string{"logging"},
					},
				},
			},
			wantWarn: []string{"component api: unhandled trait logging"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan := &core.TransformerMatchPlan{
				Matches: tc.matches,
			}
			got := transformer.CollectWarnings(plan)
			if tc.wantWarn == nil {
				assert.Empty(t, got)
			} else {
				assert.ElementsMatch(t, tc.wantWarn, got)
			}
		})
	}
}
