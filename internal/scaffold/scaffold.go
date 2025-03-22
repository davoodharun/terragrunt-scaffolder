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

// Add a function to find the Git repository root
func findGitRepoRoot() (string, error) {
	// Start with the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree looking for .git
	for {
		// Check if .git exists in the current directory
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, nil // Found the git repo root
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root of the filesystem without finding .git
			return "", fmt.Errorf("no .git directory found in any parent directory")
		}
		dir = parent
	}
}

// Update the function to get the infrastructure path
func getInfrastructurePath() string {
	// Always use .infrastructure at the repo root
	return ".infrastructure"
}

func Generate() error {
	// Get the infrastructure path
	infraPath := ".infrastructure"

	// Read TGS config
	tgsConfig, err := ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Track existing subscriptions
	existingSubs := make(map[string]bool)

	// Get existing subscriptions (excluding special directories)
	if entries, err := os.ReadDir(infraPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), "_") && !strings.HasPrefix(entry.Name(), ".") && entry.Name() != "config" && entry.Name() != "diagrams" {
				existingSubs[entry.Name()] = true
			}
		}
	}

	// Create infrastructure directory if it doesn't exist
	if err := os.MkdirAll(infraPath, 0755); err != nil {
		return fmt.Errorf("failed to create infrastructure directory: %w", err)
	}

	// Create base directory structure
	baseDir := filepath.Base(infraPath)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	logger.Success("Infrastructure folder created")

	// Generate root.hcl
	if err := generateRootHCL(tgsConfig, infraPath); err != nil {
		return fmt.Errorf("failed to generate root.hcl: %w", err)
	}

	// Generate environment config files
	if err := generateEnvironmentConfigs(tgsConfig, infraPath); err != nil {
		return fmt.Errorf("failed to generate environment config files: %w", err)
	}

	// First pass: collect all unique components and their configurations
	uniqueComponents := make(map[string]config.Component)
	for _, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}

			mainConfig, err := readMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			// Add components from this stack
			for compName, comp := range mainConfig.Stack.Components {
				uniqueComponents[compName] = comp
			}
		}
	}

	// Create components directory
	componentsDir := filepath.Join(baseDir, "_components")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create components directory: %w", err)
	}

	logger.Success("Components created")

	// Start progress bar for component generation
	logger.StartProgress("Generating components", len(uniqueComponents))

	// Generate each unique component once
	mainConfig := &config.MainConfig{
		Stack: config.StackConfig{
			Components: uniqueComponents,
		},
	}

	// Generate components with all necessary files and validation
	if err := generateComponents(mainConfig); err != nil {
		return fmt.Errorf("failed to generate components: %w", err)
	}

	// Process each subscription for environment structure
	for subName, sub := range tgsConfig.Subscriptions {
		// Process each environment with its specified stack
		for _, env := range sub.Environments {
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}

			// Read the stack-specific config
			mainConfig, err := readMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			// Generate environment structure without re-validating components
			for region, components := range mainConfig.Stack.Architecture.Regions {
				if err := generateEnvironment(subName, region, env.Name, components, infraPath); err != nil {
					return fmt.Errorf("failed to generate environment structure: %w", err)
				}
			}
		}
	}

	return nil
}

func createSubscriptionConfig(subPath, subName string, sub config.Subscription) error {
	logger.Info("Creating subscription config for %s", subName)
	subscriptionHCL := fmt.Sprintf(`locals {
  subscription_name = "%s"
  remote_state_resource_group = "%s"
  remote_state_storage_account = "%s"
}`, subName, sub.RemoteState.ResourceGroup, sub.RemoteState.Name)

	return createFile(filepath.Join(subPath, "subscription.hcl"), subscriptionHCL)
}

func createRegionConfig(regionPath, region string) error {
	logger.Info("Creating region config for %s", region)

	// Determine region prefix (single letter)
	regionPrefix := getRegionPrefix(region)

	regionHCL := fmt.Sprintf(`locals {
  region_name = "%s"
  region_prefix = "%s"
}`, region, regionPrefix)

	return createFile(filepath.Join(regionPath, "region.hcl"), regionHCL)
}

// Helper function to get a single letter prefix for a region
func getRegionPrefix(region string) string {
	regionPrefixMap := map[string]string{
		"eastus":        "E",
		"eastus2":       "E2",
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

// ReadTGSConfig reads the TGS configuration from tgs.yaml
func ReadTGSConfig() (*config.TGSConfig, error) {
	// Get the config directory
	configDir := getConfigDir()

	// Try to read from the .tgs directory first
	configPath := filepath.Join(configDir, "tgs.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Try the executable's directory
		execPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("failed to get executable path: %w", err)
		}
		execDir := filepath.Dir(execPath)
		data, err = os.ReadFile(filepath.Join(execDir, "tgs.yaml"))
		if err != nil {
			// Try current directory as fallback
			data, err = os.ReadFile("tgs.yaml")
			if err != nil {
				return nil, err
			}
		}
	}

	var cfg config.TGSConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set a default project name if it's empty
	if cfg.Name == "" {
		logger.Warning("Project name not set in tgs.yaml, using default: CUSTTP")
		cfg.Name = "CUSTTP"
	}

	return &cfg, nil
}

// Update readMainConfig to accept a stack name parameter
func readMainConfig(stackName string) (*config.MainConfig, error) {
	// Get the stacks directory
	stacksDir := getStacksDir()

	// Try to read from the .tgs/stacks directory first
	stackPath := filepath.Join(stacksDir, fmt.Sprintf("%s.yaml", stackName))
	data, err := os.ReadFile(stackPath)
	if err != nil {
		// Try the executable's directory
		execPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("failed to get executable path: %w", err)
		}
		execDir := filepath.Dir(execPath)
		data, err = os.ReadFile(filepath.Join(execDir, fmt.Sprintf("%s.yaml", stackName)))
		if err != nil {
			// Try current directory as fallback
			data, err = os.ReadFile(fmt.Sprintf("%s.yaml", stackName))
			if err != nil {
				return nil, err
			}
		}
	}

	var cfg config.MainConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
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
	return ".tgs"
}

// getStacksDir returns the path to the .tgs/stacks directory
func getStacksDir() string {
	return filepath.Join(getConfigDir(), "stacks")
}

// ReadMainConfig reads the stack configuration from the .tgs/stacks directory
func ReadMainConfig(stackName string) (*config.MainConfig, error) {
	stackPath := filepath.Join(".tgs", "stacks", fmt.Sprintf("%s.yaml", stackName))
	data, err := os.ReadFile(stackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read stack config file: %w", err)
	}

	var config config.MainConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stack config: %w", err)
	}

	return &config, nil
}
