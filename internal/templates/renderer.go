package templates

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"text/template"
)

//go:embed components/*
var templateFS embed.FS

// TemplateRenderer handles loading and rendering of templates
type TemplateRenderer struct {
	templates map[string]*template.Template
}

// NewRenderer creates a new template renderer
func NewRenderer() (*TemplateRenderer, error) {
	r := &TemplateRenderer{
		templates: make(map[string]*template.Template),
	}

	// Load all templates from the embedded filesystem
	templates := []string{
		"components/component.hcl.tmpl",
		"components/resource_naming.hcl.tmpl",
		"components/dependency.hcl.tmpl",
	}

	for _, tmpl := range templates {
		content, err := templateFS.ReadFile(tmpl)
		if err != nil {
			return nil, fmt.Errorf("failed to read template %s: %w", tmpl, err)
		}

		t, err := template.New(filepath.Base(tmpl)).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", tmpl, err)
		}

		r.templates[tmpl] = t
	}

	return r, nil
}

// RenderTemplate renders a template with the given data
func (r *TemplateRenderer) RenderTemplate(name string, data interface{}) (string, error) {
	tmpl, ok := r.templates[name]
	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", name, err)
	}

	return buf.String(), nil
}

// ComponentData represents the data needed for component templates
type ComponentData struct {
	StackName        string
	ComponentName    string
	Source           string
	Version          string
	ResourceType     string
	DependencyBlocks string
	EnvConfigInputs  string
}

// ResourceNamingData represents the data needed for resource naming templates
type ResourceNamingData struct {
	ResourceType string
	Format       string
}

// DependencyData represents the data needed for dependency templates
type DependencyData struct {
	Name       string
	ConfigPath string
}
