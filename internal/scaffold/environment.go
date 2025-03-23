package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

func generateEnvironment(subscription, region string, envName string, components []config.RegionComponent, infraPath string) error {
	// Get the stack name from the environment
	stackName := "main"
	tgsConfig, err := ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Find the stack name for this environment
	if sub, ok := tgsConfig.Subscriptions[subscription]; ok {
		for _, env := range sub.Environments {
			if env.Name == envName {
				if env.Stack != "" {
					stackName = env.Stack
				}
				break
			}
		}
	}

	// Create environment base path
	basePath := filepath.Join(infraPath, subscription, region, envName)
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create environment directory: %w", err)
	}

	// Create environment.hcl
	envHclContent := fmt.Sprintf(`locals {
  environment_name = "%s"
  environment_prefix = "%s"
}`, envName, getEnvironmentPrefix(envName))

	if err := createFile(filepath.Join(basePath, "environment.hcl"), envHclContent); err != nil {
		return fmt.Errorf("failed to create environment.hcl: %w", err)
	}

	// Create region.hcl in the region directory
	regionPath := filepath.Join(infraPath, subscription, region)
	if err := os.MkdirAll(regionPath, 0755); err != nil {
		return fmt.Errorf("failed to create region directory: %w", err)
	}

	regionHclContent := fmt.Sprintf(`locals {
  region_name = "%s"
  region_prefix = "%s"
}`, region, getRegionPrefix(region))

	if err := createFile(filepath.Join(regionPath, "region.hcl"), regionHclContent); err != nil {
		return fmt.Errorf("failed to create region.hcl: %w", err)
	}

	// Create subscription.hcl in the subscription directory
	subPath := filepath.Join(infraPath, subscription)
	if err := os.MkdirAll(subPath, 0755); err != nil {
		return fmt.Errorf("failed to create subscription directory: %w", err)
	}

	sub, exists := tgsConfig.Subscriptions[subscription]
	if !exists {
		return fmt.Errorf("subscription %s not found in TGS config", subscription)
	}

	subHclContent := fmt.Sprintf(`locals {
  subscription_name = "%s"
  remote_state_resource_group = "%s"
  remote_state_storage_account = "%s"
}`, subscription, sub.RemoteState.ResourceGroup, sub.RemoteState.Name)

	if err := createFile(filepath.Join(subPath, "subscription.hcl"), subHclContent); err != nil {
		return fmt.Errorf("failed to create subscription.hcl: %w", err)
	}

	// Generate component directories and their apps
	for _, comp := range components {
		compPath := filepath.Join(basePath, comp.Component)
		if err := os.MkdirAll(compPath, 0755); err != nil {
			return fmt.Errorf("failed to create component directory: %w", err)
		}

		if len(comp.Apps) > 0 {
			// Create app-specific folders and terragrunt files
			for _, app := range comp.Apps {
				appPath := filepath.Join(compPath, app)
				if err := os.MkdirAll(appPath, 0755); err != nil {
					return fmt.Errorf("failed to create app directory: %w", err)
				}

				// Create app-specific terragrunt.hcl with stack name in the path
				terragruntContent := fmt.Sprintf(`include "root" {
  path = find_in_parent_folders("root.hcl")
}

include "component" {
  path = "${get_repo_root()}/.infrastructure/_components/%s/%s/component.hcl"
}

locals {
  app_name = "%s"
  component_vars = read_terragrunt_config("${get_repo_root()}/.infrastructure/_components/%s/%s/component.hcl")
  env_vars = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/${local.component_vars.locals.stack_name}/${local.environment_name}.hcl")
}`, stackName, comp.Component, app, stackName, comp.Component)

				if err := createFile(filepath.Join(appPath, "terragrunt.hcl"), terragruntContent); err != nil {
					return fmt.Errorf("failed to create terragrunt.hcl for app: %w", err)
				}
			}
		} else {
			// Create single terragrunt.hcl for components without apps
			terragruntContent := fmt.Sprintf(`include "root" {
  path = find_in_parent_folders("root.hcl")
}

include "component" {
  path = "${get_repo_root()}/.infrastructure/_components/%s/%s/component.hcl"
}

locals {
  component_vars = read_terragrunt_config("${get_repo_root()}/.infrastructure/_components/%s/%s/component.hcl")
  env_vars = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/${local.component_vars.locals.stack_name}/${local.environment_name}.hcl")
}`, stackName, comp.Component, stackName, comp.Component)

			if err := createFile(filepath.Join(compPath, "terragrunt.hcl"), terragruntContent); err != nil {
				return fmt.Errorf("failed to create terragrunt.hcl for component: %w", err)
			}
		}
	}

	return nil
}

func generateEnvironmentConfigs(tgsConfig *config.TGSConfig, infraPath string) error {
	// Create config directory
	configDir := filepath.Join(infraPath, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate global.hcl with the name property from tgs.yaml
	globalHCL := fmt.Sprintf(`# Global configuration values
locals {
  # Project name from tgs.yaml
  project_name = "%s"
  
  # Resource group configuration by environment and region
  resource_groups = {
    dev = {
      eastus2 = "rg-${local.project_name}-e2-d"
      westus2 = "rg-${local.project_name}-w2-d"
    }
    test = {
      eastus2 = "rg-${local.project_name}-e2-t"
      westus2 = "rg-${local.project_name}-w2-t"
    }
    stage = {
      eastus2 = "rg-${local.project_name}-e2-s"
      westus2 = "rg-${local.project_name}-w2-s"
    }
    prod = {
      eastus2 = "rg-${local.project_name}-e2-p"
      westus2 = "rg-${local.project_name}-w2-p"
    }
  }
  
  # Common tags for all resources
  common_tags = {
    Project = local.project_name
    ManagedBy = "Terragrunt"
  }
}`, tgsConfig.Name)

	globalPath := filepath.Join(configDir, "global.hcl")
	if err := createFile(globalPath, globalHCL); err != nil {
		return fmt.Errorf("failed to create global config file: %w", err)
	}

	logger.Success("Generated environment configuration files")

	// Track unique stacks to create their directories
	uniqueStacks := make(map[string]bool)

	// Generate a config file for each environment in each subscription
	for _, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			envName := env.Name

			// Use the stack specified in the environment config, default to "main" if not specified
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}

			// Track this stack
			uniqueStacks[stackName] = true

			// Create stack-specific config directory
			stackConfigDir := filepath.Join(configDir, stackName)
			if err := os.MkdirAll(stackConfigDir, 0755); err != nil {
				return fmt.Errorf("failed to create stack config directory: %w", err)
			}

			// Read the stack configuration to get actual components
			mainConfig, err := readMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			// Build environment config content with only the components that exist in the stack
			var configContent strings.Builder
			configContent.WriteString(fmt.Sprintf("# Configuration for %s environment in stack %s\n", envName, stackName))
			configContent.WriteString("# Override these values as needed for your environment\n\n")
			configContent.WriteString("locals {\n")

			// Add configurations only for components that exist in the stack
			for compName, comp := range mainConfig.Stack.Components {
				if comp.Provider == "" {
					continue
				}

				// Fetch provider schema for this component
				schema, err := fetchProviderSchema(comp.Provider, comp.Version, comp.Source)
				if err != nil || schema == nil {
					logger.Warning("Failed to fetch provider schema for %s: %v", compName, err)
					continue
				}

				// Try different provider keys
				providerKeys := []string{
					"registry.terraform.io/hashicorp/azurerm",
					"hashicorp/azurerm",
				}

				var resourceSchema struct {
					Block struct {
						Attributes map[string]SchemaAttribute `json:"attributes"`
						BlockTypes map[string]struct {
							Block struct {
								Attributes map[string]SchemaAttribute `json:"attributes"`
							} `json:"block"`
							NestingMode string `json:"nesting_mode"`
						} `json:"block_types"`
					} `json:"block"`
				}

				found := false
				for _, key := range providerKeys {
					if provider, ok := schema.ProviderSchema[key]; ok {
						if rs, ok := provider.ResourceSchemas[comp.Source]; ok {
							resourceSchema = rs
							found = true
							break
						}
					}
				}

				if !found {
					continue
				}

				// Start component configuration block
				configContent.WriteString(fmt.Sprintf("  # %s Configuration\n", compName))
				configContent.WriteString(fmt.Sprintf("  %s = {\n", compName))

				// Add required attributes with default values
				for name, attr := range resourceSchema.Block.Attributes {
					if attr.Required && !shouldSkipVariable(name, comp.Source) {
						defaultValue := getDefaultValueForType(attr.Type, name, envName)
						configContent.WriteString(fmt.Sprintf("    %s = %s\n", name, defaultValue))
					}
				}

				// Add block types (nested configurations)
				for blockName, blockType := range resourceSchema.Block.BlockTypes {
					configContent.WriteString(fmt.Sprintf("    %s = {\n", blockName))
					for attrName, attr := range blockType.Block.Attributes {
						if attr.Required {
							defaultValue := getDefaultValueForType(attr.Type, attrName, envName)
							configContent.WriteString(fmt.Sprintf("      %s = %s\n", attrName, defaultValue))
						}
					}
					configContent.WriteString("    }\n")
				}

				configContent.WriteString("  }\n\n")
			}

			configContent.WriteString("}")

			// Create environment config file in the stack-specific directory
			configPath := filepath.Join(stackConfigDir, fmt.Sprintf("%s.hcl", envName))
			if err := createFile(configPath, configContent.String()); err != nil {
				return fmt.Errorf("failed to create environment config file: %w", err)
			}

			logger.Info("Generated environment config file: %s", configPath)
		}
	}

	return nil
}

// Helper function to get default value based on type and environment
func getDefaultValueForType(attrType interface{}, name string, env string) string {
	switch t := attrType.(type) {
	case string:
		switch t {
		case "string":
			// Special cases for known attributes
			switch name {
			case "sku_name":
				if strings.Contains(env, "redis") || strings.Contains(env, "cache") {
					return fmt.Sprintf(`"%s"`, getDefaultRedisSkuForEnvironment(env))
				}
				return fmt.Sprintf(`"%s"`, getDefaultSkuForEnvironment(env))
			case "family":
				return `"C"`
			case "tier":
				return `"Standard"`
			case "os_type":
				return `"Linux"`
			default:
				return `""`
			}
		case "number":
			return "0"
		case "bool":
			return "false"
		case "list":
			return "[]"
		case "map":
			return "{}"
		default:
			return "null"
		}
	case map[string]interface{}:
		if t["type"] == "string" {
			return `""`
		}
		return "null"
	default:
		return "null"
	}
}

// Helper function to determine default SKU based on environment
func getDefaultSkuForEnvironment(env string) string {
	switch env {
	case "prod":
		return "P1v2"
	case "stage":
		return "P1v2"
	case "test":
		return "S1"
	case "dev":
		return "B1"
	default:
		return "B1"
	}
}

// Helper function to determine default Redis SKU based on environment
func getDefaultRedisSkuForEnvironment(env string) string {
	switch env {
	case "prod":
		return "Premium"
	case "stage":
		return "Standard"
	case "test":
		return "Standard"
	case "dev":
		return "Basic"
	default:
		return "Basic"
	}
}

func generateRootHCL(tgsConfig *config.TGSConfig, infraPath string) error {
	logger.Info("Generating root.hcl configuration")

	// Ensure the .infrastructure directory exists
	baseDir := filepath.Base(infraPath)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create infrastructure directory: %w", err)
	}

	rootHCL := `# Include this in all terragrunt.hcl files
locals {
  subscription_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl"))
  global_config = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/global.hcl")
  environment_vars = read_terragrunt_config(find_in_parent_folders("environment.hcl"))
  
  subscription_name = local.subscription_vars.locals.subscription_name
  project_name = local.global_config.locals.project_name
  remote_state_resource_group = local.subscription_vars.locals.remote_state_resource_group
  remote_state_storage_account = local.subscription_vars.locals.remote_state_storage_account
  
  # Infrastructure path relative to repo root
  infrastructure_path = ".infrastructure"
}

remote_state {
  backend = "azurerm"
  config = {
    resource_group_name  = local.remote_state_resource_group
    storage_account_name = local.remote_state_storage_account
    container_name       = local.project_name
    key                  = "${path_relative_to_include()}/terraform.tfstate"
  }
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }
}`

	return createFile(filepath.Join(baseDir, "root.hcl"), rootHCL)
}
