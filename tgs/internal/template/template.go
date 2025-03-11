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

// InitProject initializes a new project with tgs.yaml
func InitProject() error {
	fmt.Println("Initializing new project with tgs.yaml...")
	if err := CreateFileIfNotExists("tgs.yaml", TGSYamlTemplate); err != nil {
		return fmt.Errorf("failed to create tgs.yaml: %w", err)
	}
	fmt.Println("Successfully created tgs.yaml")
	return nil
}

// CreateStack creates a new stack configuration file
func CreateStack(name string) error {
	filename := fmt.Sprintf("%s.yaml", name)
	fmt.Printf("Creating new stack configuration: %s...\n", filename)

	if err := CreateFileIfNotExists(filename, MainYamlTemplate); err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}

	fmt.Printf("Successfully created %s\n", filename)
	return nil
}
