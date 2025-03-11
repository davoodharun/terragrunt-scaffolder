package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"gopkg.in/yaml.v3"
)

type TerraformProvider struct {
	Name    string `yaml:"provider"`
	Version string `yaml:"version"`
	Source  string `yaml:"source"`
}

type SchemaAttribute struct {
	Type        interface{} `json:"type"`
	Required    bool        `json:"required"`
	Optional    bool        `json:"optional"`
	Computed    bool        `json:"computed"`
	Description string      `json:"description"`
}

type ProviderSchema struct {
	ProviderSchema map[string]struct {
		ResourceSchemas map[string]struct {
			Block struct {
				Attributes map[string]SchemaAttribute `json:"attributes"`
				BlockTypes map[string]struct {
					Block struct {
						Attributes map[string]SchemaAttribute `json:"attributes"`
					} `json:"block"`
					NestingMode string `json:"nesting_mode"`
				} `json:"block_types"`
			} `json:"block"`
		} `json:"resource_schemas"`
	} `json:"provider_schemas"`
}

type SchemaCache struct {
	CachePath string
	Schema    *ProviderSchema
}

var schemaCache *SchemaCache

func initSchemaCache() (*SchemaCache, error) {
	if schemaCache != nil {
		return schemaCache, nil
	}

	// Create a temporary directory for terraform schema cache
	tmpDir, err := os.MkdirTemp("", "tf-schema-cache")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	schemaCache = &SchemaCache{
		CachePath: tmpDir,
	}
	return schemaCache, nil
}

func Generate() error {
	logger.Section("Starting Scaffold Generation")

	// Defer cleanup of schema cache
	defer cleanupSchemaCache()

	logger.Info("Reading configuration files")
	tgsConfig, err := readTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read tgs config: %w", err)
	}

	mainConfig, err := readMainConfig()
	if err != nil {
		return fmt.Errorf("failed to read main config: %w", err)
	}

	logger.Info("Creating base directories")
	if err := os.MkdirAll(".infrastructure", 0755); err != nil {
		return fmt.Errorf("failed to create .infrastructure directory: %w", err)
	}
	if err := os.MkdirAll(".infrastructure/_components", 0755); err != nil {
		return fmt.Errorf("failed to create _components directory: %w", err)
	}

	// Generate root.hcl
	if err := generateRootHCL(tgsConfig); err != nil {
		return fmt.Errorf("failed to generate root.hcl: %w", err)
	}

	// Generate environment config files
	if err := generateEnvironmentConfigs(tgsConfig); err != nil {
		return fmt.Errorf("failed to generate environment config files: %w", err)
	}

	// Generate components
	if err := generateComponentsWithEnvConfig(mainConfig); err != nil {
		return fmt.Errorf("failed to generate component templates: %w", err)
	}

	logger.Section("Generating Infrastructure")
	// Generate subscription structure
	for subName, sub := range tgsConfig.Subscriptions {
		logger.Info("Processing subscription: %s", subName)
		subPath := filepath.Join(".infrastructure", subName)
		if err := os.MkdirAll(subPath, 0755); err != nil {
			return fmt.Errorf("failed to create subscription directory: %w", err)
		}

		// Create subscription-level config with remote state info
		if err := createSubscriptionConfig(subPath, subName, sub); err != nil {
			return err
		}

		for region := range mainConfig.Stack.Architecture.Regions {
			regionPath := filepath.Join(subPath, region)
			if err := os.MkdirAll(regionPath, 0755); err != nil {
				return fmt.Errorf("failed to create region directory: %w", err)
			}

			// Create region-level config
			if err := createRegionConfig(regionPath, region); err != nil {
				return err
			}

			for _, env := range sub.Environments {
				if err := generateEnvironment(subName, region, env, mainConfig); err != nil {
					return fmt.Errorf("failed to generate environment %s/%s/%s: %w", subName, region, env.Name, err)
				}
			}
		}
	}

	logger.Success("Scaffold generation completed successfully")
	return nil
}

func createSubscriptionConfig(subPath, subName string, sub config.Subscription) error {
	logger.Info("Creating subscription config for %s", subName)
	subscriptionHCL := fmt.Sprintf(`locals {
  subscription_name = "%s"
  remote_state_resource_group = "%s"
  remote_state_storage_account = "%s"
}`, subName, sub.RemoteState.ResourceGroup, sub.RemoteState.Name)

	return createFile(filepath.Join(subPath, "subscription.hcl"), subscriptionHCL)
}

func createRegionConfig(regionPath, region string) error {
	regionConfig := fmt.Sprintf(`locals {
  region_name = "%s"
  region_path = "${get_parent_terragrunt_dir()}"
}`, region)

	return createFile(filepath.Join(regionPath, "region.hcl"), regionConfig)
}

func generateDependencyBlocks(deps []string) string {
	if len(deps) == 0 {
		return ""
	}

	var blocks []string
	for _, dep := range deps {
		// Parse the dependency string (e.g., "eastus2.redis" or "{region}.servicebus" or "eastus2.cosmos_db.{app}")
		parts := strings.Split(dep, ".")
		region := parts[0]
		componentName := parts[1]

		// Check if this is an app-specific dependency
		hasAppSuffix := len(parts) > 2 && parts[2] == "{app}"

		if hasAppSuffix {
			block := fmt.Sprintf(`
dependency "%s" {
  config_path = "${get_repo_root()}/.infrastructure/${local.subscription_name}/%s/${local.environment_name}/%s/${local.app_name}"
}`, componentName, resolveDependencyRegion(region), componentName)
			blocks = append(blocks, block)
		} else {
			block := fmt.Sprintf(`
dependency "%s" {
  config_path = "${get_repo_root()}/.infrastructure/${local.subscription_name}/%s/${local.environment_name}/%s"
}`, componentName, resolveDependencyRegion(region), componentName)
			blocks = append(blocks, block)
		}
	}

	return strings.Join(blocks, "\n")
}

func resolveDependencyRegion(region string) string {
	if region == "{region}" {
		return "${local.region_name}"
	}
	return region
}

func generateMainTF(comp config.Component, schema *ProviderSchema) string {
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

	// Try different provider keys
	providerKeys := []string{
		"registry.terraform.io/hashicorp/azurerm",
		"hashicorp/azurerm",
	}

	var found bool
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
		fmt.Printf("Warning: Schema not found for resource %s\n", comp.Source)
		return fmt.Sprintf(`
resource "%s" "this" {
  name                = var.name
  resource_group_name = var.resource_group_name
  location            = var.location

  tags = var.tags
}`, comp.Source)
	}

	var requiredAttributes []string
	var optionalAttributes []string
	var blocks []string

	// Add our common required fields first
	commonFields := []string{
		"  name                = var.name",
		"  resource_group_name = var.resource_group_name",
		"  location            = var.location",
		"  tags                = var.tags",
	}
	requiredAttributes = append(requiredAttributes, commonFields...)

	// Generate attribute assignments - separate required and optional
	for name, attr := range resourceSchema.Block.Attributes {
		if shouldSkipVariable(name) {
			continue
		}

		if attr.Required {
			requiredAttributes = append(requiredAttributes, fmt.Sprintf("  %s = var.%s", name, name))
		} else if attr.Optional && !attr.Computed {
			// Only include purely optional fields (not computed) as comments
			optionalAttributes = append(optionalAttributes, fmt.Sprintf("  # %s = var.%s", name, name))
		}
	}

	// Generate dynamic blocks - separate required and optional
	for blockName, blockType := range resourceSchema.Block.BlockTypes {
		var requiredBlockAttrs []string
		var optionalBlockAttrs []string

		for attrName, attr := range blockType.Block.Attributes {
			if attr.Required {
				requiredBlockAttrs = append(requiredBlockAttrs, fmt.Sprintf("      %s = %s.value.%s", attrName, blockName, attrName))
			} else if attr.Optional && !attr.Computed {
				optionalBlockAttrs = append(optionalBlockAttrs, fmt.Sprintf("      # %s = %s.value.%s", attrName, blockName, attrName))
			}
		}

		if len(requiredBlockAttrs) > 0 || len(optionalBlockAttrs) > 0 {
			block := fmt.Sprintf(`
  dynamic "%s" {
    for_each = var.%s
    content {
%s
%s
    }
  }`, blockName, blockName,
				strings.Join(requiredBlockAttrs, "\n"),
				strings.Join(optionalBlockAttrs, "\n"))
			blocks = append(blocks, block)
		}
	}

	// Combine all attributes with optional ones as comments
	allAttributes := append(requiredAttributes, optionalAttributes...)

	return fmt.Sprintf(`
resource "%s" "this" {
%s

%s

  lifecycle {
    ignore_changes = [
      tags["CreatedDate"],
      tags["Environment"]
    ]
  }
}`, comp.Source, strings.Join(allAttributes, "\n"), strings.Join(blocks, "\n"))
}

func shouldSkipVariable(name string) bool {
	// Common variables we define ourselves
	commonVars := map[string]bool{
		"name":                true,
		"resource_group_name": true,
		"location":            true,
		"tags":                true,
	}

	// Common computed fields that should not be inputs
	computedFields := map[string]bool{
		"id":                                    true,
		"principal_id":                          true,
		"tenant_id":                             true,
		"object_id":                             true,
		"type":                                  true,
		"identity":                              true,
		"system_assigned_identity":              true,
		"system_assigned_principal_id":          true,
		"system_assigned_identity_principal_id": true,
	}

	return commonVars[name] || computedFields[name]
}

func generateVariablesTF(schema *ProviderSchema, comp config.Component) string {
	// Common variables that most Azure resources need
	variables := []string{`
variable "name" {
  type        = string
  description = "The name of the resource"
}

variable "resource_group_name" {
  type        = string
  description = "The name of the resource group"
}

variable "location" {
  type        = string
  description = "The location/region of the resource"
}

variable "tags" {
  type        = map(string)
  description = "Tags to apply to the resource"
  default     = {}
}

variable "subscription_name" {
  type        = string
  description = "The name of the subscription"
}

variable "region_name" {
  type        = string
  description = "The name of the region"
}

variable "environment_name" {
  type        = string
  description = "The name of the environment"
}

variable "app_name" {
  type        = string
  description = "The name of the application"
  default     = ""
}`}

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

	var found bool
	for _, key := range providerKeys {
		if provider, ok := schema.ProviderSchema[key]; ok {
			if rs, ok := provider.ResourceSchemas[comp.Source]; ok {
				resourceSchema = rs
				found = true
				break
			}
		}
	}

	if found {
		// Add resource-specific variables based on schema
		for name, attr := range resourceSchema.Block.Attributes {
			// Skip common variables and computed fields
			if shouldSkipVariable(name) {
				continue
			}

			// Skip computed-only fields
			if attr.Computed && !attr.Required && !attr.Optional {
				continue
			}

			// Generate smart defaults based on attribute name and type
			defaultValue := generateSmartDefault(name, attr)

			varBlock := fmt.Sprintf(`
variable "%s" {
  type        = %s
  description = "%s"
  %s
}`, name,
				convertType(attr.Type),
				sanitizeDescription(attr.Description),
				defaultValue)
			variables = append(variables, varBlock)
		}

		// Handle nested blocks
		for blockName, blockType := range resourceSchema.Block.BlockTypes {
			variables = append(variables, generateNestedBlockVariable(blockName, blockType))
		}
	}

	return strings.Join(variables, "\n")
}

func generateSmartDefault(name string, attr SchemaAttribute) string {
	if attr.Computed && !attr.Required && !attr.Optional {
		return "" // No default for computed-only fields
	}

	if !attr.Required && !attr.Optional {
		return ""
	}

	switch v := attr.Type.(type) {
	case string:
		switch v {
		case "string":
			// Common naming patterns
			if strings.Contains(name, "sku") {
				return `default = "Standard"`
			}
			if strings.Contains(name, "tier") {
				return `default = "Standard"`
			}
			if strings.Contains(name, "version") {
				return `default = "latest"`
			}
			if strings.Contains(name, "kind") {
				return `default = ""`
			}
			if strings.Contains(name, "enabled") {
				return `default = true`
			}
			return `default = ""`
		case "number":
			if strings.Contains(name, "capacity") {
				return "default = 1"
			}
			if strings.Contains(name, "count") {
				return "default = 1"
			}
			return "default = 0"
		case "bool":
			if strings.Contains(name, "enabled") || strings.Contains(name, "enable") {
				return "default = true"
			}
			return "default = false"
		case "list":
			return "default = []"
		case "map":
			return "default = {}"
		}
	case []interface{}:
		if len(v) > 0 {
			if typeStr, ok := v[0].(string); ok {
				return generateSmartDefault(name, SchemaAttribute{
					Type:        typeStr,
					Required:    attr.Required,
					Optional:    attr.Optional,
					Computed:    attr.Computed,
					Description: attr.Description,
				})
			}
		}
	}
	return ""
}

func convertType(tfType interface{}) string {
	switch v := tfType.(type) {
	case string:
		switch v {
		case "string":
			return "string"
		case "number":
			return "number"
		case "bool":
			return "bool"
		case "list":
			return "list(any)"
		case "map":
			return "map(any)"
		default:
			return "any"
		}
	case []interface{}:
		if len(v) > 0 {
			if typeStr, ok := v[0].(string); ok {
				return convertType(typeStr)
			}
		}
		return "any"
	default:
		return "any"
	}
}

func generateProviderTF(comp config.Component) string {
	return fmt.Sprintf(`terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "%s"
    }
  }
}

provider "azurerm" {
  	features {}
	resource_provider_registrations = "none"
}

data "azurerm_client_config" "current" {}
`, comp.Version)
}

func sanitizeDescription(desc string) string {
	// Remove any special characters that might break the HCL
	return strings.ReplaceAll(desc, `"`, `\"`)
}

func generateNestedBlockVariable(blockName string, blockType struct {
	Block struct {
		Attributes map[string]SchemaAttribute `json:"attributes"`
	} `json:"block"`
	NestingMode string `json:"nesting_mode"`
}) string {
	var attrs []string
	for attrName, attr := range blockType.Block.Attributes {
		if attr.Required || attr.Optional {
			attrs = append(attrs, fmt.Sprintf("      %s = optional(%s)", attrName, convertType(attr.Type)))
		}
	}

	return fmt.Sprintf(`
variable "%s" {
  type = list(object({
%s
  }))
  description = "%s configuration block"
  default     = []
}`, blockName, strings.Join(attrs, "\n"), blockName)
}

func cleanupSchemaCache() {
	if schemaCache != nil {
		// Clean up .terraform directory
		tfDir := filepath.Join(schemaCache.CachePath, ".terraform")
		if err := os.RemoveAll(tfDir); err != nil {
			fmt.Printf("Warning: failed to remove .terraform directory: %v\n", err)
		}
		// Clean up cache directory
		if err := os.RemoveAll(schemaCache.CachePath); err != nil {
			fmt.Printf("Warning: failed to remove cache directory: %v\n", err)
		}
	}
}

func readTGSConfig() (*config.TGSConfig, error) {
	// Get the executable's directory
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Read from the tgs directory
	data, err := os.ReadFile(filepath.Join(execDir, "tgs.yaml"))
	if err != nil {
		// Try current directory as fallback
		data, err = os.ReadFile("tgs.yaml")
		if err != nil {
			return nil, err
		}
	}

	var cfg config.TGSConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func readMainConfig() (*config.MainConfig, error) {
	// Get the executable's directory
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Read from the tgs directory
	data, err := os.ReadFile(filepath.Join(execDir, "main.yaml"))
	if err != nil {
		// Try current directory as fallback
		data, err = os.ReadFile("main.yaml")
		if err != nil {
			return nil, err
		}
	}

	var cfg config.MainConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal main.yaml: %w", err)
	}

	return &cfg, nil
}

func createFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func generateRootHCL(tgsConfig *config.TGSConfig) error {
	logger.Info("Generating root.hcl configuration")
	rootHCL := `# Include this in all terragrunt.hcl files
locals {
  subscription_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl"))
  subscription_name = local.subscription_vars.locals.subscription_name
  remote_state_resource_group = local.subscription_vars.locals.remote_state_resource_group
  remote_state_storage_account = local.subscription_vars.locals.remote_state_storage_account
}

remote_state {
  backend = "azurerm"
  config = {
    resource_group_name  = local.remote_state_resource_group
    storage_account_name = local.remote_state_storage_account
    container_name       = local.subscription_name
    key                  = "${path_relative_to_include()}/terraform.tfstate"
  }
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }
}
`

	return createFile(filepath.Join(".infrastructure", "root.hcl"), rootHCL)
}

// Add this function to generate environment config files
func generateEnvironmentConfigs(tgsConfig *config.TGSConfig) error {
	logger.Info("Generating environment configuration files")

	// Create config directory
	configDir := filepath.Join(".infrastructure", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate a config file for each environment in each subscription
	for _, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			envName := env.Name

			// Create environment config file with default values
			configContent := fmt.Sprintf(`# Configuration for %s environment
# Override these values as needed for your environment

locals {
  # Resource group naming convention
  resource_group_prefix = "rg"
  
  # Common settings
  resource_group_name = "${local.resource_group_prefix}-${local.app_name}-${local.environment_name}"
  
  # Default SKUs and tiers for various services
  app_service_sku = {
    name     = "%s"
    tier     = "Standard"
    size     = "S1"
    capacity = 1
  }
  
  redis_cache_sku = {
    name     = "Standard"
    family   = "C"
    capacity = 1
  }
  
  cosmos_db_settings = {
    offer_type       = "Standard"
    consistency_level = "Session"
    max_throughput   = 1000
  }
  
  servicebus_sku = "Standard"
  
  # Add more environment-specific settings as needed
}`, getDefaultSkuForEnvironment(envName))

			configPath := filepath.Join(configDir, fmt.Sprintf("%s.hcl", envName))
			if err := createFile(configPath, configContent); err != nil {
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

// Update the component.hcl template to use environment config files
func generateComponentsWithEnvConfig(mainConfig *config.MainConfig) error {
	logger.Info("Generating components")

	// Create components directory
	componentsDir := filepath.Join(".infrastructure", "_components")
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
  
  # Load environment-specific configuration
  env_config = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/${local.environment_vars.locals.environment_name}.hcl")

  subscription_name = local.subscription_vars.locals.subscription_name
  region_name = local.region_vars.locals.region_name
  environment_name = local.environment_vars.locals.environment_name
  
  # Get the directory name as the app name, defaulting to empty string if at component root
  app_name = try(basename(dirname(get_terragrunt_dir())), basename(get_terragrunt_dir()), "")
}

terraform {
  source = "${get_repo_root()}/.infrastructure/_components/%s"
}

%s

inputs = {
  subscription_name = local.subscription_name
  region_name = local.region_name
  environment_name = local.environment_vars.locals.environment_name
  app_name = local.app_name
  name = coalesce(try("${local.app_name}-${local.environment_name}", ""), local.environment_name)
  
  # Use environment-specific resource group name from config
  resource_group_name = try(local.env_config.locals.resource_group_name, "rg-${local.app_name}-${local.environment_name}")
  
  location = local.region_name
  tags = {
    Environment = local.environment_name
    Application = local.app_name
  }
  
  # Include environment-specific configurations based on component type
  %s
}`, compName, generateDependencyBlocks(comp.Deps), generateEnvConfigInputs(compName))

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

// Helper function to generate environment-specific inputs based on component type
func generateEnvConfigInputs(compName string) string {
	switch compName {
	case "appservice":
		return `# App Service specific settings
sku_name = try(local.env_config.locals.app_service_sku.name, "B1")
sku_tier = try(local.env_config.locals.app_service_sku.tier, "Basic")
sku_size = try(local.env_config.locals.app_service_sku.size, "B1")
sku_capacity = try(local.env_config.locals.app_service_sku.capacity, 1)`
	case "serviceplan":
		return `# Service Plan specific settings
sku_name = try(local.env_config.locals.app_service_sku.name, "B1")
sku_tier = try(local.env_config.locals.app_service_sku.tier, "Basic")
sku_size = try(local.env_config.locals.app_service_sku.size, "B1")
sku_capacity = try(local.env_config.locals.app_service_sku.capacity, 1)`
	case "rediscache":
		return `# Redis Cache specific settings
sku_name = try(local.env_config.locals.redis_cache_sku.name, "Basic")
family = try(local.env_config.locals.redis_cache_sku.family, "C")
capacity = try(local.env_config.locals.redis_cache_sku.capacity, 0)`
	case "cosmos_account", "cosmos_db":
		return `# Cosmos DB specific settings
offer_type = try(local.env_config.locals.cosmos_db_settings.offer_type, "Standard")
consistency_level = try(local.env_config.locals.cosmos_db_settings.consistency_level, "Session")
max_throughput = try(local.env_config.locals.cosmos_db_settings.max_throughput, 1000)`
	case "servicebus":
		return `# Service Bus specific settings
sku = try(local.env_config.locals.servicebus_sku, "Standard")`
	default:
		return "# No component-specific settings"
	}
}
