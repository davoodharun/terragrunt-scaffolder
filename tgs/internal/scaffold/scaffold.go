package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/tgs/internal/config"
	"github.com/davoodharun/tgs/internal/logger"
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

	// Generate components
	if err := generateComponents(mainConfig); err != nil {
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
	subscriptionConfig := fmt.Sprintf(`locals {
  subscription_name = "%s"
  subscription_path = "${get_repo_root()}/.infrastructure/%s"
  remote_state = {
    resource_group_name  = "%s"
    storage_account_name = "%s"
  }
}

# Generate an Azure provider block
generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<EOF
provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
  tenant_id       = var.tenant_id
}
EOF
}

# Set common variables for this subscription
inputs = {
  subscription_id = get_env("ARM_SUBSCRIPTION_ID")
  tenant_id       = get_env("ARM_TENANT_ID")
}`, subName, subName, sub.RemoteState.ResourceGroup, sub.RemoteState.Name)

	return createFile(filepath.Join(subPath, "subscription.hcl"), subscriptionConfig)
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

	var attributes []string
	var blocks []string

	// Generate attribute assignments
	for name, attr := range resourceSchema.Block.Attributes {
		if attr.Required || attr.Optional {
			attributes = append(attributes, fmt.Sprintf("  %s = var.%s", name, name))
		}
	}

	// Generate dynamic blocks
	for blockName, blockType := range resourceSchema.Block.BlockTypes {
		block := fmt.Sprintf(`
  dynamic "%s" {
    for_each = var.%s
    content {
`, blockName, blockName)

		for attrName, attr := range blockType.Block.Attributes {
			if attr.Required || attr.Optional {
				block += fmt.Sprintf("      %s = %s.value.%s\n", attrName, blockName, attrName)
			}
		}

		block += "    }\n  }"
		blocks = append(blocks, block)
	}

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
}`, comp.Source, strings.Join(attributes, "\n"), strings.Join(blocks, "\n"))
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
			if attr.Required || attr.Optional {
				// Skip common variables we've already defined
				if name == "name" || name == "resource_group_name" || name == "location" || name == "tags" {
					continue
				}

				varBlock := fmt.Sprintf(`
variable "%s" {
  type        = %s
  description = "%s"
  %s
}`, name,
					convertType(attr.Type),
					sanitizeDescription(attr.Description),
					generateDefault(SchemaAttribute{
						Type:        attr.Type,
						Required:    attr.Required,
						Optional:    attr.Optional,
						Computed:    attr.Computed,
						Description: attr.Description,
					}))
				variables = append(variables, varBlock)
			}
		}

		// Handle nested blocks
		for blockName, blockType := range resourceSchema.Block.BlockTypes {
			variables = append(variables, generateNestedBlockVariable(blockName, blockType))
		}
	}

	return strings.Join(variables, "\n")
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

func generateDefault(attr SchemaAttribute) string {
	if attr.Optional && !attr.Required {
		switch v := attr.Type.(type) {
		case string:
			switch v {
			case "string":
				return `default = ""`
			case "number":
				return "default = 0"
			case "bool":
				return "default = false"
			case "list":
				return "default = []"
			case "map":
				return "default = {}"
			}
		case []interface{}:
			if len(v) > 0 {
				if typeStr, ok := v[0].(string); ok {
					return generateDefault(SchemaAttribute{
						Type:        typeStr,
						Required:    attr.Required,
						Optional:    attr.Optional,
						Computed:    attr.Computed,
						Description: attr.Description,
					})
				}
			}
		}
	}
	return ""
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

  subscription_id = dependency.subscription.outputs.subscription_id
  tenant_id       = dependency.subscription.outputs.tenant_id
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
	rootHCL := fmt.Sprintf(`# Include this in all terragrunt.hcl files
locals {
  subscription_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl"))
  subscription_name = local.subscription_vars.locals.subscription_name
}

remote_state {
  backend = "azurerm"
  config = {
    resource_group_name  = "%s"
    storage_account_name = "%s"
    container_name      = "${local.subscription_name}"
    key                = "${path_relative_to_include()}/terraform.tfstate"
  }
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }
}

# Generate providers.tf file with default provider configurations
generate "providers" {
  path      = "providers.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<EOF
terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
  backend "azurerm" {}
}

provider "azurerm" {
  features {}
}
EOF
}
`, tgsConfig.Subscriptions[tgsConfig.Name].RemoteState.ResourceGroup,
		tgsConfig.Subscriptions[tgsConfig.Name].RemoteState.Name)

	return createFile(filepath.Join(".infrastructure", "root.hcl"), rootHCL)
}
