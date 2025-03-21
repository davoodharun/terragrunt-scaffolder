package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

func generateComponents(mainConfig *config.MainConfig) error {
	logger.Info("Generating components")

	// Get the infrastructure path
	infraPath := getInfrastructurePath()

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

	// Generate each component
	for compName, comp := range mainConfig.Stack.Components {
		// Skip if we've already validated this component
		if validatedComponents[compName] {
			continue
		}

		logger.Info("Generating component: %s", compName)

		// Create component directory under the stack-specific directory
		componentPath := filepath.Join(stackComponentsDir, compName)
		if err := os.MkdirAll(componentPath, 0755); err != nil {
			return err
		}

		// Generate Terraform files from provider schema if available
		if err := generateTerraformFiles(componentPath, comp); err != nil {
			return err
		}

		// Fetch provider schema for this component
		var requiredInputs string
		if comp.Provider != "" {
			schema, err := fetchProviderSchema(comp.Provider, comp.Version, comp.Source)
			if err == nil && schema != nil {
				// Try different provider keys
				providerKeys := []string{
					"registry.terraform.io/hashicorp/azurerm",
					"hashicorp/azurerm",
				}

				for _, key := range providerKeys {
					if provider, ok := schema.ProviderSchema[key]; ok {
						if resourceSchema, ok := provider.ResourceSchemas[comp.Source]; ok {
							var requiredFields []string
							for name, attr := range resourceSchema.Block.Attributes {
								if attr.Required && !shouldSkipVariable(name, comp.Source) {
									if name == "service_plan_id" && strings.Contains(comp.Source, "web_app") {
										requiredFields = append(requiredFields, fmt.Sprintf("%s = dependency.serviceplan.outputs.id", name))
									} else {
										requiredFields = append(requiredFields, fmt.Sprintf("%s = try(local.env_config.locals.%s.%s, null)", name, compName, name))
									}
								}
							}
							if len(requiredFields) > 0 {
								requiredInputs = "# Required settings from provider schema\n" + strings.Join(requiredFields, "\n")
							}
							break
						}
					}
				}
			}
		}

		// If no schema-based inputs, fall back to basic inputs from generateEnvConfigInputs
		if requiredInputs == "" {
			requiredInputs = generateEnvConfigInputs(compName)
		}

		// Create component.hcl with dependency blocks
		componentHcl := fmt.Sprintf(`
locals {
  subscription_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl"))
  region_vars = read_terragrunt_config(find_in_parent_folders("region.hcl"))
  environment_vars = read_terragrunt_config(find_in_parent_folders("environment.hcl"))
  
  # Load global and environment-specific configurations
  global_config = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/global.hcl")
  env_config = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/${local.environment_vars.locals.environment_name}.hcl")
  
  # Common variables
  project_name = local.global_config.locals.project_name
  subscription_name = local.subscription_vars.locals.subscription_name
  region_name = local.region_vars.locals.region_name
  region_prefix = local.region_vars.locals.region_prefix
  environment_name = local.environment_vars.locals.environment_name
  environment_prefix = local.environment_vars.locals.environment_prefix
  
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
  source = "${get_repo_root()}/.infrastructure/_components/%s/%s"
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
    }
  )

  # Include environment-specific configurations based on component type
%s
}`,
			getResourceTypeAbbreviation(compName), mainConfig.Stack.Name, compName, generateDependencyBlocks(comp.Deps, infraPath), requiredInputs)

		if err := createFile(filepath.Join(componentPath, "component.hcl"), componentHcl); err != nil {
			return err
		}

		// Validate component structure
		if err := ValidateComponentStructure(componentPath); err != nil {
			return fmt.Errorf("component structure validation failed for %s: %w", compName, err)
		}

		// Validate component variables against environment config
		envConfigPath := filepath.Join(infraPath, "config", "dev.hcl") // Use dev.hcl as base for validation
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
func getResourceTypeAbbreviation(resourceType string) string {
	resourceAbbreviations := map[string]string{
		"serviceplan":    "svcpln",
		"appservice":     "app",
		"appservice_api": "app",
		"functionapp":    "fncapp",
		"rediscache":     "cache",
		"keyvault":       "kv",
		"servicebus":     "sbus",
		"cosmos_account": "cosmos",
		"cosmos_db":      "cdb",
		"apim":           "apim",
		"storage":        "st",
		"sql_server":     "sql",
		"sql_database":   "sqldb",
		"eventhub":       "evhub",
		"loganalytics":   "log",
	}

	if abbr, ok := resourceAbbreviations[resourceType]; ok {
		return abbr
	}

	// If no abbreviation found, return first 3 letters of the resource type
	if len(resourceType) > 3 {
		return resourceType[:3]
	}
	return resourceType
}

// Helper function to generate environment-specific inputs based on component type
func generateEnvConfigInputs(compName string) string {
	switch compName {
	case "serviceplan":
		return `# Service Plan specific settings
sku_name = try(local.env_config.locals.serviceplan.sku_name, "B1")`
	default:
		return ""
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
