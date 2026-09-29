package modref

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// refusalOf asserts err is a refusal and returns it.
func refusalOf(t *testing.T, err error) *RefusalError {
	t.Helper()
	var r *RefusalError
	require.True(t, errors.As(err, &r), "want a refusal, got %v", err)
	return r
}

func TestParsePath(t *testing.T) {
	valid := []string{
		"opmodel.dev/modules/web_app",
		"example.com/m",
		"testing.opmodel.dev/modules/cli/podinfo-app",
	}
	for _, p := range valid {
		got, err := ParsePath(p)
		require.NoError(t, err, p)
		assert.Equal(t, p, got)
	}

	cases := []struct {
		arg, headline, action string
	}{
		{"opmodel.dev/modules/web_app@v1", "must not carry a major", "opmodel.dev/modules/web_app --version v1"},
		{"opmodel.dev/modules/web_app@v1.0.4", "must not carry a major", "opmodel.dev/modules/web_app --version 1.0.4"},
		{"opmodel.dev/modules/web_app@latest", "must not carry a major", "opmodel.dev/modules/web_app --version <vN | X.Y.Z>"},
		{"oci://ghcr.io/open-platform-model/opmodel.dev/modules/web_app", "a module is named by its module path", ""},
		{"https://opmodel.dev/modules/web_app", "a module is named by its module path", ""},
		{"opmodel.dev", "not a module path", ""},
		{"modules/web_app", "not a module path", ""},
		{"opmodel.dev/modules/WebApp", "not a module path", ""},
		{"opmodel.dev//web_app", "not a module path", ""},
	}
	for _, c := range cases {
		_, err := ParsePath(c.arg)
		r := refusalOf(t, err)
		assert.Contains(t, r.Refusal.Headline, c.headline, c.arg)
		assert.Contains(t, r.Refusal.Action, c.action, c.arg)
	}
}

func TestParseSelector(t *testing.T) {
	valid := map[string]Selector{
		"":                       {},
		"v0":                     {Major: "v0"},
		"v1":                     {Major: "v1"},
		"v12":                    {Major: "v12"},
		"1.0.4":                  {Exact: "1.0.4"},
		"2.0.0-alpha.2":          {Exact: "2.0.0-alpha.2"},
		"1.2.0-0.dev.3.gabc1234": {Exact: "1.2.0-0.dev.3.gabc1234"},
	}
	for in, want := range valid {
		got, err := ParseSelector(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}

	for _, in := range []string{"v1.0.4", "1", "1.0", "v01", "latest", "1.0.4+build", "V1"} {
		_, err := ParseSelector(in)
		r := refusalOf(t, err)
		assert.Contains(t, r.Refusal.Headline, "v1 or 1.0.4", in)
		assert.Contains(t, r.Refusal.Headline, in, in)
	}
}

func TestReport(t *testing.T) {
	cases := []struct {
		name string
		res  Resolution
		want []string
	}{
		{
			name: "pinned",
			res:  Resolution{Path: "opmodel.dev/modules/web_app", Major: "v1", Version: "v1.0.3", Strategy: Exact, CoreMajor: "v2"},
			want: []string{"Resolved opmodel.dev/modules/web_app -> v1 1.0.3 (pinned)"},
		},
		{
			name: "float",
			res:  Resolution{Path: "opmodel.dev/modules/web_app", Major: "v1", Version: "v1.0.4", Strategy: FloatMajor, CoreMajor: "v2"},
			want: []string{"Resolved opmodel.dev/modules/web_app -> v1 1.0.4 (newest in v1)"},
		},
		{
			name: "highest compatible with skips",
			res: Resolution{
				Path: "opmodel.dev/modules/web_app", Major: "v1", Version: "v1.0.4", Strategy: HighestCompatibleMajor, CoreMajor: "v2",
				Skipped: []Skipped{{"v3", "declares no opmodel.dev/core dependency"}, {"v2", "requires core v3"}},
			},
			want: []string{
				"Resolved opmodel.dev/modules/web_app -> v1 1.0.4 (highest major on core v2)",
				"  skipped v3: declares no opmodel.dev/core dependency",
				"  skipped v2: requires core v3",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, Report(&c.res))
		})
	}
}

func TestResolutionImport(t *testing.T) {
	r := &Resolution{Path: "opmodel.dev/modules/web_app", Major: "v1"}
	assert.Equal(t, "opmodel.dev/modules/web_app@v1", r.Import())
}
