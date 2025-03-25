package templates

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed components/* environment/*
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
		"environment/terragrunt.hcl.tmpl",
		"environment/environment.hcl.tmpl",
		"environment/region.hcl.tmpl",
		"environment/subscription.hcl.tmpl",
		"environment/root.hcl.tmpl",
		"environment/global.hcl.tmpl",
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

// EnvironmentTemplateData represents the data needed for environment templates
type EnvironmentTemplateData struct {
	EnvironmentName           string
	EnvironmentPrefix         string
	Region                    string
	RegionPrefix              string
	Subscription              string
	RemoteStateResourceGroup  string
	RemoteStateStorageAccount string
	StackName                 string
	Component                 string
}

// GlobalConfigData represents the data needed for global configuration templates
type GlobalConfigData struct {
	ProjectName string
	Stacks      map[string]StackConfig
}

// StackConfig represents the configuration for a stack
type StackConfig struct {
	Environments map[string]EnvironmentConfig
}

// EnvironmentConfig represents the configuration for an environment
type EnvironmentConfig struct {
	Prefix  string
	Regions map[string]RegionConfig
}

// RegionConfig represents the configuration for a region
type RegionConfig struct {
	Prefix string
}

// Render renders a template file with the given data and writes it to the output file
func Render(templatePath, outputPath string, data interface{}) error {
	// Read the template file from the embedded filesystem
	templateContent, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	// Parse the template
	tmpl, err := template.New(templatePath).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	// Create the output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	// Execute the template
	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return nil
}
