package diagram

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	diagram.WriteString("graph LR\n\n")

	nodeMap := make(map[string]struct {
		sub, region, env, component, app string
		deps                             []string
		isDataFlow                       bool
	})
	// For uniqueness: map baseID to count
	baseIDCount := make(map[string]int)
	// For reverse lookup: map base info to unique node ID
	type nodeKey struct{ component, sub, region, env, app string }
	nodeKeyToID := make(map[nodeKey]string)

	for subName, sub := range tgsConfig.Subscriptions {
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
		//diagram.WriteString(fmt.Sprintf("  subgraph %s\n", subName))
		for _, env := range sub.Environments {
			stackMatch := stackName
			if env.Stack != "" {
				stackMatch = env.Stack
			}
			if env.Name != envName || stackMatch != stackName {
				continue
			}
			for region, comps := range mainConfig.Stack.Architecture.Regions {
				// label := fmt.Sprintf("%s_%s", region, env.Name)
				// diagram.WriteString(fmt.Sprintf("    subgraph %s [%s - %s]\n", label, region, env.Name))
				for _, comp := range comps {
					if len(comp.Apps) > 0 {
						for _, app := range comp.Apps {
							baseID := nodeID(comp.Component, subName, region, env.Name, "")
							baseIDCount[baseID]++
							uniqueID := baseID
							if baseIDCount[baseID] > 1 {
								uniqueID = fmt.Sprintf("%s_%d", baseID, baseIDCount[baseID])
							}
							// icon := getMermaidIcon(comp.Component)
							// label := fmt.Sprintf("%s (%s)", app, comp.Component)
							// diagram.WriteString(fmt.Sprintf("      %s[\"%s %s\"]:::azure\n", uniqueID, icon, label))
							nodeMap[uniqueID] = struct {
								sub, region, env, component, app string
								deps                             []string
								isDataFlow                       bool
							}{subName, region, env.Name, comp.Component, app, mainConfig.Stack.Components[comp.Component].Deps, comp.Component == "rediscache" || comp.Component == "cosmos_db" || comp.Component == "servicebus"}
							nodeKeyToID[nodeKey{comp.Component, subName, region, env.Name, app}] = uniqueID
						}
					} else {
						baseID := nodeID(comp.Component, subName, region, env.Name, "")
						baseIDCount[baseID]++
						uniqueID := baseID
						if baseIDCount[baseID] > 1 {
							uniqueID = fmt.Sprintf("%s_%d", baseID, baseIDCount[baseID])
						}
						// icon := getMermaidIcon(comp.Component)
						// label := fmt.Sprintf("%s (%s)", comp.Component, comp.Component)
						// diagram.WriteString(fmt.Sprintf("      %s[\"%s %s\"]:::azure\n", uniqueID, icon, label))
						nodeMap[uniqueID] = struct {
							sub, region, env, component, app string
							deps                             []string
							isDataFlow                       bool
						}{subName, region, env.Name, comp.Component, "", mainConfig.Stack.Components[comp.Component].Deps, comp.Component == "rediscache" || comp.Component == "cosmos_db" || comp.Component == "servicebus"}
						nodeKeyToID[nodeKey{comp.Component, subName, region, env.Name, ""}] = uniqueID
					}
				}
				//diagram.WriteString("    end\n")
			}
		}
		//diagram.WriteString("  end\n\n")
	}

	// --- Deployment Order for Diagram ---
	// Build dependency graph for diagram (reuse logic from deployment order section)
	type nodeInfo struct {
		id, displayName string
		deps            []string
		wave            int
	}

	graph := make(map[string]*nodeInfo)
	idToDeps := make(map[string][]string)
	idToDisplay := make(map[string]string)
	for id, n := range nodeMap {
		displayName := ""
		if n.app != "" {
			displayName = fmt.Sprintf("%s (%s, %s)", n.app, n.component, n.region)
		} else {
			displayName = fmt.Sprintf("%s (%s, %s)", n.component, n.component, n.region)
		}
		var depIDs []string
		for _, dep := range n.deps {
			parts := strings.Split(dep, ".")
			if len(parts) >= 2 {
				depRegion := parts[0]
				if depRegion == "{region}" {
					depRegion = n.region
				}
				depComp := parts[1]
				depApp := ""
				if len(parts) > 2 {
					depApp = parts[2]
					if depApp == "{app}" {
						depApp = n.app
					}
				}
				// Find the node ID for this dependency
				depID := ""
				for depNodeID, depNode := range nodeMap {
					if depNode.component == depComp && depNode.region == depRegion && depNode.env == n.env && depNode.sub == n.sub {
						if depApp != "" && depApp != "{app}" {
							if depNode.app == depApp {
								depID = depNodeID
								break
							}
						} else if depApp == "" && depNode.app == "" {
							depID = depNodeID
							break
						}
					}
				}
				if depID != "" {
					depIDs = append(depIDs, depID)
				}
			}
		}
		graph[id] = &nodeInfo{id, displayName, depIDs, -1}
		idToDeps[id] = depIDs
		idToDisplay[id] = displayName
	}
	// Robust wave assignment: assign wave 0 to nodes with no dependencies, others get 1+max(dep wave)
	for id := range graph {
		graph[id].wave = -1
	}
	changed := true
	for changed {
		changed = false
		for id, n := range graph {
			if n.wave != -1 {
				continue
			}
			if len(idToDeps[id]) == 0 {
				n.wave = 0
				changed = true
			} else {
				maxDepWave := -1
				allDepsSet := true
				for _, dep := range idToDeps[id] {
					if graph[dep].wave == -1 {
						allDepsSet = false
						break
					}
					if graph[dep].wave > maxDepWave {
						maxDepWave = graph[dep].wave
					}
				}
				if allDepsSet {
					n.wave = maxDepWave + 1
					changed = true
				}
			}
		}
	}

	// Group nodes by wave
	waves := [][]string{}
	maxWave := -1
	for _, n := range graph {
		if n.wave > maxWave {
			maxWave = n.wave
		}
	}
	for i := 0; i <= maxWave; i++ {
		waves = append(waves, []string{})
	}
	for id, n := range graph {
		waves[n.wave] = append(waves[n.wave], id)
	}
	// Sort each wave for deterministic output
	for i := range waves {
		sort.Strings(waves[i])
	}

	// Output nodes grouped by deployment wave (column), grouped by component type, using subgraphs for each wave
	for waveIdx, wave := range waves {
		if len(wave) == 0 {
			continue
		}
		diagram.WriteString(fmt.Sprintf("  subgraph Wave %d\n", waveIdx+1))
		compTypeToNodes := make(map[string][]string)
		for _, id := range wave {
			n := nodeMap[id]
			compTypeToNodes[n.component] = append(compTypeToNodes[n.component], id)
		}
		for _, ids := range compTypeToNodes {
			for _, id := range ids {
				n := nodeMap[id]
				icon := getMermaidIcon(n.component)
				label := ""
				if n.app != "" {
					label = fmt.Sprintf("%s (%s, %s)", n.app, n.component, n.region)
				} else {
					label = fmt.Sprintf("%s (%s, %s)", n.component, n.component, n.region)
				}
				diagram.WriteString(fmt.Sprintf("    %s[\"%s %s\"]:::azure\n", id, icon, label))
			}
		}
		diagram.WriteString("  end\n\n")
	}

	// Draw dependency arrows (left to right, dependency to dependent)
	for _, wave := range waves {
		for _, id := range wave {
			n := nodeMap[id]
			for _, dep := range n.deps {
				parts := strings.Split(dep, ".")
				if len(parts) >= 2 {
					depRegion := parts[0]
					if depRegion == "{region}" {
						depRegion = n.region
					}
					depComp := parts[1]
					depApp := ""
					if len(parts) > 2 {
						depApp = parts[2]
						if depApp == "{app}" {
							depApp = n.app
						}
					}
					depID := ""
					for depNodeID, depNode := range nodeMap {
						if depNode.component == depComp && depNode.region == depRegion && depNode.env == n.env && depNode.sub == n.sub {
							if depApp != "" && depApp != "{app}" {
								if depNode.app == depApp {
									depID = depNodeID
									break
								}
							} else if depApp == "" && depNode.app == "" {
								depID = depNodeID
								break
							}
						}
					}
					if depID != "" {
						diagram.WriteString(fmt.Sprintf("%s --> %s\n", depID, id))
					}
				}
			}
		}
	}

	diagram.WriteString("\nclassDef azure fill:#0072C6,stroke:#0072C6,color:white\n")
	diagram.WriteString("classDef invisible fill:#0000,stroke:#0000,color:#0000\n\n")
	diagram.WriteString("```\n")

	// Write dependency summary in Markdown
	diagram.WriteString("\n## Component Dependencies\n\n")
	// Group nodes by component type
	compTypeToNodes := make(map[string][]struct {
		displayName string
		deps        []string
		region      string
		app         string
		component   string
	})
	for _, n := range nodeMap {
		displayName := ""
		if n.app != "" {
			displayName = fmt.Sprintf("%s (%s) [%s]", n.app, n.component, n.region)
		} else {
			displayName = fmt.Sprintf("%s (%s) [%s]", n.component, n.component, n.region)
		}
		compTypeToNodes[n.component] = append(compTypeToNodes[n.component], struct {
			displayName string
			deps        []string
			region      string
			app         string
			component   string
		}{displayName, n.deps, n.region, n.app, n.component})
	}
	// Output grouped summary
	for compType, nodes := range compTypeToNodes {
		diagram.WriteString(fmt.Sprintf("### %s\n", compType))
		for _, n := range nodes {
			if len(n.deps) == 0 {
				diagram.WriteString(fmt.Sprintf("- `%s` has no dependencies\n", n.displayName))
			} else {
				var depNames []string
				for _, dep := range n.deps {
					parts := strings.Split(dep, ".")
					if len(parts) >= 2 {
						depRegion := parts[0]
						if depRegion == "{region}" {
							depRegion = n.region
						}
						depComp := parts[1]
						depApp := ""
						if len(parts) > 2 {
							depApp = parts[2]
							if depApp == "{app}" {
								depApp = n.app
							}
						}
						depCompName := depComp
						if compCfg, ok := mainConfig.Stack.Components[depComp]; ok {
							depCompName = compCfg.Source
						}
						if depApp != "" && depApp != "{app}" {
							depNames = append(depNames, fmt.Sprintf("%s (%s, %s) [%s]", depApp, depComp, depCompName, depRegion))
						} else {
							depNames = append(depNames, fmt.Sprintf("%s (%s, %s) [%s]", depComp, depComp, depCompName, depRegion))
						}
					}
				}
				diagram.WriteString(fmt.Sprintf("- `%s` depends on: %s\n", n.displayName, strings.Join(depNames, ", ")))
			}
		}
		diagram.WriteString("\n")
	}

	// --- Deployment Order Section ---
	diagram.WriteString("\n## Deployment Order\n\n")
	// Output waves
	for i, wave := range waves {
		diagram.WriteString(fmt.Sprintf("**Wave %d:**\n", i+1))
		for _, id := range wave {
			n := nodeMap[id]
			displayName := idToDisplay[id]
			if len(n.deps) == 0 {
				diagram.WriteString(fmt.Sprintf("- `%s` has no dependencies\n", displayName))
			} else {
				var depNames []string
				for _, dep := range n.deps {
					parts := strings.Split(dep, ".")
					if len(parts) >= 2 {
						depRegion := parts[0]
						if depRegion == "{region}" {
							depRegion = n.region
						}
						depComp := parts[1]
						depApp := ""
						if len(parts) > 2 {
							depApp = parts[2]
							if depApp == "{app}" {
								depApp = n.app
							}
						}
						depCompName := depComp
						if compCfg, ok := mainConfig.Stack.Components[depComp]; ok {
							depCompName = compCfg.Source
						}
						if depApp != "" && depApp != "{app}" {
							depNames = append(depNames, fmt.Sprintf("%s (%s, %s) [%s]", depApp, depComp, depCompName, depRegion))
						} else {
							depNames = append(depNames, fmt.Sprintf("%s (%s, %s) [%s]", depComp, depComp, depCompName, depRegion))
						}
					}
				}
				diagram.WriteString(fmt.Sprintf("- `%s` depends on: %s\n", displayName, strings.Join(depNames, ", ")))
			}
		}
		diagram.WriteString("\n")
	}

	diagram.WriteString("\n## Component Summary\n\n")
	for _, n := range nodeMap {
		compConfig, ok := mainConfig.Stack.Components[n.component]
		if !ok {
			continue
		}
		displayName := ""
		if n.app != "" {
			displayName = fmt.Sprintf("%s (%s)", n.app, n.component)
		} else {
			displayName = fmt.Sprintf("%s", n.component)
		}
		diagram.WriteString(fmt.Sprintf("- `%s`\n", displayName))
		diagram.WriteString(fmt.Sprintf("  - Provider: %s (v%s)\n", compConfig.Provider, compConfig.Version))
		diagram.WriteString(fmt.Sprintf("  - Terraform module: %s\n", compConfig.Source))
	}

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
