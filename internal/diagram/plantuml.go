package diagram

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"github.com/davoodharun/terragrunt-scaffolder/internal/scaffold"
)

// Azure resource type to PlantUML sprite mapping
var azureSprites = map[string]string{
	"appservice":     "AzureAppService",
	"serviceplan":    "AzureAppServicePlan",
	"rediscache":     "AzureRedisCache",
	"cosmos_account": "AzureCosmosDb",
	"cosmos_db":      "AzureCosmosDb",
	"servicebus":     "AzureServiceBus",
	"keyvault":       "AzureKeyVault",
	"storage":        "AzureStorage",
	"functionapp":    "AzureFunction",
	"apim":           "AzureAPIManagement",
	"sql_server":     "AzureSQLServer",
	"sql_database":   "AzureSQLDatabase",
	"eventhub":       "AzureEventHub",
	"loganalytics":   "AzureLogAnalytics",
}

// generatePlantUMLDiagram generates a PlantUML diagram for a specific stack and environment
func generatePlantUMLDiagram(stackName string, tgsConfig *config.TGSConfig, envName string) error {
	logger.Info("Generating PlantUML diagram for stack: %s, environment: %s", stackName, envName)

	// Read stack config
	mainConfig, err := readStackConfig(stackName)
	if err != nil {
		return fmt.Errorf("failed to read stack config: %w", err)
	}

	// Start building the PlantUML diagram
	var diagram strings.Builder
	diagram.WriteString("@startuml\n")

	// Include Azure sprites
	diagram.WriteString("!define AzurePuml https://raw.githubusercontent.com/plantuml-stdlib/Azure-PlantUML/master/dist\n")
	diagram.WriteString("!includeurl AzurePuml/AzureCommon.puml\n")
	diagram.WriteString("!includeurl AzurePuml/AzureSimplified.puml\n")
	diagram.WriteString("!includeurl AzurePuml/Web/all.puml\n")
	diagram.WriteString("!includeurl AzurePuml/Compute/all.puml\n")
	diagram.WriteString("!includeurl AzurePuml/Databases/all.puml\n")
	diagram.WriteString("!includeurl AzurePuml/Integration/all.puml\n")
	diagram.WriteString("!includeurl AzurePuml/Security/all.puml\n")
	diagram.WriteString("!includeurl AzurePuml/Storage/all.puml\n\n")

	// Set up styling
	diagram.WriteString("' Styling\n")
	diagram.WriteString("skinparam rectangle {\n")
	diagram.WriteString("  BackgroundColor<<region>> Azure\n")
	diagram.WriteString("  BorderColor<<region>> Navy\n")
	diagram.WriteString("  FontColor<<region>> Black\n")
	diagram.WriteString("}\n\n")

	// Create a map to track deployable resources and their dependencies
	resources := make(map[string]struct {
		region     string
		component  string
		app        string
		deps       []string
		isDataFlow bool
	})

	// First pass: collect all deployable resources
	for region, comps := range mainConfig.Stack.Architecture.Regions {
		for _, comp := range comps {
			if len(comp.Apps) > 0 {
				// For components with apps, create a resource for each app
				for _, app := range comp.Apps {
					key := fmt.Sprintf("%s.%s.%s", region, comp.Component, app)
					resources[key] = struct {
						region     string
						component  string
						app        string
						deps       []string
						isDataFlow bool
					}{
						region:     region,
						component:  comp.Component,
						app:        app,
						deps:       mainConfig.Stack.Components[comp.Component].Deps,
						isDataFlow: comp.Component == "rediscache" || comp.Component == "cosmos_db" || comp.Component == "servicebus",
					}
				}
			} else {
				// For components without apps, create a single resource
				key := fmt.Sprintf("%s.%s", region, comp.Component)
				resources[key] = struct {
					region     string
					component  string
					app        string
					deps       []string
					isDataFlow bool
				}{
					region:     region,
					component:  comp.Component,
					app:        "",
					deps:       mainConfig.Stack.Components[comp.Component].Deps,
					isDataFlow: comp.Component == "rediscache" || comp.Component == "cosmos_db" || comp.Component == "servicebus",
				}
			}
		}
	}

	// Create region subgraphs
	for region := range mainConfig.Stack.Architecture.Regions {
		// Create a more readable region label
		regionLabel := strings.Title(strings.ReplaceAll(region, "_", " ")) // e.g., "eastus2" -> "East Us 2"
		diagram.WriteString(fmt.Sprintf("rectangle \"%s\" as %s <<region>> {\n", regionLabel, region))

		// Add resources for this region
		for key, res := range resources {
			if res.region == region {
				sprite := azureSprites[res.component]
				if sprite == "" {
					sprite = "AzureAppService" // default sprite
				}

				resourceId := key
				displayName := res.component
				if res.app != "" {
					displayName = res.app
				}

				// Add the resource with proper three-parameter format
				diagram.WriteString(fmt.Sprintf("  %s(\"%s\", \"%s\", \"%s\")\n",
					sprite,
					resourceId,
					strings.Title(strings.ReplaceAll(displayName, "_", " ")), // Capitalize words
					region))

				// Get component configuration
				component := mainConfig.Stack.Components[res.component]
				resourceType := getResourceTypeAbbreviation(res.component)

				// Add note with resource details
				diagram.WriteString(fmt.Sprintf("  note right of \"%s\"\n", resourceId))
				diagram.WriteString(fmt.Sprintf("    Resource Type: %s\n", component.Source))
				diagram.WriteString(fmt.Sprintf("    Provider Version: %s\n", component.Version))

				// Get region and environment prefixes for example
				regionPrefix := getRegionPrefix(res.region)
				envPrefix := getEnvironmentPrefix(envName)

				if res.app != "" {
					// Resource with app
					diagram.WriteString("    Naming Pattern: {project}-{region}{env}-{resourcetype}-{app}\n")
					diagram.WriteString(fmt.Sprintf("    Example: %s-%s%s-%s-%s\n",
						tgsConfig.Name,
						regionPrefix,
						envPrefix,
						resourceType,
						res.app))
				} else {
					// Resource without app
					diagram.WriteString("    Naming Pattern: {project}-{region}{env}-{resourcetype}\n")
					diagram.WriteString(fmt.Sprintf("    Example: %s-%s%s-%s\n",
						tgsConfig.Name,
						regionPrefix,
						envPrefix,
						resourceType))
				}

				diagram.WriteString("  end note\n\n")
			}
		}
		diagram.WriteString("}\n\n")
	}

	// Add dependencies between resources
	for key, res := range resources {
		for _, dep := range res.deps {
			parts := strings.Split(dep, ".")
			if len(parts) >= 2 {
				depRegion := parts[0]
				depComp := parts[1]
				depApp := ""
				if len(parts) > 2 {
					depApp = parts[2]
				}

				// Handle {region} placeholder
				if depRegion == "{region}" {
					depRegion = res.region
				}

				// Construct the dependency key
				var depKey string
				if depApp != "" && depApp != "{app}" {
					depKey = fmt.Sprintf("%s.%s.%s", depRegion, depComp, depApp)
				} else if depApp == "{app}" && res.app != "" {
					depKey = fmt.Sprintf("%s.%s.%s", depRegion, depComp, res.app)
				} else {
					depKey = fmt.Sprintf("%s.%s", depRegion, depComp)
				}

				// Check if the dependency exists
				if _, exists := resources[depKey]; exists {
					if res.isDataFlow {
						// Data flow dependency (dotted line)
						diagram.WriteString(fmt.Sprintf("  \"%s\" ..> \"%s\" : data flow\n", key, depKey))
					} else {
						// Provisioning dependency (solid line)
						diagram.WriteString(fmt.Sprintf("  \"%s\" --> \"%s\" : depends on\n", key, depKey))
					}
				}
			}
		}
	}

	// End the diagram
	diagram.WriteString("\n@enduml\n")

	// Write the diagram to a file in the .infrastructure/diagrams directory
	outputDir := filepath.Join(".infrastructure", "diagrams")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create diagrams directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.puml", stackName, envName))
	if err := os.WriteFile(outputPath, []byte(diagram.String()), 0644); err != nil {
		return fmt.Errorf("failed to write diagram file: %w", err)
	}

	logger.Info("Generated diagram at: %s", outputPath)
	return nil
}

// Helper function to get region prefix
func getRegionPrefix(region string) string {
	return scaffold.GetRegionPrefix(region)
}

// Helper function to get environment prefix
func getEnvironmentPrefix(env string) string {
	envPrefixMap := map[string]string{
		"dev":   "D",
		"test":  "T",
		"stage": "S",
		"prod":  "P",
		"qa":    "Q",
		"uat":   "U",
	}

	if prefix, ok := envPrefixMap[env]; ok {
		return prefix
	}
	return strings.ToUpper(env[0:1])
}

// Helper function to get resource type abbreviation
func getResourceTypeAbbreviation(resourceType string) string {
	resourceAbbreviations := map[string]string{
		"serviceplan":    "svcpln",
		"appservice":     "appsvc",
		"functionapp":    "fncapp",
		"rediscache":     "cache",
		"keyvault":       "kv",
		"servicebus":     "sbus",
		"cosmos_account": "cosmos",
		"cosmos_db":      "cdb",
		"apim":           "apim",
		"storage":        "st",
		"sql_server":     "sql",
		"sql_database":   "sqldb",
		"eventhub":       "evhub",
		"loganalytics":   "log",
	}

	if abbr, ok := resourceAbbreviations[resourceType]; ok {
		return abbr
	}

	if len(resourceType) > 3 {
		return resourceType[:3]
	}
	return resourceType
}

// Helper function to sanitize IDs for PlantUML
func sanitizeId(id string) string {
	return strings.ReplaceAll(strings.ReplaceAll(id, "-", "_"), ".", "_")
}
