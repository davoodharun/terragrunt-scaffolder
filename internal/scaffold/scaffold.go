package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"github.com/davoodharun/terragrunt-scaffolder/internal/validate"
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

// Update the function to get the infrastructure path
func getInfrastructurePath() string {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		logger.Warning("Failed to get current working directory: %v", err)
		return ".infrastructure"
	}

	// Check if .infrastructure exists in the current directory
	infraPath := filepath.Join(cwd, ".infrastructure")
	if _, err := os.Stat(infraPath); err == nil {
		return infraPath
	}

	// If not found, create it
	if err := os.MkdirAll(infraPath, 0755); err != nil {
		logger.Warning("Failed to create .infrastructure directory: %v", err)
		return ".infrastructure"
	}

	return infraPath
}

// Generate scaffolds the infrastructure based on the TGS configuration
func Generate() error {
	// Read TGS configuration
	tgsConfig, err := config.ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Validate TGS configuration
	if errors := validate.ValidateTGSConfig(tgsConfig); len(errors) > 0 {
		return fmt.Errorf("invalid TGS configuration: %v", errors[0])
	}

	logger.Info("Creating infrastructure directory structure")

	// Create infrastructure directory if it doesn't exist
	infraPath := ".infrastructure"
	if err := os.MkdirAll(infraPath, 0755); err != nil {
		return fmt.Errorf("failed to create infrastructure directory: %w", err)
	}

	// Create config directory
	configDir := filepath.Join(infraPath, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create components directory
	componentsDir := filepath.Join(infraPath, "_components")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create components directory: %w", err)
	}

	// Create architecture directory
	architectureDir := filepath.Join(infraPath, "architecture")
	if err := os.MkdirAll(architectureDir, 0755); err != nil {
		return fmt.Errorf("failed to create architecture directory: %w", err)
	}

	logger.Success("Infrastructure directory structure created")

	// Calculate total steps for progress bar
	totalSteps := 0
	// Count components
	for _, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}
			mainConfig, err := config.ReadMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}
			totalSteps += len(mainConfig.Stack.Components)
		}
	}
	// Add steps for root.hcl, environment configs, and architecture
	totalSteps += 1 + len(tgsConfig.Subscriptions) + len(tgsConfig.Subscriptions)

	// Initialize progress bar
	logger.StartProgress("Generating infrastructure", totalSteps)

	// Generate root.hcl file
	logger.Info("Generating root.hcl configuration")
	if err := generateRootHCL(tgsConfig, infraPath); err != nil {
		return fmt.Errorf("failed to generate root.hcl: %w", err)
	}
	logger.Success("Root.hcl configuration generated")
	logger.UpdateProgress()

	// Generate environment configurations
	logger.Info("Generating environment configurations")
	if err := generateEnvironmentConfigs(tgsConfig, infraPath); err != nil {
		return fmt.Errorf("failed to generate environment configs: %w", err)
	}
	logger.Success("Environment configurations generated")
	logger.UpdateProgress()

	// Generate architecture and components for each subscription and environment
	for subName, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}
			mainConfig, err := config.ReadMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			// Process each region in the stack configuration
			for region, components := range mainConfig.Stack.Architecture.Regions {
				logger.Info("Generating architecture for %s/%s/%s", subName, env.Name, region)
				// Generate architecture
				if err := generateEnvironment(subName, region, env.Name, components, infraPath); err != nil {
					return fmt.Errorf("failed to generate environment for %s/%s: %w", subName, env.Name, err)
				}
				logger.Success("Architecture generated for %s/%s/%s", subName, env.Name, region)
				logger.UpdateProgress()

				logger.Info("Generating components for %s/%s/%s", subName, env.Name, region)
				// Generate components
				if err := generateComponents(mainConfig, infraPath); err != nil {
					return fmt.Errorf("failed to generate components: %w", err)
				}
				logger.Success("Components generated for %s/%s/%s", subName, env.Name, region)
				logger.UpdateProgress()
			}
		}
	}

	// Finish progress bar
	logger.FinishProgress()
	logger.Success("Infrastructure generation completed successfully")

	return nil
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

// ReadMainConfig reads the stack configuration from the .tgs/stacks directory
func ReadMainConfig(stackName string) (*config.MainConfig, error) {
	stackPath := filepath.Join(".tgs", "stacks", fmt.Sprintf("%s.yaml", stackName))
	data, err := os.ReadFile(stackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read stack config: %w", err)
	}

	var mainConfig config.MainConfig
	if err := yaml.Unmarshal(data, &mainConfig); err != nil {
		return nil, fmt.Errorf("failed to parse stack config: %w", err)
	}

	return &mainConfig, nil
}

func createFile(path string, content string) error {
	// Ensure the parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// getConfigDir returns the path to the .tgs config directory
func getConfigDir() string {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		logger.Warning("Failed to get current working directory: %v", err)
		return ".tgs"
	}

	// Check if .tgs exists in the current directory
	configPath := filepath.Join(cwd, ".tgs")
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}

	// If not found, create it
	if err := os.MkdirAll(configPath, 0755); err != nil {
		logger.Warning("Failed to create .tgs directory: %v", err)
		return ".tgs"
	}

	return configPath
}

// getStacksDir returns the path to the .tgs/stacks directory
func getStacksDir() string {
	configDir := getConfigDir()
	stacksDir := filepath.Join(configDir, "stacks")

	// Create stacks directory if it doesn't exist
	if err := os.MkdirAll(stacksDir, 0755); err != nil {
		logger.Warning("Failed to create stacks directory: %v", err)
		return filepath.Join(".tgs", "stacks")
	}

	return stacksDir
}

// GetRegionPrefix returns a prefix for a region
func GetRegionPrefix(region string) string {
	regionPrefixMap := map[string]string{
		"eastus":        "E",
		"eastus2":       "E2",
		"canadacentral": "CC",
		"canadaeast":    "CE",
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
