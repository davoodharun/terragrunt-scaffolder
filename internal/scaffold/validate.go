package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// ValidateGeneratedConfigs checks the HCL syntax of all generated configuration files
func ValidateGeneratedConfigs() error {
	logger.Info("Validating generated configuration files")

	// Get the infrastructure path
	infraPath := getInfrastructurePath()

	// Check environment config files
	configDir := filepath.Join(infraPath, "config")
	if err := validateHCLFiles(configDir, "*.hcl"); err != nil {
		return fmt.Errorf("environment config validation failed: %w", err)
	}

	// Check component config files
	componentsDir := filepath.Join(infraPath, "_components")
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
			if os.IsNotExist(err) {
				return fmt.Errorf("required file %s is missing in component", file)
			}
			return fmt.Errorf("error checking file %s: %w", file, err)
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

// ValidateComponentVariables validates component variables against environment config
func ValidateComponentVariables(componentPath string, envConfigPath string) error {
	// Check if the environment config file exists
	if _, err := os.Stat(envConfigPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("environment config file %s does not exist", envConfigPath)
		}
		return fmt.Errorf("error checking environment config file: %w", err)
	}

	// Check if the component.hcl file exists
	compPath := filepath.Join(componentPath, "component.hcl")
	if _, err := os.Stat(compPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("component.hcl file does not exist in %s", componentPath)
		}
		return fmt.Errorf("error checking component.hcl file: %w", err)
	}

	return nil
}
