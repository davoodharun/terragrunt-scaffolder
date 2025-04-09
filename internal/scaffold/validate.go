package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// fileExists checks if a file exists and is not a directory
func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

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
	tgsConfig, err := config.ReadTGSConfig()
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
	logger.Info("Validating component structure at %s", componentPath)
	requiredFiles := []string{
		"component.hcl",
		"main.tf",
		"variables.tf",
		"provider.tf",
	}

	for _, file := range requiredFiles {
		filePath := filepath.Join(componentPath, file)
		exists, err := fileExists(filePath)
		if err != nil {
			logger.Error("Error checking file %s: %v", file, err)
			return fmt.Errorf("error checking file %s: %w", file, err)
		}
		if !exists {
			logger.Error("Required file %s is missing in component", file)
			return fmt.Errorf("required file %s is missing in component", file)
		}
	}

	logger.Success("Component structure validation passed")
	return nil
}

// ValidateComponentVariables validates component variables against environment config
func ValidateComponentVariables(componentPath string, envConfigPath string) error {
	logger.Info("Validating component variables against environment config")

	exists, err := fileExists(envConfigPath)
	if err != nil {
		logger.Error("Error checking environment config file: %v", err)
		return fmt.Errorf("error checking environment config file: %w", err)
	}
	if !exists {
		logger.Error("Environment config file %s does not exist", envConfigPath)
		return fmt.Errorf("environment config file %s does not exist", envConfigPath)
	}

	componentHCL := filepath.Join(componentPath, "component.hcl")
	exists, err = fileExists(componentHCL)
	if err != nil {
		logger.Error("Error checking component.hcl file: %v", err)
		return fmt.Errorf("error checking component.hcl file: %w", err)
	}
	if !exists {
		logger.Error("component.hcl file does not exist in %s", componentPath)
		return fmt.Errorf("component.hcl file does not exist in %s", componentPath)
	}

	logger.Success("Component variables validation passed")
	return nil
}

// getInfrastructurePath returns the path to the infrastructure directory
func getInfrastructurePath() string {
	return ".infrastructure"
}
