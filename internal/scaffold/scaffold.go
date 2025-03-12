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

// Add a function to find the Git repository root
func findGitRepoRoot() (string, error) {
	// Start with the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree looking for .git
	for {
		// Check if .git exists in the current directory
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, nil // Found the git repo root
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root of the filesystem without finding .git
			return "", fmt.Errorf("no .git directory found in any parent directory")
		}
		dir = parent
	}
}

// Update the function to get the infrastructure path
func getInfrastructurePath() string {
	// Always use .infrastructure at the repo root
	return ".infrastructure"
}

func Generate() error {
	logger.Info("Starting terragrunt scaffolding generation")

	// Get the infrastructure path
	infraPath := ".infrastructure"
	logger.Info("Using infrastructure path: %s", infraPath)

	// Create infrastructure directory
	if err := os.MkdirAll(infraPath, 0755); err != nil {
		return fmt.Errorf("failed to create infrastructure directory: %w", err)
	}

	// Read TGS config
	tgsConfig, err := ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Create base directory structure
	baseDir := filepath.Base(infraPath)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Generate root.hcl
	if err := generateRootHCL(tgsConfig, infraPath); err != nil {
		return fmt.Errorf("failed to generate root.hcl: %w", err)
	}

	// Generate environment config files
	if err := generateEnvironmentConfigs(tgsConfig, infraPath); err != nil {
		return fmt.Errorf("failed to generate environment config files: %w", err)
	}

	// Track processed stacks to avoid duplicate component generation
	processedStacks := make(map[string]bool)

	// Create subscription directories and configs
	for subName, sub := range tgsConfig.Subscriptions {
		subPath := filepath.Join(baseDir, subName)
		if err := os.MkdirAll(subPath, 0755); err != nil {
			return fmt.Errorf("failed to create subscription directory: %w", err)
		}

		// Create subscription.hcl
		if err := createSubscriptionConfig(subPath, subName, sub); err != nil {
			return fmt.Errorf("failed to create subscription config: %w", err)
		}

		// Process each environment with its specified stack
		for _, env := range sub.Environments {
			// Use the stack specified in the environment config, default to "main" if not specified
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}

			// Read the stack-specific config
			mainConfig, err := readMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			// Generate components for this stack if we haven't already
			if !processedStacks[stackName] {
				// Create components directory
				componentsDir := filepath.Join(baseDir, "_components")
				if err := os.MkdirAll(componentsDir, 0755); err != nil {
					return fmt.Errorf("failed to create components directory: %w", err)
				}

				// Generate components with environment config
				if err := generateComponentsWithEnvConfig(mainConfig, infraPath); err != nil {
					return fmt.Errorf("failed to generate components for stack %s: %w", stackName, err)
				}

				processedStacks[stackName] = true
			}

			// Create region directories and their contents
			for region, _ := range mainConfig.Stack.Architecture.Regions {
				regionPath := filepath.Join(subPath, region)
				if err := os.MkdirAll(regionPath, 0755); err != nil {
					return fmt.Errorf("failed to create region directory: %w", err)
				}

				// Create region.hcl
				if err := createRegionConfig(regionPath, region); err != nil {
					return fmt.Errorf("failed to create region config: %w", err)
				}

				// Create environment directory and its contents
				if err := generateEnvironment(subName, region, env, mainConfig); err != nil {
					return fmt.Errorf("failed to generate environment: %w", err)
				}
			}
		}
	}

	logger.Info("Terragrunt scaffolding generation complete")
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
	logger.Info("Creating region config for %s", region)

	// Determine region prefix (single letter)
	regionPrefix := getRegionPrefix(region)

	regionHCL := fmt.Sprintf(`locals {
  region_name = "%s"
  region_prefix = "%s"
}`, region, regionPrefix)

	return createFile(filepath.Join(regionPath, "region.hcl"), regionHCL)
}

// Helper function to get a single letter prefix for a region
func getRegionPrefix(region string) string {
	regionPrefixMap := map[string]string{
		"eastus":        "E",
		"eastus2":       "E2",
		"westus":        "W",
		"westus2":       "W2",
		"centralus":     "C",
		"northeurope":   "NE",
		"westeurope":    "WE",
		"uksouth":       "UKS",
		"ukwest":        "UKW",
		"southeastasia": "SEA",
		"eastasia":      "EA",
	}

	// Check if we have a predefined prefix
	if prefix, ok := regionPrefixMap[region]; ok {
		return prefix
	}

	// Default to first letter uppercase if not in map
	if len(region) > 0 {
		return strings.ToUpper(region[0:1])
	}

	return "R" // Default fallback
}

// Helper function to get a single letter prefix for an environment
func getEnvironmentPrefix(env string) string {
	envPrefixMap := map[string]string{
		"dev":   "D",
		"test":  "T",
		"stage": "S",
		"prod":  "P",
		"qa":    "Q",
		"uat":   "U",
	}

	// Check if we have a predefined prefix
	if prefix, ok := envPrefixMap[env]; ok {
		return prefix
	}

	// Default to first letter uppercase if not in map
	if len(env) > 0 {
		return strings.ToUpper(env[0:1])
	}

	return "E" // Default fallback
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
}

# Output the resource ID and name for reference by other resources
output "id" {
  value = resource.%s.this.id
  description = "The ID of the %s"
}

output "name" {
  value = resource.%s.this.name
  description = "The name of the %s"
}`, comp.Source, strings.Join(allAttributes, "\n"), strings.Join(blocks, "\n"),
		comp.Source, comp.Source, comp.Source, comp.Source)
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

// ReadTGSConfig reads the TGS configuration from tgs.yaml
func ReadTGSConfig() (*config.TGSConfig, error) {
	// Get the config directory
	configDir := getConfigDir()

	// Try to read from the .tgs directory first
	configPath := filepath.Join(configDir, "tgs.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Try the executable's directory
		execPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("failed to get executable path: %w", err)
		}
		execDir := filepath.Dir(execPath)
		data, err = os.ReadFile(filepath.Join(execDir, "tgs.yaml"))
		if err != nil {
			// Try current directory as fallback
			data, err = os.ReadFile("tgs.yaml")
			if err != nil {
				return nil, err
			}
		}
	}

	var cfg config.TGSConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set a default project name if it's empty
	if cfg.Name == "" {
		logger.Warning("Project name not set in tgs.yaml, using default: CUSTTP")
		cfg.Name = "CUSTTP"
	}

	return &cfg, nil
}

// Update readMainConfig to accept a stack name parameter
func readMainConfig(stackName string) (*config.MainConfig, error) {
	// Get the stacks directory
	stacksDir := getStacksDir()

	// Try to read from the .tgs/stacks directory first
	stackPath := filepath.Join(stacksDir, fmt.Sprintf("%s.yaml", stackName))
	data, err := os.ReadFile(stackPath)
	if err != nil {
		// Try the executable's directory
		execPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("failed to get executable path: %w", err)
		}
		execDir := filepath.Dir(execPath)
		data, err = os.ReadFile(filepath.Join(execDir, fmt.Sprintf("%s.yaml", stackName)))
		if err != nil {
			// Try current directory as fallback
			data, err = os.ReadFile(fmt.Sprintf("%s.yaml", stackName))
			if err != nil {
				return nil, err
			}
		}
	}

	var cfg config.MainConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func createFile(path string, content string) error {
	// Ensure the parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return os.WriteFile(path, []byte(content), 0644)
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

			// Create environment config file with default values
			configContent := fmt.Sprintf(`# Configuration for %s environment
# Override these values as needed for your environment

locals {
  # Default SKUs and tiers for various services
  app_service_sku = {
    name     = "%s"
    tier     = "Standard"
    size     = "%s"
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
}`, envName, getDefaultSkuForEnvironment(envName), getDefaultSkuForEnvironment(envName))

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

// Update the component.hcl template to use environment config files and infrastructure path
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

// Helper function to generate environment-specific inputs based on component type
func generateEnvConfigInputs(compName string) string {
	switch compName {
	case "appservice":
		return `# App Service specific settings
service_plan_id = dependency.serviceplan.outputs.id
app_settings = try(local.env_config.locals.app_service_settings, {})
https_only = true

site_config {
  application_stack {
    dotnet_version = "6.0"
  }
  always_on = true
}`
	case "serviceplan":
		return `# Service Plan specific settings
sku_name = try(local.env_config.locals.app_service_sku.name, "B1")
sku_tier = try(local.env_config.locals.app_service_sku.tier, "Basic")
sku_size = try(local.env_config.locals.app_service_sku.size, "B1")
sku_capacity = try(local.env_config.locals.app_service_sku.capacity, 1)
os_type = "Linux"
worker_count = try(local.env_config.locals.app_service_sku.capacity, 1)`
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

// getConfigDir returns the path to the .tgs config directory
func getConfigDir() string {
	return ".tgs"
}

// getStacksDir returns the path to the .tgs/stacks directory
func getStacksDir() string {
	return filepath.Join(getConfigDir(), "stacks")
}

// ReadMainConfig reads the stack configuration from the .tgs/stacks directory
func ReadMainConfig(stackName string) (*config.MainConfig, error) {
	stackPath := filepath.Join(".tgs", "stacks", fmt.Sprintf("%s.yaml", stackName))
	data, err := os.ReadFile(stackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read stack config file: %w", err)
	}

	var config config.MainConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stack config: %w", err)
	}

	return &config, nil
}
