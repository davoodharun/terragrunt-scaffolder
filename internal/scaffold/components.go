package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"github.com/davoodharun/terragrunt-scaffolder/internal/templates"
)

func generateComponents(mainConfig *config.MainConfig, infraPath string) error {
	// Initialize template renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to initialize template renderer: %w", err)
	}

	// Create components directory
	componentsDir := filepath.Join(infraPath, "_components")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create components directory: %w", err)
	}

	// Create stack-specific components directory
	stackComponentsDir := filepath.Join(componentsDir, mainConfig.Stack.Name)
	if err := os.MkdirAll(stackComponentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create stack components directory: %w", err)
	}

	// Track validated components to avoid duplicate messages
	validatedComponents := make(map[string]bool)

	// Generate component files
	for compName, comp := range mainConfig.Stack.Components {
		if validatedComponents[compName] {
			continue
		}

		// Create component directory
		componentPath := filepath.Join(stackComponentsDir, compName)
		if err := os.MkdirAll(componentPath, 0755); err != nil {
			return fmt.Errorf("failed to create component directory: %w", err)
		}

		// Generate Terraform files
		if err := generateTerraformFiles(componentPath, comp); err != nil {
			return fmt.Errorf("failed to generate terraform files: %w", err)
		}

		// Generate dependency blocks
		var dependencyBlocks string
		if len(comp.Deps) > 0 {
			deps := generateDependencyBlocks(comp.Deps, infraPath)
			dependencyBlocks = deps
		}

		// Prepare component data
		componentData := &templates.ComponentData{
			StackName:        mainConfig.Stack.Name,
			ComponentName:    compName,
			Source:           comp.Source,
			Version:          comp.Version,
			ResourceType:     getResourceTypeAbbreviation(compName),
			DependencyBlocks: dependencyBlocks,
			EnvConfigInputs:  generateEnvConfigInputs(comp),
		}

		// Render component.hcl template
		componentHcl, err := renderer.RenderTemplate("components/component.hcl.tmpl", componentData)
		if err != nil {
			return fmt.Errorf("failed to render component.hcl template: %w", err)
		}

		// Write component.hcl file
		if err := createFile(filepath.Join(componentPath, "component.hcl"), componentHcl); err != nil {
			return fmt.Errorf("failed to create component.hcl: %w", err)
		}

		// Validate component structure
		if err := ValidateComponentStructure(componentPath); err != nil {
			return fmt.Errorf("component structure validation failed for %s: %w", compName, err)
		}

		// Validate component variables against environment config
		envConfigPath := filepath.Join(infraPath, "config", mainConfig.Stack.Name, "dev.hcl") // Use dev.hcl as base for validation
		if err := ValidateComponentVariables(componentPath, envConfigPath); err != nil {
			return fmt.Errorf("component variables validation failed for %s: %w", compName, err)
		}

		logger.Success("Generated and validated component: %s", compName)
		logger.UpdateProgress()

		// Mark this component as validated
		validatedComponents[compName] = true
	}

	return nil
}

// Helper function to get resource type abbreviation
func getResourceTypeAbbreviation(componentName string) string {
	abbreviations := map[string]string{
		"serviceplan": "asp",
		"appservice":  "app",
		"functionapp": "func",
		"redis":       "redis",
		"storage":     "st",
		"keyvault":    "kv",
		"sql":         "sql",
		"cosmos":      "cos",
	}

	for key, abbr := range abbreviations {
		if strings.Contains(strings.ToLower(componentName), key) {
			return abbr
		}
	}

	// Default to first three letters if no match
	if len(componentName) >= 3 {
		return strings.ToLower(componentName[0:3])
	}
	return strings.ToLower(componentName)
}

// Helper function to generate environment-specific inputs based on component type
func generateEnvConfigInputs(comp config.Component) string {
	// Extract component type from source
	compType := strings.TrimPrefix(comp.Source, "azurerm_")

	switch compType {
	case "service_plan":
		return `# Service Plan specific settings
    sku_name = try(local.env_vars.locals.serviceplan.sku_name, "B1")
    os_type = try(local.env_vars.locals.serviceplan.os_type, "Linux")`
	default:
		return "# No specific inputs required for this component type"
	}
}

// Helper function to generate dependency blocks
func generateDependencyBlocks(deps []string, infraPath string) string {
	if len(deps) == 0 {
		return ""
	}

	// Initialize template renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		logger.Warning("Failed to initialize template renderer: %v", err)
		return ""
	}

	var blocks []string
	for _, dep := range deps {
		parts := strings.Split(dep, ".")

		if len(parts) < 2 {
			logger.Warning("Invalid dependency format: %s, skipping", dep)
			continue
		}

		region := parts[0]
		component := parts[1]
		app := ""

		if len(parts) > 2 {
			app = parts[2]
		}

		// Replace placeholders
		if region == "{region}" {
			region = "${local.region_vars.locals.region_name}"
		}

		depName := component
		configPath := ""

		if app == "" || app == "{app}" {
			if app == "{app}" {
				// App-specific dependency using current app
				configPath = fmt.Sprintf("${get_repo_root()}/.infrastructure/${local.subscription_name}/%s/${local.environment_name}/%s/${local.app_name}", region, component)
			} else {
				// Component-level dependency
				configPath = fmt.Sprintf("${get_repo_root()}/.infrastructure/${local.subscription_name}/%s/${local.environment_name}/%s", region, component)
			}
		} else {
			// App-specific dependency with fixed app name
			configPath = fmt.Sprintf("${get_repo_root()}/.infrastructure/${local.subscription_name}/%s/${local.environment_name}/%s/%s", region, component, app)
			depName = fmt.Sprintf("%s_%s", component, app)
		}

		// Render dependency template
		dependencyData := &templates.DependencyData{
			Name:       depName,
			ConfigPath: configPath,
		}
		block, err := renderer.RenderTemplate("components/dependency.hcl.tmpl", dependencyData)
		if err != nil {
			logger.Warning("Failed to render dependency template: %v", err)
			continue
		}
		blocks = append(blocks, block)
	}

	return strings.Join(blocks, "\n")
}

func generateComponentFile(infraPath string, component *config.Component, stackName string) error {
	// Read the template file
	templatePath := filepath.Join(infraPath, "_components", stackName, fmt.Sprintf("%s.tf", component.Source))
	template, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Get naming configuration from TGS config
	tgsConfig, err := ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to load TGS config: %w", err)
	}
	namingConfig := tgsConfig.Naming

	// Initialize template renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to initialize template renderer: %w", err)
	}

	// Get resource prefix for this component type
	resourcePrefix := namingConfig.ResourcePrefixes[component.Source]
	if resourcePrefix == "" {
		resourcePrefix = component.Source
	}

	// Get format for this component
	format := namingConfig.Format
	if componentFormat, exists := namingConfig.ComponentFormats[component.Source]; exists {
		if componentFormat.Format != "" {
			format = componentFormat.Format
		}
	}

	// Render resource naming template
	namingData := &templates.ResourceNamingData{
		ResourceType: resourcePrefix,
		Format:       format,
	}
	locals, err := renderer.RenderTemplate("components/resource_naming.hcl.tmpl", namingData)
	if err != nil {
		return fmt.Errorf("failed to render resource naming template: %w", err)
	}

	// Update template with new locals block
	content := string(template)
	content = strings.Replace(content, "locals {", locals, 1)

	// Write the updated content back to the file
	err = os.WriteFile(templatePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write component file: %w", err)
	}

	return nil
}
