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
			getResourceTypeAbbreviation(compName), compName, generateDependencyBlocks(comp.Deps, infraPath), generateEnvConfigInputs(compName))

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
app_settings = try(local.env_config.locals.appservice.app_settings, {})
auth_settings = try(local.env_config.locals.appservice.auth_settings, {})
identity = try(local.env_config.locals.appservice.identity, {})
backup = try(local.env_config.locals.appservice.backup, {})
connection_string = try(local.env_config.locals.appservice.connection_string, [])
cors = try(local.env_config.locals.appservice.cors, {})
ip_restriction = try(local.env_config.locals.appservice.ip_restriction, [])
scm_ip_restriction = try(local.env_config.locals.appservice.scm_ip_restriction, [])
virtual_network_subnet_id = try(local.env_config.locals.appservice.virtual_network_subnet_id, null)
key_vault_reference_identity_id = try(local.env_config.locals.appservice.key_vault_reference_identity_id, null)
client_cert_enabled = try(local.env_config.locals.appservice.client_cert_enabled, false)
client_cert_mode = try(local.env_config.locals.appservice.client_cert_mode, "Required")`

	case "serviceplan":
		return `# Service Plan specific settings
sku_name = try(local.env_config.locals.serviceplan.sku_name, "B1")`

	case "functionapp":
		return `# Function App specific settings
service_plan_id = dependency.serviceplan.outputs.id

# Import all function app settings from environment config
app_settings = try(local.env_config.locals.functionapp.app_settings, {})
site_config = try(local.env_config.locals.functionapp.site_config, {})
auth_settings = try(local.env_config.locals.functionapp.auth_settings, {})
identity = try(local.env_config.locals.functionapp.identity, {})
backup = try(local.env_config.locals.functionapp.backup, {})
connection_string = try(local.env_config.locals.functionapp.connection_string, [])
cors = try(local.env_config.locals.functionapp.cors, {})
ip_restriction = try(local.env_config.locals.functionapp.ip_restriction, [])
scm_ip_restriction = try(local.env_config.locals.functionapp.scm_ip_restriction, [])
virtual_network_subnet_id = try(local.env_config.locals.functionapp.virtual_network_subnet_id, null)
key_vault_reference_identity_id = try(local.env_config.locals.functionapp.key_vault_reference_identity_id, null)
client_cert_enabled = try(local.env_config.locals.functionapp.client_cert_enabled, false)
client_cert_mode = try(local.env_config.locals.functionapp.client_cert_mode, "Required")`

	case "rediscache":
		return `# Redis Cache specific settings
sku_name = try(local.env_config.locals.rediscache.sku_name, "Basic")
capacity = try(local.env_config.locals.rediscache.capacity, 0)
family = try(local.env_config.locals.rediscache.family, "C")
enable_non_ssl_port = try(local.env_config.locals.rediscache.enable_non_ssl_port, false)
minimum_tls_version = try(local.env_config.locals.rediscache.minimum_tls_version, "1.2")
redis_version = try(local.env_config.locals.rediscache.redis_version, "6")`

	case "keyvault":
		return `# Key Vault specific settings
sku_name = try(local.env_config.locals.keyvault.sku_name, "standard")
enabled_for_disk_encryption = try(local.env_config.locals.keyvault.enabled_for_disk_encryption, true)
enabled_for_deployment = try(local.env_config.locals.keyvault.enabled_for_deployment, true)
enabled_for_template_deployment = try(local.env_config.locals.keyvault.enabled_for_template_deployment, true)
purge_protection_enabled = try(local.env_config.locals.keyvault.purge_protection_enabled, false)
soft_delete_retention_days = try(local.env_config.locals.keyvault.soft_delete_retention_days, 7)
network_acls = try(local.env_config.locals.keyvault.network_acls, {})
public_network_access_enabled = try(local.env_config.locals.keyvault.public_network_access_enabled, true)`

	case "servicebus":
		return `# Service Bus specific settings
sku = try(local.env_config.locals.servicebus.sku, "Standard")
capacity = try(local.env_config.locals.servicebus.capacity, 0)
zone_redundant = try(local.env_config.locals.servicebus.zone_redundant, false)`

	case "cosmos_account":
		return `# Cosmos DB Account specific settings
offer_type = try(local.env_config.locals.cosmos_account.offer_type, "Standard")
kind = try(local.env_config.locals.cosmos_account.kind, "GlobalDocumentDB")
consistency_level = try(local.env_config.locals.cosmos_account.consistency_level, "Session")
enable_automatic_failover = try(local.env_config.locals.cosmos_account.enable_automatic_failover, false)
enable_multiple_write_locations = try(local.env_config.locals.cosmos_account.enable_multiple_write_locations, false)
is_virtual_network_filter_enabled = try(local.env_config.locals.cosmos_account.is_virtual_network_filter_enabled, false)
virtual_network_rules = try(local.env_config.locals.cosmos_account.virtual_network_rules, [])`

	case "cosmos_db":
		return `# Cosmos DB specific settings
resource_group_name = dependency.cosmos_account.outputs.resource_group_name
account_name = dependency.cosmos_account.outputs.name
throughput = try(local.env_config.locals.cosmos_db.throughput, 400)`

	case "apim":
		return `# API Management specific settings
sku_name = try(local.env_config.locals.apim.sku_name, "Developer_1")
identity = try(local.env_config.locals.apim.identity, {})
protocols = try(local.env_config.locals.apim.protocols, ["http", "https"])
certificate = try(local.env_config.locals.apim.certificate, [])
security = try(local.env_config.locals.apim.security, {})
hostname_configuration = try(local.env_config.locals.apim.hostname_configuration, [])
virtual_network_type = try(local.env_config.locals.apim.virtual_network_type, "None")
virtual_network_configuration = try(local.env_config.locals.apim.virtual_network_configuration, [])`

	case "storage":
		return `# Storage Account specific settings
account_tier = try(local.env_config.locals.storage.account_tier, "Standard")
account_replication_type = try(local.env_config.locals.storage.account_replication_type, "LRS")
enable_https_traffic_only = try(local.env_config.locals.storage.enable_https_traffic_only, true)
min_tls_version = try(local.env_config.locals.storage.min_tls_version, "TLS1_2")
allow_nested_items_to_be_public = try(local.env_config.locals.storage.allow_nested_items_to_be_public, false)
shared_access_key_enabled = try(local.env_config.locals.storage.shared_access_key_enabled, true)
network_rules = try(local.env_config.locals.storage.network_rules, {})
blob_properties = try(local.env_config.locals.storage.blob_properties, {})
queue_properties = try(local.env_config.locals.storage.queue_properties, {})
static_website = try(local.env_config.locals.storage.static_website, {})`

	case "sql_server":
		return `# SQL Server specific settings
version = try(local.env_config.locals.sql_server.version, "12.0")
administrator_login = try(local.env_config.locals.sql_server.administrator_login, "sqladmin")
administrator_login_password = try(local.env_config.locals.sql_server.administrator_login_password, null)
minimum_tls_version = try(local.env_config.locals.sql_server.minimum_tls_version, "1.2")
public_network_access_enabled = try(local.env_config.locals.sql_server.public_network_access_enabled, true)
identity = try(local.env_config.locals.sql_server.identity, {})
extended_auditing_policy = try(local.env_config.locals.sql_server.extended_auditing_policy, [])
threat_detection_policy = try(local.env_config.locals.sql_server.threat_detection_policy, [])
azuread_administrator = try(local.env_config.locals.sql_server.azuread_administrator, [])`

	case "sql_database":
		return `# SQL Database specific settings
server_id = dependency.sql_server.outputs.id
sku_name = try(local.env_config.locals.sql_database.sku_name, "Basic")
max_size_gb = try(local.env_config.locals.sql_database.max_size_gb, 5)
zone_redundant = try(local.env_config.locals.sql_database.zone_redundant, false)
read_replicas = try(local.env_config.locals.sql_database.read_replicas, 0)
read_scale = try(local.env_config.locals.sql_database.read_scale, false)
collation = try(local.env_config.locals.sql_database.collation, "SQL_Latin1_General_CP1_CI_AS")`

	case "eventhub":
		return `# Event Hub specific settings
namespace_name = dependency.eventhub_namespace.outputs.name
resource_group_name = dependency.eventhub_namespace.outputs.resource_group_name
partition_count = try(local.env_config.locals.eventhub.partition_count, 2)
message_retention = try(local.env_config.locals.eventhub.message_retention, 1)
capture_description = try(local.env_config.locals.eventhub.capture_description, [])`

	case "loganalytics":
		return `# Log Analytics Workspace specific settings
sku = try(local.env_config.locals.loganalytics.sku, "PerGB2018")
retention_in_days = try(local.env_config.locals.loganalytics.retention_in_days, 30)
daily_quota_gb = try(local.env_config.locals.loganalytics.daily_quota_gb, -1)
internet_ingestion_enabled = try(local.env_config.locals.loganalytics.internet_ingestion_enabled, true)
internet_query_enabled = try(local.env_config.locals.loganalytics.internet_query_enabled, true)
reservation_capacity_in_gb_per_day = try(local.env_config.locals.loganalytics.reservation_capacity_in_gb_per_day, -1)`

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
