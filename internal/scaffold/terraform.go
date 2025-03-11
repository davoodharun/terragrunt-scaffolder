package scaffold

import (
	"fmt"
	"path/filepath"

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
