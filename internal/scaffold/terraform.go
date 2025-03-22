package scaffold

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

// Move all terraform file generation functions here
// (generateMainTF, generateVariablesTF, generateProviderTF, etc.)

func generateTerraformFiles(compPath string, comp config.Component) error {
	if comp.Provider == "" {
		logger.Warning("No provider specified for component, skipping terraform file generation")
		return nil
	}

	// Fetch provider schema from Terraform Registry
	schema, err := fetchProviderSchema(comp.Provider, comp.Version, comp.Source)
	if err != nil {
		logger.Warning("Failed to fetch provider schema: %v, generating basic terraform files", err)
		// Generate basic files without schema
		return generateBasicTerraformFiles(compPath, comp)
	}

	if schema == nil {
		logger.Warning("Provider schema is nil, generating basic terraform files")
		return generateBasicTerraformFiles(compPath, comp)
	}

	// Generate main.tf using the schema
	mainContent := generateMainTF(comp, schema)
	if err := createFile(filepath.Join(compPath, "main.tf"), mainContent); err != nil {
		return err
	}

	// Generate variables.tf using the schema
	varsContent := generateVariablesTF(schema, comp)
	if err := createFile(filepath.Join(compPath, "variables.tf"), varsContent); err != nil {
		return err
	}

	// Generate provider.tf
	providerContent := generateProviderTF(comp)
	if err := createFile(filepath.Join(compPath, "provider.tf"), providerContent); err != nil {
		return err
	}

	return nil
}

func generateBasicTerraformFiles(compPath string, comp config.Component) error {
	// Generate basic main.tf
	mainContent := fmt.Sprintf(`
resource "%s" "this" {
  name                = var.name
  resource_group_name = var.resource_group_name
  location            = var.location

  tags = var.tags
}`, comp.Source)

	if err := createFile(filepath.Join(compPath, "main.tf"), mainContent); err != nil {
		return err
	}

	// Generate basic variables.tf
	varsContent := `
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
}`

	if err := createFile(filepath.Join(compPath, "variables.tf"), varsContent); err != nil {
		return err
	}

	// Generate provider.tf
	providerContent := generateProviderTF(comp)
	if err := createFile(filepath.Join(compPath, "provider.tf"), providerContent); err != nil {
		return err
	}

	return nil
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
	skip_provider_registration = true
}

data "azurerm_client_config" "current" {}
`, comp.Version)
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
		if shouldSkipVariable(name, comp.Source) {
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

func shouldSkipVariable(name string, resourceType string) bool {
	// Skip common variables that are handled separately
	commonVars := []string{
		"name",
		"resource_group_name",
		"location",
		"tags",
	}

	for _, v := range commonVars {
		if name == v {
			return true
		}
	}

	// Skip certain attributes for specific resource types
	skipForResource := map[string][]string{
		"azurerm_redis_cache": {
			"zones", // zones is not used in the current implementation
		},
	}

	if attrs, ok := skipForResource[resourceType]; ok {
		for _, attr := range attrs {
			if name == attr {
				return true
			}
		}
	}

	return false
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
			if shouldSkipVariable(name, comp.Source) {
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
	// Handle common defaults
	switch name {
	case "tags":
		return `default = {}`
	case "os_type":
		return `default = "Windows"`
	case "sku_name":
		return `default = "B1"`
	case "worker_count":
		return `default = 1`
	case "maximum_elastic_worker_count":
		return `default = 1`
	case "per_site_scaling_enabled":
		return `default = true`
	case "app_service_environment_id":
		return `default = ""`
	case "id":
		return `default = ""`
	case "timeouts":
		return `default = []`
	case "app_settings":
		return `default = {}`
	case "tenant_settings":
		return `default = {}`
	case "enable_non_ssl_port":
		return `default = false`
	case "public_network_access_enabled":
		return `default = true`
	case "redis_version":
		return `default = "6"`
	case "shard_count":
		return `default = 1`
	case "minimum_tls_version":
		return `default = "1.2"`
	case "private_static_ip_address":
		return `default = ""`
	case "capacity":
		return `default = 1`
	case "family":
		return `default = "C"`
	case "replicas_per_primary":
		return `default = 1`
	case "subnet_id":
		return `default = ""`
	case "replicas_per_master":
		return `default = 1`
	case "patch_schedule":
		return `default = []`
	case "access_policy":
		return `default = []`
	}

	// Handle type-specific defaults
	switch attr.Type {
	case "string":
		return `default = ""`
	case "number":
		return `default = 0`
	case "bool":
		return `default = false`
	case "list":
		return `default = []`
	case "map":
		return `default = {}`
	case "object":
		return `default = {}`
	default:
		return ""
	}
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

// getRequiredAttributes returns a list of required attributes for a given resource type
func getRequiredAttributes(schema *ProviderSchema, resourceType string) []string {
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
			if rs, ok := provider.ResourceSchemas[resourceType]; ok {
				resourceSchema = rs
				found = true
				break
			}
		}
	}

	if !found {
		return []string{"name", "resource_group_name", "location"}
	}

	required := []string{"name", "resource_group_name", "location"} // Always include common required fields

	for name, attr := range resourceSchema.Block.Attributes {
		if shouldSkipVariable(name, resourceType) {
			continue
		}
		if attr.Required {
			required = append(required, name)
		}
	}

	// Also check for required nested blocks
	for blockName, blockType := range resourceSchema.Block.BlockTypes {
		hasRequiredAttrs := false
		for _, attr := range blockType.Block.Attributes {
			if attr.Required {
				hasRequiredAttrs = true
				break
			}
		}
		if hasRequiredAttrs {
			required = append(required, blockName)
		}
	}

	return required
}
