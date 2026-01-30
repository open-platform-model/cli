// Package templates provides embedded module templates and rendering.
package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidTemplates(t *testing.T) {
	templates := ValidTemplates()
	assert.Len(t, templates, 3)
	assert.Contains(t, templates, "simple")
	assert.Contains(t, templates, "standard")
	assert.Contains(t, templates, "advanced")
}

func TestIsValidTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     bool
	}{
		{"simple is valid", "simple", true},
		{"standard is valid", "standard", true},
		{"advanced is valid", "advanced", true},
		{"unknown is invalid", "unknown", false},
		{"empty is invalid", "", false},
		{"SIMPLE case-sensitive", "SIMPLE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidTemplate(tt.template)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListTemplateFiles(t *testing.T) {
	tests := []struct {
		name         string
		template     TemplateName
		wantFiles    []string
		wantMinCount int
		wantErr      bool
	}{
		{
			name:     "simple template files",
			template: Simple,
			wantFiles: []string{
				"module.cue",
				"values.cue",
				"cue.mod/module.cue",
			},
			wantMinCount: 3,
		},
		{
			name:     "standard template files",
			template: Standard,
			wantFiles: []string{
				"module.cue",
				"values.cue",
				"components.cue",
				"cue.mod/module.cue",
			},
			wantMinCount: 4,
		},
		{
			name:         "advanced template files",
			template:     Advanced,
			wantMinCount: 10, // Has components/, scopes/ subdirectories
		},
		{
			name:     "unknown template",
			template: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := ListTemplateFiles(tt.template)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(files), tt.wantMinCount)

			// Check specific expected files
			for _, wantFile := range tt.wantFiles {
				assert.Contains(t, files, wantFile, "missing expected file: %s", wantFile)
			}

			// Verify no .tmpl extensions in output
			for _, f := range files {
				assert.False(t, strings.HasSuffix(f, ".tmpl"), "file should not have .tmpl suffix: %s", f)
			}
		})
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name         string
		template     TemplateName
		data         TemplateData
		wantFiles    []string
		wantContains map[string]string // file -> expected content substring
		wantErr      bool
	}{
		{
			name:     "render simple template",
			template: Simple,
			data: TemplateData{
				ModuleName: "test-app",
				ModulePath: "example.com/test-app@v0",
				Version:    "0.1.0",
			},
			wantFiles: []string{
				"module.cue",
				"values.cue",
				"cue.mod/module.cue",
			},
			wantContains: map[string]string{
				"module.cue":         "test-app",
				"cue.mod/module.cue": "example.com/test-app@v0",
			},
		},
		{
			name:     "render standard template",
			template: Standard,
			data: TemplateData{
				ModuleName: "my-service",
				ModulePath: "mycompany.com/my-service@v0",
				Version:    "1.0.0",
			},
			wantFiles: []string{
				"module.cue",
				"values.cue",
				"components.cue",
				"cue.mod/module.cue",
			},
		},
		{
			name:     "unknown template",
			template: "invalid",
			data:     TemplateData{ModuleName: "test"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "template-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Render template
			createdFiles, err := Render(tt.template, tmpDir, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Check all expected files were created
			for _, wantFile := range tt.wantFiles {
				fullPath := filepath.Join(tmpDir, wantFile)
				assert.FileExists(t, fullPath, "expected file not created: %s", wantFile)
				assert.Contains(t, createdFiles, wantFile, "file not in createdFiles list: %s", wantFile)
			}

			// Check content contains expected strings
			for file, wantContent := range tt.wantContains {
				fullPath := filepath.Join(tmpDir, file)
				content, err := os.ReadFile(fullPath)
				require.NoError(t, err)
				assert.Contains(t, string(content), wantContent,
					"file %s should contain %q", file, wantContent)
			}

			// Verify no .tmpl files in output
			err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					assert.False(t, strings.HasSuffix(path, ".tmpl"),
						"output should not contain .tmpl files: %s", path)
				}
				return nil
			})
			require.NoError(t, err)
		})
	}
}

func TestRender_DirectoryExists(t *testing.T) {
	// Create temp directory with existing file
	tmpDir, err := os.MkdirTemp("", "template-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Rendering to an existing directory should work (files get overwritten)
	data := TemplateData{
		ModuleName: "test-app",
		ModulePath: "example.com/test-app@v0",
		Version:    "0.1.0",
	}

	_, err = Render(Simple, tmpDir, data)
	assert.NoError(t, err)
}

func TestRender_AdvancedTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "template-test-advanced-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	data := TemplateData{
		ModuleName: "platform-app",
		ModulePath: "platform.io/platform-app@v0",
		Version:    "0.1.0",
	}

	createdFiles, err := Render(Advanced, tmpDir, data)
	require.NoError(t, err)

	// Advanced template should have components and scopes subdirectories
	expectedDirs := []string{
		"components",
		"scopes",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(tmpDir, dir)
		info, err := os.Stat(dirPath)
		require.NoError(t, err, "expected directory: %s", dir)
		assert.True(t, info.IsDir(), "%s should be a directory", dir)
	}

	// Should have multiple files
	assert.Greater(t, len(createdFiles), 10, "advanced template should create many files")
}

func TestGetFS(t *testing.T) {
	tests := []struct {
		name     string
		template TemplateName
		wantRoot string
		wantErr  bool
	}{
		{"simple", Simple, "simple", false},
		{"standard", Standard, "standard", false},
		{"advanced", Advanced, "advanced", false},
		{"invalid", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, root, err := getFS(tt.template)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRoot, root)
		})
	}
}
