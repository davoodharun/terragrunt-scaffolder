package diagram

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

// Azure resource type to Mermaid icon mapping
var azureIcons = map[string]string{
	"appservice":     "🌐",
	"serviceplan":    "📋",
	"rediscache":     "⚡",
	"cosmos_account": "🌌",
	"cosmos_db":      "🌌",
	"servicebus":     "🚌",
	"keyvault":       "🔑",
	"storage":        "💾",
	"functionapp":    "⚡",
	"apim":           "🔌",
	"sql_server":     "🗄️",
	"sql_database":   "📚",
	"eventhub":       "📡",
	"loganalytics":   "📊",
}

// generateMermaidDiagram generates a Mermaid diagram for a specific stack and environment
func abbr(s string) string {
	switch strings.ToLower(s) {
	case "nonprod":
		return "np"
	case "prod":
		return "p"
	case "dev":
		return "d"
	case "test":
		return "t"
	case "stage":
		return "s"
	case "eastus2":
		return "e2"
	case "westus2":
		return "w2"
	}
	if len(s) > 2 {
		return s[:2]
	}
	return s
}

func nodeID(component, sub, region, env, app string) string {
	id := component
	if app != "" {
		id = app
	}
	return fmt.Sprintf("%s_%s_%s_%s", id, abbr(sub), abbr(region), abbr(env))
}

func generateMermaidDiagram(stackName string, tgsConfig *config.TGSConfig, envName string) error {
	logger.Info("Generating Mermaid diagram for stack %s, environment %s", stackName, envName)

	mainConfig, err := readStackConfig(stackName)
	if err != nil {
		return fmt.Errorf("failed to read stack config: %w", err)
	}

	outputDir := filepath.Join(".infrastructure", "diagrams")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create diagrams directory: %w", err)
	}

	var diagram strings.Builder
	diagram.WriteString("```mermaid\n")
	diagram.WriteString("graph TD\n\n")

	nodeMap := make(map[string]struct {
		sub, region, env, component, app string
		deps                             []string
		isDataFlow                       bool
	})

	// Only include subscriptions and regions for this environment
	for subName, sub := range tgsConfig.Subscriptions {
		// Only include this subscription if it has the target environment for this stack
		foundEnv := false
		for _, env := range sub.Environments {
			stackMatch := stackName
			if env.Stack != "" {
				stackMatch = env.Stack
			}
			if env.Name == envName && stackMatch == stackName {
				foundEnv = true
				break
			}
		}
		if !foundEnv {
			continue
		}
		diagram.WriteString(fmt.Sprintf("  subgraph %s\n", subName))
		for _, env := range sub.Environments {
			stackMatch := stackName
			if env.Stack != "" {
				stackMatch = env.Stack
			}
			if env.Name != envName || stackMatch != stackName {
				continue
			}
			for region, comps := range mainConfig.Stack.Architecture.Regions {
				label := fmt.Sprintf("%s_%s", region, env.Name)
				diagram.WriteString(fmt.Sprintf("    subgraph %s [%s - %s]\n", label, region, env.Name))
				for _, comp := range comps {
					if len(comp.Apps) > 0 {
						for _, app := range comp.Apps {
							id := nodeID(comp.Component, subName, region, env.Name, app)
							icon := getMermaidIcon(comp.Component)
							label := app
							diagram.WriteString(fmt.Sprintf("      %s[\"%s %s\"]:::azure\n", id, icon, label))
							nodeMap[id] = struct {
								sub, region, env, component, app string
								deps                             []string
								isDataFlow                       bool
							}{subName, region, env.Name, comp.Component, app, mainConfig.Stack.Components[comp.Component].Deps, comp.Component == "rediscache" || comp.Component == "cosmos_db" || comp.Component == "servicebus"}
						}
					} else {
						id := nodeID(comp.Component, subName, region, env.Name, "")
						icon := getMermaidIcon(comp.Component)
						label := comp.Component
						diagram.WriteString(fmt.Sprintf("      %s[\"%s %s\"]:::azure\n", id, icon, label))
						nodeMap[id] = struct {
							sub, region, env, component, app string
							deps                             []string
							isDataFlow                       bool
						}{subName, region, env.Name, comp.Component, "", mainConfig.Stack.Components[comp.Component].Deps, comp.Component == "rediscache" || comp.Component == "cosmos_db" || comp.Component == "servicebus"}
					}
				}
				diagram.WriteString("    end\n")
			}
		}
		diagram.WriteString("  end\n\n")
	}

	// Draw dependencies only between nodes present in this environment
	for id, n := range nodeMap {
		for _, dep := range n.deps {
			parts := strings.Split(dep, ".")
			if len(parts) >= 2 {
				depRegion := parts[0]
				depComp := parts[1]
				depApp := ""
				if len(parts) > 2 {
					depApp = parts[2]
				}
				if depRegion == "{region}" {
					depRegion = n.region
				}
				depID := ""
				if depApp != "" && depApp != "{app}" {
					depID = nodeID(depComp, n.sub, depRegion, n.env, depApp)
				} else if depApp == "{app}" && n.app != "" {
					depID = nodeID(depComp, n.sub, depRegion, n.env, n.app)
				} else {
					depID = nodeID(depComp, n.sub, depRegion, n.env, "")
				}
				if _, exists := nodeMap[depID]; exists {
					if n.isDataFlow {
						diagram.WriteString(fmt.Sprintf("%s -.-> %s\n", id, depID))
					} else {
						diagram.WriteString(fmt.Sprintf("%s --> %s\n", id, depID))
					}
				}
			}
		}
	}

	diagram.WriteString("\nclassDef azure fill:#0072C6,stroke:#0072C6,color:white\n\n")
	diagram.WriteString("```\n")

	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.md", stackName, envName))
	if err := os.WriteFile(outputPath, []byte(diagram.String()), 0644); err != nil {
		return fmt.Errorf("failed to write diagram file: %w", err)
	}

	return nil
}

// getMermaidIcon returns the appropriate Mermaid icon for an Azure resource type
func getMermaidIcon(resourceType string) string {
	if icon, ok := azureIcons[resourceType]; ok {
		return icon
	}
	return "📦" // Default icon for unknown resource types
}
