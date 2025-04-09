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

// Generate creates the infrastructure directory structure and files
func Generate(tgsConfig *config.TGSConfig) error {
	// Calculate total steps for progress bar
	totalSteps := 1 // root.hcl
	totalSteps++    // environment configs

	// Count regions per environment
	regionCount := 0
	stackName := "main"
	mainConfig, err := ReadMainConfig(stackName)
	if err != nil {
		logger.Error("Failed to read stack config %s: %v", stackName, err)
		return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
	}
	regionCount = len(mainConfig.Stack.Architecture.Regions)

	// Add steps for each environment's regions
	for _, sub := range tgsConfig.Subscriptions {
		totalSteps += len(sub.Environments) * regionCount
	}
	totalSteps++ // components generation

	logger.StartProgress("Generating infrastructure", totalSteps)
	logger.Info("Starting infrastructure generation")

	// Create infrastructure directory
	infraPath := ".infrastructure"
	if err := createDirectory(infraPath); err != nil {
		logger.Error("Failed to create infrastructure directory: %v", err)
		return fmt.Errorf("failed to create infrastructure directory: %w", err)
	}
	logger.Success("Infrastructure folder created at %s", infraPath)

	// Create required directories
	dirs := []string{
		filepath.Join(infraPath, "config"),
		filepath.Join(infraPath, "_components"),
		filepath.Join(infraPath, "architecture"),
	}

	for _, dir := range dirs {
		if err := createDirectory(dir); err != nil {
			logger.Error("Failed to create directory %s: %v", dir, err)
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		logger.Success("Created directory: %s", dir)
	}

	// Generate root.hcl
	logger.Info("Generating root.hcl configuration")
	if err := generateRootHCL(tgsConfig, infraPath); err != nil {
		logger.Error("Failed to generate root.hcl: %v", err)
		return fmt.Errorf("failed to generate root.hcl: %w", err)
	}
	logger.Success("Generated root.hcl configuration")
	logger.UpdateProgress()

	// Generate environment configs
	logger.Info("Generating environment configurations")
	if err := generateEnvironmentConfigs(tgsConfig, infraPath); err != nil {
		logger.Error("Failed to generate environment configs: %v", err)
		return fmt.Errorf("failed to generate environment configs: %w", err)
	}
	logger.Success("Generated environment configurations")
	logger.UpdateProgress()

	// Process each subscription and environment
	for subName, sub := range tgsConfig.Subscriptions {
		logger.Info("Processing subscription: %s", subName)
		for _, env := range sub.Environments {
			logger.Info("Processing environment: %s in subscription %s", env.Name, subName)

			// Generate environment-specific files
			for region, components := range mainConfig.Stack.Architecture.Regions {
				logger.Info("Generating files for region %s", region)
				if err := generateEnvironment(subName, region, env.Name, components, infraPath); err != nil {
					logger.Error("Failed to generate environment for %s/%s: %v", subName, env.Name, err)
					return fmt.Errorf("failed to generate environment for %s/%s: %w", subName, env.Name, err)
				}
				logger.Success("Generated files for %s/%s/%s", subName, env.Name, region)
				logger.UpdateProgress()
			}
		}
	}

	// Generate components
	logger.Info("Generating components")
	if err := generateComponents(mainConfig, infraPath); err != nil {
		logger.Error("Failed to generate components: %v", err)
		return fmt.Errorf("failed to generate components: %w", err)
	}
	logger.Success("Components generated successfully")
	logger.UpdateProgress()

	logger.FinishProgress()
	logger.Success("Infrastructure generation completed successfully")
	return nil
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

// createDirectory creates a directory and its parents if they don't exist
func createDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// joinPath joins path elements and creates the directory if it doesn't exist
func joinPath(elem ...string) (string, error) {
	path := filepath.Join(elem...)
	if err := createDirectory(path); err != nil {
		return "", err
	}
	return path, nil
}
