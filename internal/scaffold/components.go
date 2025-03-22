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
	baseDir := filepath.Base(infraPath)
	componentsDir := filepath.Join(baseDir, "_components")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create components directory: %w", err)
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

		// Create component directory
		componentPath := filepath.Join(componentsDir, compName)
		if err := os.MkdirAll(componentPath, 0755); err != nil {
			return err
		}

		// Fetch provider schema
		schema, err := fetchProviderSchema(comp.Provider, comp.Version, comp.Source)
		if err != nil {
			logger.Warning("Failed to fetch provider schema: %v", err)
			schema = nil
		}

		// Generate Terraform files from provider schema if available
		if err := generateTerraformFiles(componentPath, comp); err != nil {
			return err
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
  source = "${get_repo_root()}/.infrastructure/_components/%s"
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
			getResourceTypeAbbreviation(comp.Source),
			comp.Source,
			generateDependencyBlocks(comp.Deps, infraPath),
			generateResourceSpecificInputs(comp.Source, schema))

		if err := createFile(filepath.Join(componentPath, "component.hcl"), componentHcl); err != nil {
			return err
		}

		// Validate component structure
		if err := ValidateComponentStructure(componentPath); err != nil {
			return fmt.Errorf("component structure validation failed for %s: %w", compName, err)
		}

		// Validate component variables against environment config
		envConfigPath := filepath.Join(baseDir, "config", "dev.hcl") // Use dev.hcl as base for validation
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
	switch resourceType {
	case "azurerm_linux_web_app", "azurerm_windows_web_app":
		return "appsvc"
	case "azurerm_service_plan":
		return "svcplan"
	case "azurerm_redis_cache":
		return "redis"
	case "azurerm_key_vault":
		return "kv"
	default:
		return strings.TrimPrefix(resourceType, "azurerm_")
	}
}

// Helper function to generate environment-specific inputs based on component type
func generateResourceSpecificInputs(resourceType string, schema *ProviderSchema) string {
	if schema != nil {
		required := getRequiredAttributes(schema, resourceType)
		var inputs []string

		// Handle common required attributes
		for _, attr := range required {
			switch attr {
			case "name", "resource_group_name", "location", "tags":
				continue // These are already handled in the main inputs block
			case "service_plan_id":
				inputs = append(inputs, "# App Service specific settings\nservice_plan_id = dependency.serviceplan.outputs.id")
			case "site_config":
				inputs = append(inputs, `# Required site configuration
site_config = try(local.env_config.locals.appservice.site_config, {
  application_stack = {
    dotnet_version = "7.0"
    node_version = "18-lts"
  }
  always_on = true
  minimum_tls_version = "1.2"
})`)
			default:
				inputs = append(inputs, fmt.Sprintf("%s = try(local.env_config.locals.%s.%s, null)",
					attr,
					strings.TrimPrefix(resourceType, "azurerm_"),
					attr))
			}
		}
		return strings.Join(inputs, "\n\n")
	}

	// Fallback to hardcoded defaults if no schema available
	switch resourceType {
	case "azurerm_linux_web_app", "azurerm_windows_web_app":
		return `# App Service specific settings
service_plan_id = dependency.serviceplan.outputs.id

# Required site configuration
site_config = try(local.env_config.locals.appservice.site_config, {
  application_stack = {
    dotnet_version = "7.0"
    node_version = "18-lts"
  }
  always_on = true
  minimum_tls_version = "1.2"
})`
	case "azurerm_service_plan":
		return `# Service Plan specific settings
sku_name = try(local.env_config.locals.service_plan.sku_name, null)
os_type = try(local.env_config.locals.service_plan.os_type, null)`
	case "azurerm_redis_cache":
		return `# Redis Cache specific settings
sku_name = try(local.env_config.locals.rediscache.sku_name, "Standard")
enable_non_ssl_port = try(local.env_config.locals.rediscache.enable_non_ssl_port, false)
minimum_tls_version = try(local.env_config.locals.rediscache.minimum_tls_version, "1.2")
redis_version = try(local.env_config.locals.rediscache.redis_version, "6")`
	case "azurerm_key_vault":
		return `# Key Vault specific settings
sku_name = try(local.env_config.locals.keyvault.sku_name, "standard")
tenant_id = data.azurerm_client_config.current.tenant_id
object_id = data.azurerm_client_config.current.object_id
access_policy = try(local.env_config.locals.keyvault.access_policy, [])
enabled_for_disk_encryption = try(local.env_config.locals.keyvault.enabled_for_disk_encryption, true)
enabled_for_deployment = try(local.env_config.locals.keyvault.enabled_for_deployment, true)
enabled_for_template_deployment = try(local.env_config.locals.keyvault.enabled_for_template_deployment, true)
purge_protection_enabled = try(local.env_config.locals.keyvault.purge_protection_enabled, false)
soft_delete_retention_days = try(local.env_config.locals.keyvault.soft_delete_retention_days, 7)`
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
