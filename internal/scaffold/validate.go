package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"github.com/hashicorp/hcl/v2"
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
		_, diags := hclsyntax.ParseConfig(content, path, hcl.Pos{Line: 1, Column: 1})
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
