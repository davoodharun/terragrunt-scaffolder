package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

// Move all terraform file generation functions here
// (generateMainTF, generateVariablesTF, generateProviderTF, etc.)

func generateTerraformFiles(compPath string, comp config.Component) error {
	if comp.Provider == "" {
		return fmt.Errorf("no provider specified for component")
	}

	// Fetch provider schema from Terraform Registry
	schema, err := fetchProviderSchema(comp.Provider, comp.Version, comp.Source)
	if err != nil {
		logger.Warning("Failed to fetch provider schema: %v, generating basic terraform files", err)
		schema = nil
	}

	// Generate main.tf
	var mainContent string
	if schema != nil {
		mainContent = generateMainTF(comp, schema)
	} else {
		mainContent = fmt.Sprintf(`
resource "%s" "this" {
  name                = var.name
  resource_group_name = var.resource_group_name
  location            = var.location

  tags = var.tags
}

output "id" {
  value = resource.%s.this.id
  description = "The ID of the %s"
}

output "name" {
  value = resource.%s.this.name
  description = "The name of the %s"
}`, comp.Source, comp.Source, comp.Source, comp.Source, comp.Source)
	}

	mainPath := filepath.Join(compPath, "main.tf")
	if err := createFile(mainPath, mainContent); err != nil {
		return fmt.Errorf("failed to create main.tf: %w", err)
	}

	// Generate variables.tf
	var varsContent string
	if schema != nil {
		varsContent = generateVariablesTF(schema, comp)
	} else {
		varsContent = `
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
	}

	varsPath := filepath.Join(compPath, "variables.tf")
	if err := createFile(varsPath, varsContent); err != nil {
		return fmt.Errorf("failed to create variables.tf: %w", err)
	}

	// Generate provider.tf
	providerContent := generateProviderTF(comp)
	providerPath := filepath.Join(compPath, "provider.tf")
	if err := createFile(providerPath, providerContent); err != nil {
		return fmt.Errorf("failed to create provider.tf: %w", err)
	}

	// Verify all required files exist
	requiredFiles := []string{"main.tf", "variables.tf", "provider.tf"}
	for _, file := range requiredFiles {
		filePath := filepath.Join(compPath, file)
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("failed to verify required file %s: %w", file, err)
		}
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

	// Special handling for Redis Cache
	isRedisCache := strings.Contains(comp.Source, "redis_cache")

	// Generate attribute assignments - separate required and optional
	for name, attr := range resourceSchema.Block.Attributes {
		if shouldSkipVariable(name, comp.Source) {
			continue
		}

		if attr.Required {
			// Special handling for Redis Cache family attribute
			if isRedisCache && name == "family" {
				requiredAttributes = append(requiredAttributes, fmt.Sprintf("  %s = coalesce(var.family, \"C\")", name))
			} else {
				requiredAttributes = append(requiredAttributes, fmt.Sprintf("  %s = var.%s", name, name))
			}
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
			if name == "family" {
				return `default = "C"`
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
