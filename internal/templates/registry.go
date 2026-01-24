package templates

import "fmt"

// DefaultTemplateName is the template used when --template is not specified.
const DefaultTemplateName = "standard"

// templates is the internal registry of available templates.
var templates = map[string]Template{
	"simple": {
		Name:        "simple",
		Description: "Single-file inline - Learning OPM, prototypes",
		UseCase:     "New users learning OPM, quick prototypes, minimal projects",
		Default:     false,
	},
	"standard": {
		Name:        "standard",
		Description: "Separated components - Team projects, production modules",
		UseCase:     "Team collaboration, production modules, maintainable code",
		Default:     true,
	},
	"advanced": {
		Name:        "advanced",
		Description: "Multi-package with subpackages - Complex platforms, enterprise",
		UseCase:     "Complex platforms, multiple teams, enterprise deployments",
		Default:     false,
	},
}

// Get returns a template by name.
// Returns an error if the template is not found.
func Get(name string) (Template, error) {
	t, ok := templates[name]
	if !ok {
		return Template{}, fmt.Errorf("unknown template %q; valid templates: simple, standard, advanced", name)
	}
	return t, nil
}

// List returns all available templates.
func List() []Template {
	return []Template{
		templates["simple"],
		templates["standard"],
		templates["advanced"],
	}
}

// GetDefault returns the default template.
func GetDefault() Template {
	return templates[DefaultTemplateName]
}

// Names returns all template names.
func Names() []string {
	return []string{"simple", "standard", "advanced"}
}
