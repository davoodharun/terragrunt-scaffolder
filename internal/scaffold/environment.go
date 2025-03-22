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

	// Get the subscription configuration to access remote state details
	subConfig, err := ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	sub, exists := subConfig.Subscriptions[subscription]
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

				// Create app-specific terragrunt.hcl
				terragruntContent := fmt.Sprintf(`include "root" {
  path = find_in_parent_folders("root.hcl")
}

include "component" {
  path = "${get_repo_root()}/.infrastructure/_components/%s/component.hcl"
}

locals {
  app_name = "%s"
}`, comp.Component, app)

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
  path = "${get_repo_root()}/.infrastructure/_components/%s/component.hcl"
}`, comp.Component)

			if err := createFile(filepath.Join(compPath, "terragrunt.hcl"), terragruntContent); err != nil {
				return fmt.Errorf("failed to create terragrunt.hcl for component: %w", err)
			}
		}
	}

	return nil
}

func generateEnvironmentConfigs(tgsConfig *config.TGSConfig, infraPath string) error {
	logger.Info("Generating environment configuration files")

	// Ensure the .infrastructure directory exists first
	baseDir := filepath.Base(infraPath)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create infrastructure directory: %w", err)
	}

	// Create config directory
	configDir := filepath.Join(baseDir, "config")
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

	logger.Info("Generated global config file: %s", globalPath)

	// Generate a config file for each environment in each subscription
	for _, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			envName := env.Name

			// Use the stack specified in the environment config, default to "main" if not specified
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}

			// Read the stack configuration to get actual components
			mainConfig, err := readMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			// Build environment config content with only the components that exist in the stack
			var configContent strings.Builder
			configContent.WriteString(fmt.Sprintf("# Configuration for %s environment\n", envName))
			configContent.WriteString("# Override these values as needed for your environment\n\n")
			configContent.WriteString("locals {\n")

			// Add configurations only for components that exist in the stack
			for compName := range mainConfig.Stack.Components {
				switch compName {
				case "serviceplan":
					configContent.WriteString(`  # Service Plan Configuration
  service_plan = {
    sku_name = "` + getDefaultSkuForEnvironment(envName) + `"
    os_type = "Linux"
    worker_count = 1
  }

`)
				case "appservice":
					configContent.WriteString(`  # App Service Configuration
  appservice = {
    https_only = true
    site_config = {
      always_on = true
      application_stack = {
        dotnet_version = "6.0"
      }
      use_32_bit_worker = false
      websockets_enabled = false
    }
    app_settings = {
      WEBSITES_ENABLE_APP_SERVICE_STORAGE = false
      WEBSITE_RUN_FROM_PACKAGE = 1
    }
  }

`)
				case "functionapp":
					configContent.WriteString(`  # Function App Configuration
  functionapp = {
    https_only = true
    site_config = {
      always_on = true
      application_stack = {
        node_version = "16"
      }
    }
    app_settings = {
      FUNCTIONS_WORKER_RUNTIME = "node"
      WEBSITE_NODE_DEFAULT_VERSION = "~16"
    }
  }

`)
				case "rediscache":
					configContent.WriteString(`  # Redis Cache Configuration
  rediscache = {
    sku = {
      name     = "Basic"
      family   = "C"
      capacity = 1
    }
    enable_non_ssl_port = false
    minimum_tls_version = "1.2"
  }

`)
				case "keyvault":
					configContent.WriteString(`  # Key Vault Configuration
  keyvault = {
    sku_name = "standard"
    enabled_for_disk_encryption = true
    enabled_for_deployment = true
    enabled_for_template_deployment = true
    purge_protection_enabled = true
  }

`)
				}
			}

			configContent.WriteString("}")

			// Create environment config file
			configPath := filepath.Join(configDir, fmt.Sprintf("%s.hcl", envName))
			if err := createFile(configPath, configContent.String()); err != nil {
				return fmt.Errorf("failed to create environment config file: %w", err)
			}

			logger.Info("Generated environment config file: %s", configPath)
		}
	}

	return nil
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
