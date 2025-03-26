package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"github.com/davoodharun/terragrunt-scaffolder/internal/templates"
)

type EnvironmentTemplateData struct {
	EnvironmentName           string
	EnvironmentPrefix         string
	Region                    string
	RegionPrefix              string
	Subscription              string
	RemoteStateResourceGroup  string
	RemoteStateStorageAccount string
	StackName                 string
	Component                 string
	HasAppSettings            bool
	HasPolicyFiles            bool
}

func generateEnvironment(subscription, region string, envName string, components []config.RegionComponent, infraPath string) error {
	// Get the stack name from the environment
	stackName := "main"
	tgsConfig, err := config.ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Find the stack name for this environment
	if sub, ok := tgsConfig.Subscriptions[subscription]; ok {
		for _, env := range sub.Environments {
			if env.Name == envName {
				if env.Stack != "" {
					stackName = env.Stack
				}
				break
			}
		}
	}

	// Create architecture folder structure
	architecturePath := filepath.Join(infraPath, "architecture")
	if err := os.MkdirAll(architecturePath, 0755); err != nil {
		return fmt.Errorf("failed to create architecture directory: %w", err)
	}

	// Create environment base path
	basePath := filepath.Join(architecturePath, subscription, region, envName)
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create environment directory: %w", err)
	}

	// Create environment.hcl
	envData := EnvironmentTemplateData{
		EnvironmentName:   envName,
		EnvironmentPrefix: getEnvironmentPrefix(envName),
	}
	if err := templates.Render("environment/environment.hcl.tmpl", filepath.Join(basePath, "environment.hcl"), envData); err != nil {
		return fmt.Errorf("failed to create environment.hcl: %w", err)
	}

	// Create region.hcl in the region directory
	regionPath := filepath.Join(architecturePath, subscription, region)
	if err := os.MkdirAll(regionPath, 0755); err != nil {
		return fmt.Errorf("failed to create region directory: %w", err)
	}

	regionData := EnvironmentTemplateData{
		Region:       region,
		RegionPrefix: getRegionPrefix(region),
	}
	if err := templates.Render("environment/region.hcl.tmpl", filepath.Join(regionPath, "region.hcl"), regionData); err != nil {
		return fmt.Errorf("failed to create region.hcl: %w", err)
	}

	// Create subscription.hcl in the subscription directory
	subPath := filepath.Join(architecturePath, subscription)
	if err := os.MkdirAll(subPath, 0755); err != nil {
		return fmt.Errorf("failed to create subscription directory: %w", err)
	}

	sub, exists := tgsConfig.Subscriptions[subscription]
	if !exists {
		return fmt.Errorf("subscription %s not found in TGS config", subscription)
	}

	subData := EnvironmentTemplateData{
		Subscription:              subscription,
		RemoteStateResourceGroup:  sub.RemoteState.ResourceGroup,
		RemoteStateStorageAccount: sub.RemoteState.Name,
	}
	if err := templates.Render("environment/subscription.hcl.tmpl", filepath.Join(subPath, "subscription.hcl"), subData); err != nil {
		return fmt.Errorf("failed to create subscription.hcl: %w", err)
	}

	// Generate component directories and their apps
	for _, comp := range components {
		compPath := filepath.Join(basePath, comp.Component)
		if err := os.MkdirAll(compPath, 0755); err != nil {
			return fmt.Errorf("failed to create component directory: %w", err)
		}

		// Read the stack configuration to check if app_settings is enabled
		mainConfig, err := ReadMainConfig(stackName)
		if err != nil {
			return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
		}

		// Check if the component has app_settings or policy_files enabled
		hasAppSettings := false
		hasPolicyFiles := false
		if compConfig, ok := mainConfig.Stack.Components[comp.Component]; ok {
			hasAppSettings = compConfig.AppSettings
			hasPolicyFiles = compConfig.PolicyFiles
		}

		compData := EnvironmentTemplateData{
			StackName:      stackName,
			Component:      comp.Component,
			HasAppSettings: hasAppSettings,
			HasPolicyFiles: hasPolicyFiles,
		}

		if len(comp.Apps) > 0 {
			// Create app-specific folders and terragrunt files
			for _, app := range comp.Apps {
				appPath := filepath.Join(compPath, app)
				if err := os.MkdirAll(appPath, 0755); err != nil {
					return fmt.Errorf("failed to create app directory %s: %w", appPath, err)
				}

				if err := templates.Render("environment/terragrunt.hcl.tmpl", filepath.Join(appPath, "terragrunt.hcl"), compData); err != nil {
					return fmt.Errorf("failed to create terragrunt.hcl for app: %w", err)
				}
			}
		} else {
			// Create single terragrunt.hcl for components without apps
			if err := templates.Render("environment/terragrunt.hcl.tmpl", filepath.Join(compPath, "terragrunt.hcl"), compData); err != nil {
				return fmt.Errorf("failed to create terragrunt.hcl for component: %w", err)
			}
		}
	}

	return nil
}

func generateEnvironmentConfigs(tgsConfig *config.TGSConfig, infraPath string) error {
	// Create config directory
	configDir := filepath.Join(infraPath, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Build the data structure for global.hcl template
	globalData := templates.GlobalConfigData{
		ProjectName: tgsConfig.Name,
		Stacks:      make(map[string]templates.StackConfig),
	}

	// Track unique stacks and their environments
	uniqueStacks := make(map[string]bool)
	stackEnvironments := make(map[string]map[string]bool)

	// First pass: collect all unique stacks and their environments
	for _, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}
			uniqueStacks[stackName] = true

			if _, ok := stackEnvironments[stackName]; !ok {
				stackEnvironments[stackName] = make(map[string]bool)
			}
			stackEnvironments[stackName][env.Name] = true
		}
	}

	// Second pass: build the complete data structure
	for stackName := range uniqueStacks {
		stackConfig := templates.StackConfig{
			Environments: make(map[string]templates.EnvironmentConfig),
		}

		// Add environments for this stack
		for envName := range stackEnvironments[stackName] {
			envConfig := templates.EnvironmentConfig{
				Prefix:  getEnvironmentPrefix(envName),
				Regions: make(map[string]templates.RegionConfig),
			}

			// Add regions for this environment
			// We'll use the regions from the stack configuration
			mainConfig, err := ReadMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			for region := range mainConfig.Stack.Architecture.Regions {
				envConfig.Regions[region] = templates.RegionConfig{
					Prefix: getRegionPrefix(region),
				}
			}

			stackConfig.Environments[envName] = envConfig
		}

		globalData.Stacks[stackName] = stackConfig
	}

	// Generate global.hcl using the template
	globalPath := filepath.Join(configDir, "global.hcl")
	if err := templates.Render("environment/global.hcl.tmpl", globalPath, globalData); err != nil {
		return fmt.Errorf("failed to create global config file: %w", err)
	}

	logger.Success("Generated environment configuration files")

	// Generate a config file for each environment in each subscription
	for subName, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			envName := env.Name

			// Use the stack specified in the environment config, default to "main" if not specified
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}

			// Create environments directory under the stack's config folder
			environmentsDir := filepath.Join(configDir, stackName, "environments", subName)
			if err := os.MkdirAll(environmentsDir, 0755); err != nil {
				return fmt.Errorf("failed to create environments directory: %w", err)
			}

			// Read the stack configuration to get actual components
			mainConfig, err := ReadMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			// Build environment config content with only the components that exist in the stack
			var configContent strings.Builder
			configContent.WriteString(fmt.Sprintf("# Configuration for %s environment in stack %s\n", envName, stackName))
			configContent.WriteString("# Override these values as needed for your environment\n\n")
			configContent.WriteString("locals {\n")

			// Add configurations only for components that exist in the stack
			for compName, comp := range mainConfig.Stack.Components {
				if comp.Provider == "" {
					continue
				}

				// Fetch provider schema for this component
				schema, err := fetchProviderSchema(comp.Provider, comp.Version, comp.Source)
				if err != nil || schema == nil {
					logger.Warning("Failed to fetch provider schema for %s: %v", compName, err)
					continue
				}

				// Try different provider keys
				providerKeys := []string{
					"registry.terraform.io/hashicorp/azurerm",
					"hashicorp/azurerm",
				}

				var resourceSchema struct {
					Block struct {
						Attributes map[string]SchemaAttribute `json:"attributes"`
						BlockTypes map[string]struct {
							Block struct {
								Attributes map[string]SchemaAttribute `json:"attributes"`
							} `json:"block"`
							NestingMode string `json:"nesting_mode"`
						} `json:"block_types"`
					} `json:"block"`
				}

				found := false
				for _, key := range providerKeys {
					if provider, ok := schema.ProviderSchema[key]; ok {
						if rs, ok := provider.ResourceSchemas[comp.Source]; ok {
							resourceSchema = rs
							found = true
							break
						}
					}
				}

				if !found {
					continue
				}

				// Start component configuration block
				configContent.WriteString(fmt.Sprintf("  # %s Configuration\n", compName))
				configContent.WriteString(fmt.Sprintf("  %s = {\n", compName))

				// Add required attributes with default values
				for name, attr := range resourceSchema.Block.Attributes {
					if attr.Required && !shouldSkipVariable(name, comp.Source) {
						defaultValue := getDefaultValueForType(attr.Type, name, envName)
						configContent.WriteString(fmt.Sprintf("    %s = %s\n", name, defaultValue))
					}
				}

				// Add block types (nested configurations)
				for blockName, blockType := range resourceSchema.Block.BlockTypes {
					configContent.WriteString(fmt.Sprintf("    %s = {\n", blockName))
					for attrName, attr := range blockType.Block.Attributes {
						if attr.Required {
							defaultValue := getDefaultValueForType(attr.Type, attrName, envName)
							configContent.WriteString(fmt.Sprintf("      %s = %s\n", attrName, defaultValue))
						}
					}
					configContent.WriteString("    }\n")
				}

				configContent.WriteString("  }\n\n")
			}

			configContent.WriteString("}")

			// Create environment config file in the environments directory
			configPath := filepath.Join(environmentsDir, fmt.Sprintf("%s.env.hcl", envName))
			if err := createFile(configPath, configContent.String()); err != nil {
				return fmt.Errorf("failed to create environment config file: %w", err)
			}

			logger.Info("Generated environment config file: %s", configPath)
		}
	}

	return nil
}

// Helper function to get default value based on type and environment
func getDefaultValueForType(attrType interface{}, name string, env string) string {
	switch t := attrType.(type) {
	case string:
		switch t {
		case "string":
			// Special cases for known attributes
			switch name {
			case "sku_name":
				if strings.Contains(env, "redis") || strings.Contains(env, "cache") {
					return fmt.Sprintf(`"%s"`, getDefaultRedisSkuForEnvironment(env))
				}
				return fmt.Sprintf(`"%s"`, getDefaultSkuForEnvironment(env))
			case "family":
				return `"C"`
			case "tier":
				return `"Standard"`
			case "os_type":
				return `"Linux"`
			case "service_plan_id":
				return `"" # Required: Set this in environment config`
			default:
				return `""`
			}
		case "number":
			return "0"
		case "bool":
			return "false"
		case "list":
			return "[]"
		case "map":
			return "{}"
		default:
			return "null"
		}
	case map[string]interface{}:
		if t["type"] == "string" {
			return `""`
		}
		return "null"
	default:
		return "null"
	}
}

// Helper function to determine default SKU based on environment
func getDefaultSkuForEnvironment(env string) string {
	switch env {
	case "prod":
		return "P1v2"
	case "stage":
		return "P1v2"
	case "test":
		return "S1"
	case "dev":
		return "B1"
	default:
		return "B1"
	}
}

// Helper function to determine default Redis SKU based on environment
func getDefaultRedisSkuForEnvironment(env string) string {
	switch env {
	case "prod":
		return "Premium"
	case "stage":
		return "Standard"
	case "test":
		return "Standard"
	case "dev":
		return "Basic"
	default:
		return "Basic"
	}
}

func generateRootHCL(tgsConfig *config.TGSConfig, infraPath string) error {
	logger.Info("Generating root.hcl configuration")

	// Ensure the .infrastructure directory exists
	baseDir := filepath.Base(infraPath)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create infrastructure directory: %w", err)
	}

	// Create a new template renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to create template renderer: %w", err)
	}

	// Render the root.hcl template
	rootHCL, err := renderer.RenderTemplate("environment/root.hcl.tmpl", nil)
	if err != nil {
		return fmt.Errorf("failed to render root.hcl template: %w", err)
	}

	return createFile(filepath.Join(baseDir, "root.hcl"), rootHCL)
}

// generateEnvironmentConfig creates environment-specific configuration files
func generateEnvironmentConfig(infraPath string, tgsConfig *config.TGSConfig, stackName string) error {
	// Create environments directory under the stack's config folder
	environmentsDir := filepath.Join(infraPath, "config", stackName, "environments")
	if err := os.MkdirAll(environmentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create environments directory: %w", err)
	}

	// Initialize template renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to create template renderer: %w", err)
	}

	// Generate environment config files for each subscription and environment
	for subName, sub := range tgsConfig.Subscriptions {
		// Create subscription directory
		subDir := filepath.Join(environmentsDir, subName)
		if err := os.MkdirAll(subDir, 0755); err != nil {
			return fmt.Errorf("failed to create subscription directory: %w", err)
		}

		for _, env := range sub.Environments {
			// Skip environments that don't belong to this stack
			if env.Stack != "" && env.Stack != stackName {
				continue
			}

			// Create environment config file
			envConfigPath := filepath.Join(subDir, env.Name+".env.hcl")
			envConfigData := templates.EnvironmentConfigData{
				EnvironmentName:   env.Name,
				EnvironmentPrefix: env.Prefix,
				StackName:         stackName,
			}
			envConfigContent, err := renderer.RenderTemplate("environment_config.hcl.tmpl", envConfigData)
			if err != nil {
				return fmt.Errorf("failed to render environment config template: %w", err)
			}

			if err := createFile(envConfigPath, envConfigContent); err != nil {
				return fmt.Errorf("failed to create environment config file: %w", err)
			}
		}
	}

	return nil
}
