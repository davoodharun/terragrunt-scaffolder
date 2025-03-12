// Package template provides functionality for generating template files
package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TGSYamlTemplate is the default template for tgs.yaml
const TGSYamlTemplate = `name: projectA  # Your project name
subscriptions:
  nonprod:
    remotestate:
      name: stprojectanonprodtf  # Example: Storage account name for remote state
      resource_group: rg-projecta-nonprod-tf  # Example: Resource group for remote state
    environments:
      - name: dev
        stack: main
      - name: test
        stack: main
  prod:
    remotestate:
      name: stprojectaprodtf  # Example: Storage account name for remote state
      resource_group: rg-projecta-prod-tf  # Example: Resource group for remote state
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

// InitProject initializes a new project with tgs.yaml
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
	if err := CreateStack("main"); err != nil {
		return fmt.Errorf("failed to create main.yaml: %w", err)
	}

	fmt.Println("Successfully created tgs.yaml in", configPath)
	fmt.Println("Successfully created main.yaml in", mainStackPath)
	fmt.Println("Project initialization complete!")
	return nil
}

// CreateStack creates a new stack configuration file
func CreateStack(name string) error {
	// Create stacks directory if it doesn't exist
	stacksDir := ".tgs/stacks"
	if err := os.MkdirAll(stacksDir, 0755); err != nil {
		return fmt.Errorf("failed to create stacks directory: %w", err)
	}

	// Create the YAML structure with ordered nodes
	root := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "stack"},
					{
						Kind: yaml.MappingNode,
						Content: []*yaml.Node{
							// Stack metadata first
							{Kind: yaml.ScalarNode, Value: "name"},
							{Kind: yaml.ScalarNode, Value: name},
							{Kind: yaml.ScalarNode, Value: "version"},
							{Kind: yaml.ScalarNode, Value: "1.0.0"},
							{Kind: yaml.ScalarNode, Value: "description"},
							{Kind: yaml.ScalarNode, Value: "Default infrastructure stack with web applications and supporting services"},
							// Components section second
							{Kind: yaml.ScalarNode, Value: "components"},
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "serviceplan"},
									{
										Kind: yaml.MappingNode,
										Content: []*yaml.Node{
											{Kind: yaml.ScalarNode, Value: "source"},
											{Kind: yaml.ScalarNode, Value: "azurerm_service_plan"},
											{Kind: yaml.ScalarNode, Value: "provider"},
											{Kind: yaml.ScalarNode, Value: "azurerm"},
											{Kind: yaml.ScalarNode, Value: "version"},
											{Kind: yaml.ScalarNode, Value: "3.0.0"},
											{Kind: yaml.ScalarNode, Value: "description"},
											{Kind: yaml.ScalarNode, Value: "Shared service plan for web applications"},
										},
									},
									{Kind: yaml.ScalarNode, Value: "appservice"},
									{
										Kind: yaml.MappingNode,
										Content: []*yaml.Node{
											{Kind: yaml.ScalarNode, Value: "source"},
											{Kind: yaml.ScalarNode, Value: "azurerm_linux_web_app"},
											{Kind: yaml.ScalarNode, Value: "provider"},
											{Kind: yaml.ScalarNode, Value: "azurerm"},
											{Kind: yaml.ScalarNode, Value: "version"},
											{Kind: yaml.ScalarNode, Value: "3.0.0"},
											{Kind: yaml.ScalarNode, Value: "description"},
											{Kind: yaml.ScalarNode, Value: "Web application service"},
											{Kind: yaml.ScalarNode, Value: "deps"},
											{
												Kind: yaml.SequenceNode,
												Content: []*yaml.Node{
													{Kind: yaml.ScalarNode, Value: "{region}.serviceplan"},
												},
											},
										},
									},
									{Kind: yaml.ScalarNode, Value: "rediscache"},
									{
										Kind: yaml.MappingNode,
										Content: []*yaml.Node{
											{Kind: yaml.ScalarNode, Value: "source"},
											{Kind: yaml.ScalarNode, Value: "azurerm_redis_cache"},
											{Kind: yaml.ScalarNode, Value: "provider"},
											{Kind: yaml.ScalarNode, Value: "azurerm"},
											{Kind: yaml.ScalarNode, Value: "version"},
											{Kind: yaml.ScalarNode, Value: "3.0.0"},
											{Kind: yaml.ScalarNode, Value: "description"},
											{Kind: yaml.ScalarNode, Value: "Redis cache for application caching"},
										},
									},
									{Kind: yaml.ScalarNode, Value: "keyvault"},
									{
										Kind: yaml.MappingNode,
										Content: []*yaml.Node{
											{Kind: yaml.ScalarNode, Value: "source"},
											{Kind: yaml.ScalarNode, Value: "azurerm_key_vault"},
											{Kind: yaml.ScalarNode, Value: "provider"},
											{Kind: yaml.ScalarNode, Value: "azurerm"},
											{Kind: yaml.ScalarNode, Value: "version"},
											{Kind: yaml.ScalarNode, Value: "3.0.0"},
											{Kind: yaml.ScalarNode, Value: "description"},
											{Kind: yaml.ScalarNode, Value: "Key Vault for storing secrets and certificates"},
										},
									},
									{Kind: yaml.ScalarNode, Value: "appservice_api"},
									{
										Kind: yaml.MappingNode,
										Content: []*yaml.Node{
											{Kind: yaml.ScalarNode, Value: "source"},
											{Kind: yaml.ScalarNode, Value: "azurerm_linux_web_app"},
											{Kind: yaml.ScalarNode, Value: "provider"},
											{Kind: yaml.ScalarNode, Value: "azurerm"},
											{Kind: yaml.ScalarNode, Value: "version"},
											{Kind: yaml.ScalarNode, Value: "3.0.0"},
											{Kind: yaml.ScalarNode, Value: "description"},
											{Kind: yaml.ScalarNode, Value: "Backend API service"},
											{Kind: yaml.ScalarNode, Value: "deps"},
											{
												Kind: yaml.SequenceNode,
												Content: []*yaml.Node{
													{Kind: yaml.ScalarNode, Value: "{region}.serviceplan"},
													{Kind: yaml.ScalarNode, Value: "{region}.rediscache"},
													{Kind: yaml.ScalarNode, Value: "{region}.keyvault"},
													{Kind: yaml.ScalarNode, Value: "westus2.appservice.web"},
												},
											},
										},
									},
								},
							},
							// Architecture section third
							{Kind: yaml.ScalarNode, Value: "architecture"},
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "regions"},
									{
										Kind: yaml.MappingNode,
										Content: []*yaml.Node{
											{Kind: yaml.ScalarNode, Value: "eastus2"},
											{
												Kind: yaml.SequenceNode,
												Content: []*yaml.Node{
													{
														Kind: yaml.MappingNode,
														Content: []*yaml.Node{
															{Kind: yaml.ScalarNode, Value: "component"},
															{Kind: yaml.ScalarNode, Value: "serviceplan"},
															{Kind: yaml.ScalarNode, Value: "apps"},
															{Kind: yaml.SequenceNode},
														},
													},
													{
														Kind: yaml.MappingNode,
														Content: []*yaml.Node{
															{Kind: yaml.ScalarNode, Value: "component"},
															{Kind: yaml.ScalarNode, Value: "rediscache"},
															{Kind: yaml.ScalarNode, Value: "apps"},
															{Kind: yaml.SequenceNode},
														},
													},
													{
														Kind: yaml.MappingNode,
														Content: []*yaml.Node{
															{Kind: yaml.ScalarNode, Value: "component"},
															{Kind: yaml.ScalarNode, Value: "keyvault"},
															{Kind: yaml.ScalarNode, Value: "apps"},
															{Kind: yaml.SequenceNode},
														},
													},
													{
														Kind: yaml.MappingNode,
														Content: []*yaml.Node{
															{Kind: yaml.ScalarNode, Value: "component"},
															{Kind: yaml.ScalarNode, Value: "appservice_api"},
															{Kind: yaml.ScalarNode, Value: "apps"},
															{
																Kind: yaml.SequenceNode,
																Content: []*yaml.Node{
																	{Kind: yaml.ScalarNode, Value: "api"},
																},
															},
														},
													},
												},
											},
											{Kind: yaml.ScalarNode, Value: "westus2"},
											{
												Kind: yaml.SequenceNode,
												Content: []*yaml.Node{
													{
														Kind: yaml.MappingNode,
														Content: []*yaml.Node{
															{Kind: yaml.ScalarNode, Value: "component"},
															{Kind: yaml.ScalarNode, Value: "serviceplan"},
															{Kind: yaml.ScalarNode, Value: "apps"},
															{Kind: yaml.SequenceNode},
														},
													},
													{
														Kind: yaml.MappingNode,
														Content: []*yaml.Node{
															{Kind: yaml.ScalarNode, Value: "component"},
															{Kind: yaml.ScalarNode, Value: "appservice"},
															{Kind: yaml.ScalarNode, Value: "apps"},
															{
																Kind: yaml.SequenceNode,
																Content: []*yaml.Node{
																	{Kind: yaml.ScalarNode, Value: "web"},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Add a comment header
	header := `# Stack Configuration
# This example demonstrates a multi-region architecture with dependencies:
#
# East US 2 Region:
# - Service Plan for hosting applications
# - Redis Cache for caching
# - Key Vault for secrets
# - API App Service with dependencies on:
#   - Local Service Plan
#   - Local Redis Cache
#   - Local Key Vault
#   - Web App in West US 2 (cross-region dependency)
#
# West US 2 Region:
# - Service Plan for hosting applications
# - Web App Service (frontend)
#
# This setup shows both:
# 1. Dependencies within the same region (API -> Service Plan, Redis, Key Vault)
# 2. Cross-region dependencies (API -> Web App)

`

	// Write the configuration to file
	filename := filepath.Join(stacksDir, fmt.Sprintf("%s.yaml", name))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create stack file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	if err := encoder.Encode(root); err != nil {
		return fmt.Errorf("failed to write stack config: %w", err)
	}

	fmt.Printf("Created stack configuration: %s\n", filename)
	return nil
}

// ListStacks lists all available stacks in the .tgs/stacks directory
func ListStacks() error {
	files, err := os.ReadDir(".tgs/stacks")
	if err != nil {
		return fmt.Errorf("failed to read stacks directory: %w", err)
	}

	fmt.Println("\nAvailable stacks:")
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			fmt.Printf("- %s\n", strings.TrimSuffix(file.Name(), ".yaml"))
		}
	}

	return nil
}
