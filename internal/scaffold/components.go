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

	// Generate each component
	for compName, comp := range mainConfig.Stack.Components {
		logger.Info("Generating component: %s", compName)

		// Create component directory
		componentPath := filepath.Join(componentsDir, compName)
		if err := os.MkdirAll(componentPath, 0755); err != nil {
			return err
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
}`, getResourceTypeAbbreviation(compName), compName, generateDependencyBlocks(comp.Deps, infraPath), generateEnvConfigInputs(compName))

		if err := createFile(filepath.Join(componentPath, "component.hcl"), componentHcl); err != nil {
			return err
		}
	}
	return nil
}

func generateComponentsWithEnvConfig(mainConfig *config.MainConfig, infraPath string) error {
	logger.Info("Generating components with environment config")

	// Create components directory
	baseDir := filepath.Base(infraPath)
	componentsDir := filepath.Join(baseDir, "_components")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create components directory: %w", err)
	}

	// Generate each component
	for compName, comp := range mainConfig.Stack.Components {
		logger.Info("Generating component: %s", compName)

		// Create component directory
		componentPath := filepath.Join(componentsDir, compName)
		if err := os.MkdirAll(componentPath, 0755); err != nil {
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
}`, getResourceTypeAbbreviation(compName), compName, generateDependencyBlocks(comp.Deps, infraPath), generateEnvConfigInputs(compName))

		if err := createFile(filepath.Join(componentPath, "component.hcl"), componentHcl); err != nil {
			return err
		}

		// Generate Terraform files
		if err := generateTerraformFiles(componentPath, comp); err != nil {
			return err
		}
	}

	return nil
}

// Helper function to get resource type abbreviation
func getResourceTypeAbbreviation(resourceType string) string {
	resourceAbbreviations := map[string]string{
		"serviceplan":    "svcpln",
		"appservice":     "appsvc",
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
	case "appservice":
		return `# App Service specific settings
service_plan_id = dependency.serviceplan.outputs.id

# Import all app service settings from environment config
https_only = try(local.env_config.locals.appservice.https_only, true)
site_config = try(local.env_config.locals.appservice.site_config, {})
app_settings = try(local.env_config.locals.appservice.app_settings, {})`

	case "serviceplan":
		return `# Service Plan specific settings
sku_name = try(local.env_config.locals.serviceplan.sku_name, "B1")`

	case "functionapp":
		return `# Function App specific settings
service_plan_id = dependency.serviceplan.outputs.id
https_only = try(local.env_config.locals.functionapp.https_only, true)
site_config = try(local.env_config.locals.functionapp.site_config, {})
app_settings = try(local.env_config.locals.functionapp.app_settings, {})`

	case "rediscache":
		return `# Redis Cache specific settings
sku_name = try(local.env_config.locals.rediscache.sku.name, "Basic")
family = try(local.env_config.locals.rediscache.sku.family, "C")
capacity = try(local.env_config.locals.rediscache.sku.capacity, 0)
enable_non_ssl_port = try(local.env_config.locals.rediscache.enable_non_ssl_port, false)
minimum_tls_version = try(local.env_config.locals.rediscache.minimum_tls_version, "1.2")`

	case "keyvault":
		return `# Key Vault specific settings
sku_name = try(local.env_config.locals.keyvault.sku_name, "standard")
enabled_for_disk_encryption = try(local.env_config.locals.keyvault.enabled_for_disk_encryption, true)
enabled_for_deployment = try(local.env_config.locals.keyvault.enabled_for_deployment, true)
enabled_for_template_deployment = try(local.env_config.locals.keyvault.enabled_for_template_deployment, true)
purge_protection_enabled = try(local.env_config.locals.keyvault.purge_protection_enabled, true)`

	case "servicebus":
		return `# Service Bus specific settings
sku = try(local.env_config.locals.servicebus.sku, "Standard")
capacity = try(local.env_config.locals.servicebus.capacity, 1)
zone_redundant = try(local.env_config.locals.servicebus.zone_redundant, false)`

	case "cosmos_account":
		return `# Cosmos DB Account specific settings
offer_type = try(local.env_config.locals.cosmos_account.offer_type, "Standard")
kind = try(local.env_config.locals.cosmos_account.kind, "GlobalDocumentDB")
consistency_level = try(local.env_config.locals.cosmos_account.consistency_level, "Session")
geo_location = try(local.env_config.locals.cosmos_account.geo_location, {})
capabilities = try(local.env_config.locals.cosmos_account.capabilities, [])`

	case "storage":
		return `# Storage Account specific settings
account_tier = try(local.env_config.locals.storage.account_tier, "Standard")
account_replication_type = try(local.env_config.locals.storage.account_replication_type, "LRS")
min_tls_version = try(local.env_config.locals.storage.min_tls_version, "TLS1_2")
allow_nested_items_to_be_public = try(local.env_config.locals.storage.allow_nested_items_to_be_public, false)`

	case "sql_server":
		return `# SQL Server specific settings
version = try(local.env_config.locals.sql_server.version, "12.0")
administrator_login = try(local.env_config.locals.sql_server.administrator_login, "sqladmin")
minimum_tls_version = try(local.env_config.locals.sql_server.minimum_tls_version, "1.2")`

	case "sql_database":
		return `# SQL Database specific settings
sku_name = try(local.env_config.locals.sql_database.sku.name, "Basic")
max_size_gb = try(local.env_config.locals.sql_database.max_size_gb, 2)
zone_redundant = try(local.env_config.locals.sql_database.zone_redundant, false)`

	case "eventhub":
		return `# Event Hub specific settings
sku = try(local.env_config.locals.eventhub.sku, "Standard")
capacity = try(local.env_config.locals.eventhub.capacity, 1)
partition_count = try(local.env_config.locals.eventhub.partition_count, 2)
message_retention = try(local.env_config.locals.eventhub.message_retention, 1)
zone_redundant = try(local.env_config.locals.eventhub.zone_redundant, false)`

	case "loganalytics":
		return `# Log Analytics specific settings
sku = try(local.env_config.locals.loganalytics.sku, "PerGB2018")
retention_in_days = try(local.env_config.locals.loganalytics.retention_in_days, 30)
daily_quota_gb = try(local.env_config.locals.loganalytics.daily_quota_gb, 1)`

	default:
		return "# No component-specific settings"
	}
}

// Update generateDependencyBlocks to use the fixed infrastructure path
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
