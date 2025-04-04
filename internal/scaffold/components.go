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

func generateComponents(mainConfig *config.MainConfig, infraPath string) error {
	// Get the components directory
	componentsDir := filepath.Join(infraPath, "_components")

	// Create a map to track which components have been validated
	validatedComponents := make(map[string]bool)

	// Process components in dependency order
	for compName, comp := range mainConfig.Stack.Components {
		// Skip if already validated
		if validatedComponents[compName] {
			continue
		}

		// Create component directory
		componentPath := filepath.Join(componentsDir, compName)
		if err := os.MkdirAll(componentPath, 0755); err != nil {
			return fmt.Errorf("failed to create component directory for %s: %w", compName, err)
		}

		// Generate component files
		if err := generateComponentFiles(compName, comp, componentPath); err != nil {
			return fmt.Errorf("failed to generate component files for %s: %w", compName, err)
		}

		// Validate component structure
		if err := ValidateComponentStructure(componentPath); err != nil {
			return fmt.Errorf("component structure validation failed for %s: %w", compName, err)
		}

		// Mark this component as validated
		validatedComponents[compName] = true
	}

	return nil
}

// Helper function to get resource type abbreviation
func getResourceTypeAbbreviation(componentName string) string {
	abbreviations := map[string]string{
		"serviceplan": "asp",
		"appservice":  "app",
		"functionapp": "func",
		"redis":       "redis",
		"storage":     "st",
		"keyvault":    "kv",
		"sql":         "sql",
		"cosmos":      "cos",
	}

	for key, abbr := range abbreviations {
		if strings.Contains(strings.ToLower(componentName), key) {
			return abbr
		}
	}

	// Default to first three letters if no match
	if len(componentName) >= 3 {
		return strings.ToLower(componentName[0:3])
	}
	return strings.ToLower(componentName)
}

// Helper function to analyze required inputs and their dependencies
func analyzeRequiredInputs(comp config.Component) ([]string, map[string]string) {
	// Map of input names to their dependency sources
	dependencyMap := map[string]string{
		"service_plan_id":     "serviceplan",
		"server_id":           "sqlserver",
		"key_vault_id":        "keyvault",
		"storage_account_id":  "storage",
		"cosmosdb_account_id": "cosmos",
	}

	// Extract component type from source
	compType := strings.TrimPrefix(comp.Source, "azurerm_")

	// Define required inputs for each resource type
	requiredInputs := make(map[string][]string)
	requiredInputs["linux_web_app"] = []string{"service_plan_id"}
	requiredInputs["windows_web_app"] = []string{"service_plan_id"}
	requiredInputs["app_service"] = []string{"service_plan_id"}
	requiredInputs["function_app"] = []string{"service_plan_id"}
	requiredInputs["sql_database"] = []string{"server_id"}
	requiredInputs["key_vault_access_policy"] = []string{"key_vault_id"}
	requiredInputs["storage_container"] = []string{"storage_account_id"}
	requiredInputs["cosmosdb_sql_container"] = []string{"cosmosdb_account_id"}

	// Get required inputs for this component type
	inputs := requiredInputs[compType]
	if inputs == nil {
		return nil, nil
	}

	// Find dependencies needed for required inputs
	var deps []string
	inputDeps := make(map[string]string)
	for _, input := range inputs {
		if dep, exists := dependencyMap[input]; exists {
			deps = append(deps, dep)
			inputDeps[input] = dep
		}
	}

	return deps, inputDeps
}

// Helper function to generate environment-specific inputs based on component type
func generateEnvConfigInputs(comp config.Component) string {
	// Extract component type from source
	compType := strings.TrimPrefix(comp.Source, "azurerm_")

	// Analyze required inputs and their dependencies
	_, inputDeps := analyzeRequiredInputs(comp)

	// Handle web app variants
	if strings.Contains(compType, "web_app") || compType == "app_service" {
		var inputs []string
		inputs = append(inputs, `# Web App specific settings`)

		// Add service_plan_id with dependency if needed
		if dep, exists := inputDeps["service_plan_id"]; exists {
			inputs = append(inputs, fmt.Sprintf(`    service_plan_id = dependency.%s.outputs.id`, dep))
		} else {
			inputs = append(inputs, `    service_plan_id = try(local.env_config.locals.serviceplan.id, "") # Required: Set this in environment config`)
		}

		inputs = append(inputs, `    app_settings = try(local.env_config.locals.appservice.app_settings, {})`,
			`    site_config = try(local.env_config.locals.appservice.site_config, {})`)

		return strings.Join(inputs, "\n")
	}

	switch compType {
	case "service_plan":
		return `# Service Plan specific settings
    sku_name = try(local.env_config.locals.serviceplan.sku_name, "B1")
    os_type = try(local.env_config.locals.serviceplan.os_type, "Linux")`
	case "function_app":
		var inputs []string
		inputs = append(inputs, `# Function App specific settings`)

		// Add service_plan_id with dependency if needed
		if dep, exists := inputDeps["service_plan_id"]; exists {
			inputs = append(inputs, fmt.Sprintf(`    service_plan_id = dependency.%s.outputs.id`, dep))
		} else {
			inputs = append(inputs, `    service_plan_id = try(local.env_config.locals.serviceplan.id, "") # Required: Set this in environment config`)
		}

		inputs = append(inputs, `    app_settings = try(local.env_config.locals.functionapp.app_settings, {})`)
		return strings.Join(inputs, "\n")
	case "sql_database":
		var inputs []string
		inputs = append(inputs, `# SQL Database specific settings`)

		// Add server_id with dependency if needed
		if dep, exists := inputDeps["server_id"]; exists {
			inputs = append(inputs, fmt.Sprintf(`    server_id = dependency.%s.outputs.id`, dep))
		} else {
			inputs = append(inputs, `    server_id = try(local.env_config.locals.sql.server_id, "") # Required: Set this in environment config`)
		}

		inputs = append(inputs, `    sku_name = try(local.env_config.locals.sql.sku_name, "Basic")`)
		return strings.Join(inputs, "\n")
	case "redis_cache":
		return `# Redis Cache specific settings
    sku_name = try(local.env_config.locals.redis.sku_name, "Basic")
    family = try(local.env_config.locals.redis.family, "C")`
	case "key_vault":
		return `# Key Vault specific settings
    sku_name = try(local.env_config.locals.keyvault.sku_name, "standard")
    purge_protection_enabled = try(local.env_config.locals.keyvault.purge_protection_enabled, false)`
	case "storage_account":
		return `# Storage Account specific settings
    account_tier = try(local.env_config.locals.storage.account_tier, "Standard")
    account_replication_type = try(local.env_config.locals.storage.account_replication_type, "LRS")`
	case "sql_server":
		return `# SQL Server specific settings
    version = try(local.env_config.locals.sql.version, "12.0")
    administrator_login = try(local.env_config.locals.sql.administrator_login, "sqladmin")
    administrator_login_password = try(local.env_config.locals.sql.administrator_login_password, "") # Required: Set this in environment config`
	case "cosmosdb_account":
		return `# Cosmos DB specific settings
    offer_type = try(local.env_config.locals.cosmos.offer_type, "Standard")
    consistency_level = try(local.env_config.locals.cosmos.consistency_level, "Session")`
	default:
		return "# No specific inputs required for this component type"
	}
}

// Helper function to generate dependency blocks
func generateDependencyBlocks(deps []string, infraPath string) string {
	if len(deps) == 0 {
		return ""
	}

	// Initialize template renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		logger.Warning("Failed to initialize template renderer: %v", err)
		return ""
	}

	var blocks []string
	usedNames := make(map[string]bool)
	for _, dep := range deps {
		// Handle both explicit dependencies and analyzed dependencies
		if strings.Contains(dep, ".") {
			// Handle explicit dependencies (region.component.app format)
			parts := strings.Split(dep, ".")
			if len(parts) < 2 {
				logger.Warning("Invalid dependency format: %s, skipping", dep)
				continue
			}

			region := parts[0]
			component := parts[1]
			app := ""

			if len(parts) > 2 {
				app = parts[2]
			}

			// Replace placeholders
			if region == "{region}" {
				region = "${local.region_vars.locals.region_name}"
			}

			depName := component
			configPath := ""

			if app == "" || app == "{app}" {
				if app == "{app}" {
					// App-specific dependency using current app
					configPath = fmt.Sprintf("${get_repo_root()}/.infrastructure/architecture/${local.subscription_vars.locals.subscription_name}/%s/${local.environment_vars.locals.environment_name}/%s/${local.app_name}", region, component)
				} else {
					// Component-level dependency
					configPath = fmt.Sprintf("${get_repo_root()}/.infrastructure/architecture/${local.subscription_vars.locals.subscription_name}/%s/${local.environment_vars.locals.environment_name}/%s", region, component)
				}
			} else {
				// App-specific dependency with fixed app name
				configPath = fmt.Sprintf("${get_repo_root()}/.infrastructure/architecture/${local.subscription_vars.locals.subscription_name}/%s/${local.environment_vars.locals.environment_name}/%s/%s", region, component, app)
				depName = fmt.Sprintf("%s_%s", component, app)
			}

			// Ensure unique dependency name
			if usedNames[depName] {
				depName = fmt.Sprintf("%s_%d", depName, len(usedNames)+1)
			}
			usedNames[depName] = true

			// Render dependency template
			dependencyData := &templates.DependencyData{
				Name:       depName,
				ConfigPath: configPath,
			}
			block, err := renderer.RenderTemplate("components/dependency.hcl.tmpl", dependencyData)
			if err != nil {
				logger.Warning("Failed to render dependency template: %v", err)
				continue
			}
			blocks = append(blocks, block)
		} else {
			// Handle analyzed dependencies (component name only)
			configPath := fmt.Sprintf("${get_repo_root()}/.infrastructure/architecture/${local.subscription_vars.locals.subscription_name}/${local.region_vars.locals.region_name}/${local.environment_vars.locals.environment_name}/%s", dep)

			// Ensure unique dependency name
			depName := dep
			if usedNames[depName] {
				depName = fmt.Sprintf("%s_%d", depName, len(usedNames)+1)
			}
			usedNames[depName] = true

			dependencyData := &templates.DependencyData{
				Name:       depName,
				ConfigPath: configPath,
			}
			block, err := renderer.RenderTemplate("components/dependency.hcl.tmpl", dependencyData)
			if err != nil {
				logger.Warning("Failed to render dependency template: %v", err)
				continue
			}
			blocks = append(blocks, block)
		}
	}

	return strings.Join(blocks, "\n")
}

// generateAppSettingsStructure creates the app settings folder structure for a component
func generateAppSettingsStructure(compName string, infraPath string, tgsConfig *config.TGSConfig, apps []string) error {
	// Create app settings directory under the stack's config folder
	appSettingsDir := filepath.Join(infraPath, "config", "main", "app_settings_"+compName)
	if err := os.MkdirAll(appSettingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create app settings directory: %w", err)
	}

	// Initialize template renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to create template renderer: %w", err)
	}

	// Generate global app settings file
	globalSettingsPath := filepath.Join(appSettingsDir, "global.appsettings.json")
	if err := createFile(globalSettingsPath, "{}"); err != nil {
		return fmt.Errorf("failed to create global app settings file: %w", err)
	}

	// Create subscription and environment folders
	for subName, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			// Create environment directory
			envDir := filepath.Join(appSettingsDir, subName, env.Name)
			if err := os.MkdirAll(envDir, 0755); err != nil {
				return fmt.Errorf("failed to create environment directory: %w", err)
			}

			// Create environment app settings file
			envSettingsPath := filepath.Join(envDir, env.Name+".appsettings.json")
			if err := createFile(envSettingsPath, "{}"); err != nil {
				return fmt.Errorf("failed to create environment app settings file: %w", err)
			}

			// Create app-specific settings files
			for _, app := range apps {
				appSettingsPath := filepath.Join(envDir, app+".appsettings.json")
				if err := createFile(appSettingsPath, "{}"); err != nil {
					return fmt.Errorf("failed to create app settings file: %w", err)
				}
			}
		}
	}

	// Generate appsettings.hcl file
	appSettingsData := templates.AppSettingsData{
		ComponentName: compName,
		StackName:     "main", // TODO: Get this from the stack config
	}
	appSettingsContent, err := renderer.RenderTemplate("appsettings.hcl.tmpl", appSettingsData)
	if err != nil {
		return fmt.Errorf("failed to render appsettings.hcl template: %w", err)
	}

	appSettingsHCLPath := filepath.Join(appSettingsDir, "appsettings.hcl")
	if err := createFile(appSettingsHCLPath, appSettingsContent); err != nil {
		return fmt.Errorf("failed to create appsettings.hcl file: %w", err)
	}

	return nil
}

// generatePolicyFilesStructure creates the policy files folder structure for a component
func generatePolicyFilesStructure(compName string, infraPath string, tgsConfig *config.TGSConfig, apps []string) error {
	// Create policy files directory under the stack's config folder
	policyFilesDir := filepath.Join(infraPath, "config", "main", "policy_files_"+compName)
	if err := os.MkdirAll(policyFilesDir, 0755); err != nil {
		return fmt.Errorf("failed to create policy files directory: %w", err)
	}

	// Initialize template renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to create template renderer: %w", err)
	}

	// Create subscription and environment folders
	for subName, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			// Create environment directory
			envDir := filepath.Join(policyFilesDir, subName, env.Name)
			if err := os.MkdirAll(envDir, 0755); err != nil {
				return fmt.Errorf("failed to create environment directory: %w", err)
			}

			// Create app-specific policy files
			for _, app := range apps {
				policyFilePath := filepath.Join(envDir, app+".policy.xml")
				if err := createFile(policyFilePath, "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<configuration>\n</configuration>"); err != nil {
					return fmt.Errorf("failed to create policy file: %w", err)
				}
			}
		}
	}

	// Generate policies.hcl file
	policyData := templates.PolicyData{
		ComponentName: compName,
		StackName:     "main", // TODO: Get this from the stack config
	}
	policyContent, err := renderer.RenderTemplate("policies.hcl.tmpl", policyData)
	if err != nil {
		return fmt.Errorf("failed to render policies.hcl template: %w", err)
	}

	policyHCLPath := filepath.Join(policyFilesDir, "policies.hcl")
	if err := createFile(policyHCLPath, policyContent); err != nil {
		return fmt.Errorf("failed to create policies.hcl file: %w", err)
	}

	return nil
}

// generateComponentFiles generates the files for a component
func generateComponentFiles(compName string, comp config.Component, componentPath string) error {
	// Initialize template renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to initialize template renderer: %w", err)
	}

	// Read TGS config to get naming format
	tgsConfig, err := config.ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Generate Terraform files
	if err := generateTerraformFiles(componentPath, comp); err != nil {
		return fmt.Errorf("failed to generate terraform files: %w", err)
	}

	// Use only explicit dependencies from the stack file
	var dependencyBlocks string
	if len(comp.Deps) > 0 {
		deps := generateDependencyBlocks(comp.Deps, ".infrastructure")
		dependencyBlocks = deps
	}

	// Prepare component data
	componentData := &templates.ComponentData{
		ComponentName:    compName,
		Source:           comp.Source,
		Version:          comp.Version,
		ResourceType:     getResourceTypeAbbreviation(compName),
		DependencyBlocks: dependencyBlocks,
		EnvConfigInputs:  generateEnvConfigInputs(comp),
		NamingFormat:     tgsConfig.Naming.Format,
	}

	// Render component.hcl template
	componentHcl, err := renderer.RenderTemplate("components/component.hcl.tmpl", componentData)
	if err != nil {
		return fmt.Errorf("failed to render component.hcl template: %w", err)
	}

	// Write component.hcl file
	if err := createFile(filepath.Join(componentPath, "component.hcl"), componentHcl); err != nil {
		return fmt.Errorf("failed to create component.hcl: %w", err)
	}

	return nil
}
