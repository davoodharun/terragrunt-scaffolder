package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

func generateComponents(mainConfig *config.MainConfig, infraPath string) error {
	logger.Info("Generating components")

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

		logger.Info("Generating component: %s", compName)

		// Create component directory
		componentPath := filepath.Join(stackComponentsDir, compName)
		if err := os.MkdirAll(componentPath, 0755); err != nil {
			return fmt.Errorf("failed to create component directory: %w", err)
		}

		// Generate Terraform files
		if err := generateTerraformFiles(componentPath, comp); err != nil {
			return fmt.Errorf("failed to generate terraform files: %w", err)
		}

		// Build dependency blocks
		var dependencyBlocks strings.Builder
		if len(comp.Deps) > 0 {
			deps := generateDependencyBlocks(comp.Deps, infraPath)
			dependencyBlocks.WriteString(deps)
		}

		// Generate component.hcl
		componentHcl := fmt.Sprintf(`locals {
  # Stack-specific configuration
  stack_name = "%s"

  # Load configuration files
  subscription_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl"))
  region_vars = read_terragrunt_config(find_in_parent_folders("region.hcl"))
  environment_vars = read_terragrunt_config(find_in_parent_folders("environment.hcl"))
  global_config = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/global.hcl")
  env_config = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/${local.stack_name}/${local.environment_vars.locals.environment_name}.hcl")

  # Common variables
  project_name = local.global_config.locals.project_name
  subscription_name = local.subscription_vars.locals.subscription_name
  region_name = local.region_vars.locals.region_name
  region_prefix = local.region_vars.locals.region_prefix
  environment_name = local.environment_vars.locals.environment_name
  environment_prefix = local.environment_vars.locals.environment_prefix

  # Component configuration
  component_name = "%s"
  provider_source = "%s"
  provider_version = "%s"

  # Get the directory name as the app name, defaulting to empty string if at component root
  app_name = try(basename(dirname(get_terragrunt_dir())), basename(get_terragrunt_dir()), "")

  # Resource type abbreviation
  resource_type = "%s"

  # Resource naming convention with prefixes and resource type
  name_prefix = "${local.project_name}-${local.region_prefix}${local.environment_prefix}-${local.resource_type}"
  resource_name = local.app_name != "" ? "${local.name_prefix}-${local.app_name}" : local.name_prefix

  # Get resource group name from global config
  resource_group_name = local.global_config.locals.resource_groups[local.environment_name][local.region_name]
}

terraform {
  source = "${get_repo_root()}/.infrastructure/_components/${local.stack_name}/${local.component_name}"
}

%s

inputs = {
  # Resource identification
  name = local.resource_name
  resource_group_name = local.resource_group_name
  location = local.region_name

  # Tags with context information embedded
  tags = merge(
    try(local.global_config.locals.common_tags, {}),
    {
      Environment = local.environment_name
      Application = local.app_name
      Project = local.project_name
      Region = local.region_name
      Stack = local.stack_name
      Component = local.component_name
    }
  )

  # Include environment-specific configurations based on component type
%s
}`, mainConfig.Stack.Name, compName, comp.Source, comp.Version,
			getResourceTypeAbbreviation(compName),
			dependencyBlocks.String(),
			generateEnvConfigInputs(comp))

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

		block := fmt.Sprintf(`
dependency "%s" {
  config_path = "%s"
}`, depName, configPath)
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

	// Get resource prefix for this component type
	resourcePrefix := namingConfig.ResourcePrefixes[component.Source] // Using component.Source as the type
	if resourcePrefix == "" {
		resourcePrefix = component.Source // Fallback to component source if no prefix defined
	}

	// Get format and separator for this component
	format := namingConfig.Format
	if componentFormat, exists := namingConfig.ComponentFormats[component.Source]; exists {
		if componentFormat.Format != "" {
			format = componentFormat.Format
		}
	}

	// Create locals block for naming
	locals := fmt.Sprintf(`locals {
  project_name = var.project_name
  region_prefix = var.region_prefix
  environment_prefix = var.environment_prefix
  resource_type = "%s"
  
  // Resource naming using configured format
  resource_name = replace(
    replace(
      replace(
        replace(
          replace(
            "%s",
            "${project}", local.project_name
          ),
          "${region}", local.region_prefix
        ),
        "${env}", local.environment_prefix
      ),
      "${type}", local.resource_type
    ),
    "${app}", try(var.app_name, "")
  )
}`, resourcePrefix, format)

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
