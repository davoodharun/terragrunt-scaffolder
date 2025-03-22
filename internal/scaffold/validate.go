package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ValidateGeneratedConfigs checks the HCL syntax of all generated configuration files
func ValidateGeneratedConfigs() error {
	logger.Info("Validating generated configuration files")

	// Check environment config files
	configDir := filepath.Join(".infrastructure", "config")
	if err := validateHCLFiles(configDir, "*.hcl"); err != nil {
		return fmt.Errorf("environment config validation failed: %w", err)
	}

	// Check component config files
	componentsDir := filepath.Join(".infrastructure", "_components")
	if err := validateHCLFiles(componentsDir, "**/*.hcl"); err != nil {
		return fmt.Errorf("component config validation failed: %w", err)
	}

	logger.Success("All configuration files validated successfully")
	return nil
}

// validateHCLFiles validates HCL syntax for all files matching the pattern in the given directory
func validateHCLFiles(dir, pattern string) error {
	// Walk through all files in the directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file matches pattern
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if !matched {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Parse HCL
		_, diags := hclparse.NewParser().ParseHCL(content, path)
		if diags.HasErrors() {
			var errors []string
			for _, diag := range diags {
				errors = append(errors, fmt.Sprintf("%s: %s", path, diag.Error()))
			}
			return fmt.Errorf("HCL syntax errors found:\n%s", strings.Join(errors, "\n"))
		}

		logger.Info("Validated HCL syntax for: %s", path)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to validate HCL files: %w", err)
	}

	return nil
}

// ValidateConfigs validates all configuration files in the project
func ValidateConfigs() error {
	logger.Info("Starting configuration validation")

	// Validate TGS config
	if err := validateTGSConfig(); err != nil {
		return fmt.Errorf("TGS config validation failed: %w", err)
	}

	// Validate stack configs
	if err := validateStackConfigs(); err != nil {
		return fmt.Errorf("stack config validation failed: %w", err)
	}

	// Validate generated configs
	if err := ValidateGeneratedConfigs(); err != nil {
		return fmt.Errorf("generated config validation failed: %w", err)
	}

	logger.Success("All configurations validated successfully")
	return nil
}

// validateTGSConfig validates the TGS configuration file
func validateTGSConfig() error {
	logger.Info("Validating TGS configuration")

	// Read TGS config
	tgsConfig, err := ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Validate project name
	if tgsConfig.Name == "" {
		return fmt.Errorf("project name is required in TGS config")
	}

	// Validate subscriptions
	if len(tgsConfig.Subscriptions) == 0 {
		return fmt.Errorf("at least one subscription is required in TGS config")
	}

	for subName, sub := range tgsConfig.Subscriptions {
		// Validate remote state
		if sub.RemoteState.Name == "" {
			return fmt.Errorf("remote state name is required for subscription %s", subName)
		}
		if sub.RemoteState.ResourceGroup == "" {
			return fmt.Errorf("remote state resource group is required for subscription %s", subName)
		}

		// Validate environments
		if len(sub.Environments) == 0 {
			return fmt.Errorf("at least one environment is required for subscription %s", subName)
		}

		for _, env := range sub.Environments {
			if env.Name == "" {
				return fmt.Errorf("environment name is required for subscription %s", subName)
			}
		}
	}

	logger.Success("TGS configuration validated successfully")
	return nil
}

// validateStackConfigs validates all stack configuration files
func validateStackConfigs() error {
	logger.Info("Validating stack configurations")

	// Get stacks directory
	stacksDir := getStacksDir()

	// Read all stack files
	entries, err := os.ReadDir(stacksDir)
	if err != nil {
		return fmt.Errorf("failed to read stacks directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			stackName := strings.TrimSuffix(entry.Name(), ".yaml")
			logger.Info("Validating stack: %s", stackName)

			// Read stack config
			mainConfig, err := ReadMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			// Validate stack name
			if mainConfig.Stack.Name == "" {
				return fmt.Errorf("stack name is required in stack config %s", stackName)
			}

			// Validate architecture
			if len(mainConfig.Stack.Architecture.Regions) == 0 {
				return fmt.Errorf("at least one region is required in stack config %s", stackName)
			}

			// Validate components
			if len(mainConfig.Stack.Components) == 0 {
				return fmt.Errorf("at least one component is required in stack config %s", stackName)
			}

			for compName, comp := range mainConfig.Stack.Components {
				if comp.Source == "" {
					return fmt.Errorf("source is required for component %s in stack %s", compName, stackName)
				}
				if comp.Provider == "" {
					return fmt.Errorf("provider is required for component %s in stack %s", compName, stackName)
				}
				if comp.Version == "" {
					return fmt.Errorf("version is required for component %s in stack %s", compName, stackName)
				}
			}
		}
	}

	logger.Success("Stack configurations validated successfully")
	return nil
}

// ValidateComponentStructure validates the structure and content of a component directory
func ValidateComponentStructure(componentPath string) error {
	// Check required files exist
	requiredFiles := []string{
		"main.tf",
		"variables.tf",
		"provider.tf",
		"component.hcl",
	}

	for _, file := range requiredFiles {
		filePath := filepath.Join(componentPath, file)
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("required file %s is missing in component %s: %w", file, componentPath, err)
		}
	}

	// Parse variables.tf to get defined variables
	varsContent, err := os.ReadFile(filepath.Join(componentPath, "variables.tf"))
	if err != nil {
		return fmt.Errorf("failed to read variables.tf: %w", err)
	}

	// Parse component.hcl to get input variables
	compContent, err := os.ReadFile(filepath.Join(componentPath, "component.hcl"))
	if err != nil {
		return fmt.Errorf("failed to read component.hcl: %w", err)
	}

	// Extract variables from variables.tf
	parser := hclparse.NewParser()
	varsFile, diags := parser.ParseHCL(varsContent, "variables.tf")
	if diags.HasErrors() {
		return fmt.Errorf("invalid HCL in variables.tf: %v", diags)
	}

	// Extract inputs from component.hcl
	compFile, diags := parser.ParseHCL(compContent, "component.hcl")
	if diags.HasErrors() {
		return fmt.Errorf("invalid HCL in component.hcl: %v", diags)
	}

	// Get all inputs defined in component.hcl
	definedInputs := make(map[string]bool)
	for _, block := range compFile.Body.(*hclsyntax.Body).Blocks {
		if block.Type == "inputs" {
			// Check direct attributes
			for name := range block.Body.Attributes {
				definedInputs[name] = true
			}
			// Check nested blocks
			for _, nestedBlock := range block.Body.Blocks {
				if nestedBlock.Type == "tags" {
					definedInputs["tags"] = true
				}
			}
		}
	}

	// Also check for inputs in locals block
	for _, block := range compFile.Body.(*hclsyntax.Body).Blocks {
		if block.Type == "locals" {
			for name := range block.Body.Attributes {
				if name == "resource_name" {
					definedInputs["name"] = true
				}
				if name == "resource_group_name" {
					definedInputs["resource_group_name"] = true
				}
				if name == "region_name" {
					definedInputs["location"] = true
				}
			}
		}
	}

	// Get the resource type from main.tf
	mainContent, err := os.ReadFile(filepath.Join(componentPath, "main.tf"))
	if err != nil {
		return fmt.Errorf("failed to read main.tf: %w", err)
	}

	mainFile, diags := parser.ParseHCL(mainContent, "main.tf")
	if diags.HasErrors() {
		return fmt.Errorf("invalid HCL in main.tf: %v", diags)
	}

	var resourceType string
	for _, block := range mainFile.Body.(*hclsyntax.Body).Blocks {
		if block.Type == "resource" && len(block.Labels) >= 2 {
			// Get the full resource type (e.g., azurerm_service_plan)
			resourceType = block.Labels[0]
			break
		}
	}

	if resourceType == "" {
		return fmt.Errorf("could not determine resource type from main.tf")
	}

	// Get required variables from provider schema
	requiredVars := getRequiredVariablesForResource(resourceType)

	// Check each required variable
	for varName := range requiredVars {
		hasDefault := false
		hasInput := false

		// Check for default value in variables.tf
		for _, block := range varsFile.Body.(*hclsyntax.Body).Blocks {
			if block.Type == "variable" && block.Labels[0] == varName {
				if _, exists := block.Body.Attributes["default"]; exists {
					hasDefault = true
					break
				}
			}
		}

		// Check for input in component.hcl
		hasInput = definedInputs[varName]

		if !hasDefault && !hasInput {
			return fmt.Errorf("required variable %s for resource %s must have either a default value in variables.tf or be set in component.hcl", varName, resourceType)
		}
	}

	// Check for unused variables
	for _, block := range varsFile.Body.(*hclsyntax.Body).Blocks {
		if block.Type != "variable" {
			continue
		}

		varName := block.Labels[0]

		// Skip common variables that are handled separately
		if varName == "name" || varName == "resource_group_name" || varName == "location" || varName == "tags" {
			continue
		}

		// Check if variable is used in main.tf
		isUsed := false
		for _, mainBlock := range mainFile.Body.(*hclsyntax.Body).Blocks {
			if mainBlock.Type == "resource" {
				for _, attr := range mainBlock.Body.Attributes {
					if attr.Name == varName {
						isUsed = true
						break
					}
				}
				if isUsed {
					break
				}
			}
		}

		// If variable is not used and has no default value, it should be set in component.hcl
		if !isUsed {
			hasDefault := false
			if _, exists := block.Body.Attributes["default"]; exists {
				hasDefault = true
			}

			if !hasDefault && !definedInputs[varName] {
				return fmt.Errorf("variable %s is defined in variables.tf but is not used in main.tf and has no default value or component.hcl input", varName)
			}
		}
	}

	return nil
}

// getRequiredVariablesForResource returns a map of required variables for a given Azure resource type
func getRequiredVariablesForResource(resourceType string) map[string]bool {
	requiredVars := make(map[string]bool)

	// Common required variables for all Azure resources
	requiredVars["name"] = true
	requiredVars["resource_group_name"] = true
	requiredVars["location"] = true

	// Resource-specific required variables
	switch resourceType {
	case "azurerm_service_plan":
		requiredVars["sku_name"] = true
		requiredVars["os_type"] = true
	case "azurerm_app_service":
		requiredVars["service_plan_id"] = true
	case "azurerm_function_app":
		requiredVars["service_plan_id"] = true
	case "azurerm_redis_cache":
		requiredVars["sku_name"] = true
	case "azurerm_key_vault":
		requiredVars["sku_name"] = true
	case "azurerm_servicebus_namespace":
		requiredVars["sku"] = true
	case "azurerm_cosmosdb_account":
		requiredVars["offer_type"] = true
		requiredVars["consistency_level"] = true
	case "azurerm_storage_account":
		requiredVars["account_tier"] = true
		requiredVars["account_replication_type"] = true
	case "azurerm_sql_server":
		requiredVars["version"] = true
		requiredVars["administrator_login"] = true
	case "azurerm_sql_database":
		requiredVars["server_id"] = true
		requiredVars["sku_name"] = true
	case "azurerm_eventhub_namespace":
		requiredVars["sku"] = true
	case "azurerm_log_analytics_workspace":
		requiredVars["sku"] = true
	}

	return requiredVars
}

// ValidateComponentVariables ensures all required variables have values either through defaults or component.hcl inputs
func ValidateComponentVariables(componentPath string, envConfigPath string) error {
	// Read variables.tf
	varsContent, err := os.ReadFile(filepath.Join(componentPath, "variables.tf"))
	if err != nil {
		return fmt.Errorf("failed to read variables.tf: %w", err)
	}

	// Parse variables.tf
	parser := hclparse.NewParser()
	varsFile, diags := parser.ParseHCL(varsContent, "variables.tf")
	if diags.HasErrors() {
		return fmt.Errorf("invalid HCL in variables.tf: %v", diags)
	}

	// Read component.hcl
	compContent, err := os.ReadFile(filepath.Join(componentPath, "component.hcl"))
	if err != nil {
		return fmt.Errorf("failed to read component.hcl: %w", err)
	}

	// Parse component.hcl
	compFile, diags := parser.ParseHCL(compContent, "component.hcl")
	if diags.HasErrors() {
		return fmt.Errorf("invalid HCL in component.hcl: %v", diags)
	}

	// Get all inputs defined in component.hcl
	definedInputs := make(map[string]bool)
	for _, block := range compFile.Body.(*hclsyntax.Body).Blocks {
		if block.Type == "inputs" {
			for _, attr := range block.Body.Attributes {
				definedInputs[attr.Name] = true
			}
		}
	}

	// Check each variable
	for _, block := range varsFile.Body.(*hclsyntax.Body).Blocks {
		if block.Type != "variable" {
			continue
		}

		varName := block.Labels[0]
		isRequired := false
		hasDefault := false

		// Check if variable is required
		if requiredAttr, exists := block.Body.Attributes["required"]; exists {
			if requiredExpr, ok := requiredAttr.Expr.(*hclsyntax.LiteralValueExpr); ok {
				isRequired = requiredExpr.Val.True()
			}
		}

		// Check for default value
		if _, exists := block.Body.Attributes["default"]; exists {
			hasDefault = true
		}

		// If variable is required, ensure it has either a default value or is set in component.hcl
		if isRequired {
			if !hasDefault && !definedInputs[varName] {
				return fmt.Errorf("required variable %s has no default value and is not set in component.hcl", varName)
			}
		}
	}

	return nil
}
