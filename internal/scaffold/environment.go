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
		RegionPrefix: GetRegionPrefix(region),
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

// generateEnvironmentConfigs generates environment configuration files
func generateEnvironmentConfigs(tgsConfig *config.TGSConfig, infraPath string) error {
	// Initialize template renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to initialize template renderer: %w", err)
	}

	// Get the config directory
	configDir := filepath.Join(infraPath, "config")

	// Process each subscription
	for subName, sub := range tgsConfig.Subscriptions {
		// Process each environment
		for _, env := range sub.Environments {
			// Create environment directory
			envDir := filepath.Join(configDir, subName, env.Name)
			if err := os.MkdirAll(envDir, 0755); err != nil {
				return fmt.Errorf("failed to create environment directory for %s/%s: %w", subName, env.Name, err)
			}

			// Prepare environment data
			envData := &templates.EnvironmentTemplateData{
				EnvironmentName:           env.Name,
				EnvironmentPrefix:         getEnvironmentPrefix(env.Name),
				Subscription:              subName,
				RemoteStateResourceGroup:  sub.RemoteState.ResourceGroup,
				RemoteStateStorageAccount: sub.RemoteState.Name,
			}

			// Render environment.hcl template
			envHcl, err := renderer.RenderTemplate("environment/environment.hcl.tmpl", envData)
			if err != nil {
				return fmt.Errorf("failed to render environment.hcl template: %w", err)
			}

			// Write environment.hcl file
			if err := createFile(filepath.Join(envDir, "environment.hcl"), envHcl); err != nil {
				return fmt.Errorf("failed to create environment.hcl: %w", err)
			}
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

	// Create root directory
	rootDir := filepath.Join(infraPath, "root")
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return fmt.Errorf("failed to create root directory: %w", err)
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

	return createFile(filepath.Join(rootDir, "root.hcl"), rootHCL)
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
