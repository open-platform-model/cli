package k8sgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Names
		wantErr string
	}{
		{
			name:  "kebab case",
			input: "my-service",
			want: Names{
				Kind:     "MyService",
				ListKind: "MyServiceList",
				Singular: "myservice",
				Plural:   "myservices",
			},
		},
		{
			name:  "single segment",
			input: "service",
			want: Names{
				Kind:     "Service",
				ListKind: "ServiceList",
				Singular: "service",
				Plural:   "services",
			},
		},
		{
			name:  "snake case",
			input: "my_thing",
			want: Names{
				Kind:     "MyThing",
				ListKind: "MyThingList",
				Singular: "mything",
				Plural:   "mythings",
			},
		},
		{
			name:  "mixed separators lowercases letters inside segments",
			input: "Mixed-Case_Name",
			want: Names{
				Kind:     "MixedCaseName",
				ListKind: "MixedCaseNameList",
				Singular: "mixedcasename",
				Plural:   "mixedcasenames",
			},
		},
		{
			name:  "already PascalCase passes through",
			input: "Service",
			want: Names{
				Kind:     "Service",
				ListKind: "ServiceList",
				Singular: "service",
				Plural:   "services",
			},
		},
		{
			name:  "consecutive separators collapse",
			input: "my--service",
			want: Names{
				Kind:     "MyService",
				ListKind: "MyServiceList",
				Singular: "myservice",
				Plural:   "myservices",
			},
		},
		{
			name:  "leading separator stripped",
			input: "_leading-sep",
			want: Names{
				Kind:     "LeadingSep",
				ListKind: "LeadingSepList",
				Singular: "leadingsep",
				Plural:   "leadingseps",
			},
		},
		{
			name:  "segments with digits preserved",
			input: "service-v2",
			want: Names{
				Kind:     "ServiceV2",
				ListKind: "ServiceV2List",
				Singular: "servicev2",
				Plural:   "servicev2s",
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: "module name is empty",
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: "module name is empty",
		},
		{
			name:    "only separators",
			input:   "-_-",
			wantErr: "contains only separators",
		},
		{
			name:    "leading digit",
			input:   "123abc",
			wantErr: "invalid CRD kind",
		},
		{
			name:    "unsupported character",
			input:   "my$service",
			wantErr: "invalid CRD kind",
		},
		{
			name:    "internal whitespace is unsupported",
			input:   "my service",
			wantErr: "invalid CRD kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := DeriveNames(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeriveVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "major zero maps to v1alpha1", input: "0.1.0", want: "v1alpha1"},
		{name: "major zero zero maps to v1alpha1", input: "0.0.0", want: "v1alpha1"},
		{name: "major one", input: "1.0.0", want: "v1"},
		{name: "major two", input: "2.3.4", want: "v2"},
		{name: "double digit major", input: "12.0.0", want: "v12"},
		{name: "leading v tolerated", input: "v1.0.0", want: "v1"},
		{name: "prerelease stripped (still v1)", input: "1.0.0-beta", want: "v1"},
		{name: "prerelease stripped on major 0", input: "0.5.0-alpha", want: "v1alpha1"},
		{name: "build metadata stripped", input: "1.0.0+build.123", want: "v1"},
		{name: "prerelease and build metadata", input: "1.0.0-rc.1+build.7", want: "v1"},
		{name: "whitespace trimmed", input: "  2.0.0  ", want: "v2"},

		{name: "empty string", input: "", wantErr: "module version is empty"},
		{name: "only v prefix", input: "v", wantErr: "module version is empty"},
		{name: "non-numeric major", input: "foo.1.0", wantErr: "non-numeric major"},
		{name: "leading-dash strips to empty", input: "-1.0.0", wantErr: "no major component"},
		{name: "leading-plus strips to empty", input: "+1.0.0", wantErr: "no major component"},
		{name: "just a dot", input: ".", wantErr: "no major component"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := DeriveVersion(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
