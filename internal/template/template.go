// Package template provides functionality for generating template files
package template

import (
	"fmt"
	"os"
	"path/filepath"
)

// TGSYamlTemplate is the default template for tgs.yaml
const TGSYamlTemplate = `name: CUSTTP  # Your project name
subscriptions:
  nonprod:
    remotestate:
      name: # Storage account name for remote state
      resource_group: # Resource group for remote state
    environments:
      - name: dev
        stack: main
      - name: test
        stack: main
  prod:
    remotestate:
      name: # Storage account name for remote state
      resource_group: # Resource group for remote state
    environments:
      - name: prod
        stack: main
`

// MainYamlTemplate is the default template for main.yaml (stack configuration)
const MainYamlTemplate = `stack:
  components:
    # Example components
    rediscache:
      source: azurerm_redis_cache
      provider: azurerm
      version: 4.22.0
      deps: []
    appservice:
      source: azurerm_app_service
      provider: azurerm
      version: 4.22.0
      deps:
        # Dependency notation examples:
        # - "eastus2.redis"                # Fixed region and component
        # - "{region}.serviceplan"         # Current region with fixed component
        # - "{region}.serviceplan.{app}"   # Current region, component, and app
        # - "eastus2.cosmos_db.api"        # Fixed region, component, and app
        - "{region}.serviceplan.{app}"     # Depends on serviceplan in same region for same app
    serviceplan:
      source: azurerm_service_plan
      provider: azurerm
      version: 4.22.0 
      deps: []
  architecture:
    regions:
      eastus2:
        - component: rediscache
          apps: []
        - component: serviceplan
          apps: 
            - api
            - web
        - component: appservice
          apps:
            - api
            - web
      westus:
        - component: serviceplan
          apps: 
            - api
            - web
        - component: appservice
          apps:
            - api
            - web
`

// getConfigDir returns the path to the .tgs config directory
func getConfigDir() string {
	return ".tgs"
}

// getStacksDir returns the path to the .tgs/stacks directory
func getStacksDir() string {
	return filepath.Join(getConfigDir(), "stacks")
}

// CreateFileIfNotExists creates a file with the given content if it doesn't exist
func CreateFileIfNotExists(path string, content string) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file %s already exists", path)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create the file
	return os.WriteFile(path, []byte(content), 0644)
}

// InitProject initializes a new project with tgs.yaml and main.yaml
func InitProject() error {
	fmt.Println("Initializing new project with tgs.yaml...")

	// Create .tgs directory
	configDir := getConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	// Create tgs.yaml in .tgs directory
	configPath := filepath.Join(configDir, "tgs.yaml")
	if err := CreateFileIfNotExists(configPath, TGSYamlTemplate); err != nil {
		return fmt.Errorf("failed to create tgs.yaml: %w", err)
	}

	// Create .tgs/stacks directory
	stacksDir := getStacksDir()
	if err := os.MkdirAll(stacksDir, 0755); err != nil {
		return fmt.Errorf("failed to create stacks directory %s: %w", stacksDir, err)
	}

	// Create default main.yaml in .tgs/stacks directory
	mainStackPath := filepath.Join(stacksDir, "main.yaml")
	if err := CreateFileIfNotExists(mainStackPath, MainYamlTemplate); err != nil {
		return fmt.Errorf("failed to create main.yaml: %w", err)
	}

	fmt.Println("Successfully created tgs.yaml in", configPath)
	fmt.Println("Successfully created main.yaml in", mainStackPath)
	fmt.Println("Project initialization complete!")
	return nil
}

// CreateStack creates a new stack configuration file
func CreateStack(name string) error {
	// Create .tgs/stacks directory
	stacksDir := getStacksDir()
	if err := os.MkdirAll(stacksDir, 0755); err != nil {
		return fmt.Errorf("failed to create stacks directory %s: %w", stacksDir, err)
	}

	filename := fmt.Sprintf("%s.yaml", name)
	stackPath := filepath.Join(stacksDir, filename)

	fmt.Printf("Creating new stack configuration: %s...\n", stackPath)

	if err := CreateFileIfNotExists(stackPath, MainYamlTemplate); err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}

	fmt.Printf("Successfully created %s\n", stackPath)
	return nil
}

// ListStacks lists all stack files in the .tgs/stacks directory
func ListStacks() error {
	stacksDir := getStacksDir()

	// Check if stacks directory exists
	if _, err := os.Stat(stacksDir); os.IsNotExist(err) {
		fmt.Println("No stacks found. Use 'tgs create stack' to create a stack.")
		return nil
	}

	// Read all files in the stacks directory
	files, err := os.ReadDir(stacksDir)
	if err != nil {
		return fmt.Errorf("failed to read stacks directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No stacks found. Use 'tgs create stack' to create a stack.")
		return nil
	}

	fmt.Println("Available stacks:")
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			stackName := file.Name()[:len(file.Name())-len(".yaml")]
			fmt.Printf("  - %s\n", stackName)
		}
	}

	return nil
}
