package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderDiff(t *testing.T) {
	styles := NoColorStyles()

	t.Run("renders no changes message", func(t *testing.T) {
		result := RenderDiff(nil, nil, nil, styles)
		assert.Equal(t, "No changes detected.", result)
	})

	t.Run("renders added resources", func(t *testing.T) {
		added := []string{"Deployment/default/new-app"}
		result := RenderDiff(added, nil, nil, styles)

		assert.Contains(t, result, "Added:")
		assert.Contains(t, result, "+ Deployment/default/new-app")
		assert.Contains(t, result, "1 added")
	})

	t.Run("renders removed resources", func(t *testing.T) {
		removed := []string{"Service/default/old-svc"}
		result := RenderDiff(nil, removed, nil, styles)

		assert.Contains(t, result, "Removed:")
		assert.Contains(t, result, "- Service/default/old-svc")
		assert.Contains(t, result, "1 removed")
	})

	t.Run("renders modified resources", func(t *testing.T) {
		modified := []ModifiedItem{
			{Name: "ConfigMap/default/config", Diff: "spec.data.key:\n  - old\n  + new"},
		}
		result := RenderDiff(nil, nil, modified, styles)

		assert.Contains(t, result, "Modified:")
		assert.Contains(t, result, "~ ConfigMap/default/config")
		assert.Contains(t, result, "spec.data.key")
		assert.Contains(t, result, "1 modified")
	})

	t.Run("renders all change types", func(t *testing.T) {
		added := []string{"Deployment/default/new"}
		removed := []string{"Service/default/old"}
		modified := []ModifiedItem{
			{Name: "ConfigMap/default/config", Diff: "changed"},
		}
		result := RenderDiff(added, removed, modified, styles)

		assert.Contains(t, result, "Added:")
		assert.Contains(t, result, "Removed:")
		assert.Contains(t, result, "Modified:")
		assert.Contains(t, result, "1 added, 1 removed, 1 modified")
	})

	t.Run("renders multiple items per category", func(t *testing.T) {
		added := []string{"Deployment/default/a", "Service/default/b", "ConfigMap/default/c"}
		result := RenderDiff(added, nil, nil, styles)

		assert.Contains(t, result, "Deployment/default/a")
		assert.Contains(t, result, "Service/default/b")
		assert.Contains(t, result, "ConfigMap/default/c")
		assert.Contains(t, result, "3 added")
	})
}

func TestDiffSummary(t *testing.T) {
	tests := []struct {
		name     string
		added    int
		removed  int
		modified int
		want     string
	}{
		{"no changes", 0, 0, 0, "No changes"},
		{"only added", 1, 0, 0, "1 added"},
		{"only removed", 0, 2, 0, "2 removed"},
		{"only modified", 0, 0, 3, "3 modified"},
		{"added and removed", 1, 2, 0, "1 added, 2 removed"},
		{"all types", 1, 2, 3, "1 added, 2 removed, 3 modified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffSummary(tt.added, tt.removed, tt.modified)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIndentDiff(t *testing.T) {
	t.Run("indents each line", func(t *testing.T) {
		input := "line1\nline2\nline3"
		result := IndentDiff(input, "    ")

		expected := "    line1\n    line2\n    line3\n"
		assert.Equal(t, expected, result)
	})

	t.Run("skips empty lines", func(t *testing.T) {
		input := "line1\n\nline2"
		result := IndentDiff(input, "  ")

		expected := "  line1\n  line2\n"
		assert.Equal(t, expected, result)
	})

	t.Run("returns empty for empty input", func(t *testing.T) {
		result := IndentDiff("", "    ")
		assert.Empty(t, result)
	})
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-1, "-1"},
		{-123, "-123"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := itoa(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
