package diagram

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"gopkg.in/yaml.v3"
)

// GenerateDiagram generates Mermaid diagrams for all stacks
func GenerateDiagram() error {
	logger.Info("Generating infrastructure diagrams")

	// Read TGS config to get subscription and environment structure
	tgsConfig, err := readTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Get all stack files
	stacksDir := filepath.Join(".tgs", "stacks")
	entries, err := os.ReadDir(stacksDir)
	if err != nil {
		return fmt.Errorf("failed to read stacks directory: %w", err)
	}

	// Create diagrams directory if it doesn't exist
	if err := os.MkdirAll("diagrams", 0755); err != nil {
		return fmt.Errorf("failed to create diagrams directory: %w", err)
	}

	// Generate diagram for each stack
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			stackName := strings.TrimSuffix(entry.Name(), ".yaml")
			if err := generateStackDiagram(stackName, tgsConfig); err != nil {
				return fmt.Errorf("failed to generate diagram for stack %s: %w", stackName, err)
			}
			// Generate simplified diagram for the dev environment
			if err := generateSimplifiedDiagram(stackName, tgsConfig); err != nil {
				return fmt.Errorf("failed to generate simplified diagram for stack %s: %w", stackName, err)
			}
		}
	}

	// Generate overview diagram showing all stacks
	if err := generateOverviewDiagram(entries); err != nil {
		return fmt.Errorf("failed to generate overview diagram: %w", err)
	}

	logger.Info("Generated infrastructure diagrams in diagrams/ directory")
	return nil
}

// generateOverviewDiagram creates a diagram showing all stacks and their relationships
func generateOverviewDiagram(entries []os.DirEntry) error {
	var diagram strings.Builder
	diagram.WriteString("```mermaid\ngraph TB\n")
	diagram.WriteString("  root[Terragrunt Infrastructure]\n")
	diagram.WriteString("  stacks[Stacks]\n")
	diagram.WriteString("  root --> stacks\n")

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			stackName := strings.TrimSuffix(entry.Name(), ".yaml")
			diagram.WriteString(fmt.Sprintf("  stack_%s[%s]\n", stackName, stackName))
			diagram.WriteString(fmt.Sprintf("  stacks --> stack_%s\n", stackName))
		}
	}

	diagram.WriteString("\n```")

	// Write the overview diagram
	outputPath := filepath.Join("diagrams", "overview.md")
	if err := os.WriteFile(outputPath, []byte(diagram.String()), 0644); err != nil {
		return fmt.Errorf("failed to write overview diagram: %w", err)
	}

	return nil
}

// generateStackDiagram generates a diagram for a specific stack
func generateStackDiagram(stackName string, tgsConfig *config.TGSConfig) error {
	logger.Info("Generating diagram for stack: %s", stackName)

	// Read stack config
	mainConfig, err := readStackConfig(stackName)
	if err != nil {
		return fmt.Errorf("failed to read stack config: %w", err)
	}

	// Start building the Mermaid diagram
	var diagram strings.Builder
	diagram.WriteString("```mermaid\ngraph TB\n")

	// Add project root
	diagram.WriteString("  root[.infrastructure]")

	// Add subscriptions
	for subName := range tgsConfig.Subscriptions {
		subNode := fmt.Sprintf("sub_%s", subName)
		diagram.WriteString(fmt.Sprintf("\n  %s[%s]", subNode, subName))
		diagram.WriteString(fmt.Sprintf("\n  root --> %s", subNode))

		// Add regions from architecture
		for region := range mainConfig.Stack.Architecture.Regions {
			regionNode := fmt.Sprintf("reg_%s_%s", subName, region)
			diagram.WriteString(fmt.Sprintf("\n  %s[%s]", regionNode, region))
			diagram.WriteString(fmt.Sprintf("\n  %s --> %s", subNode, regionNode))

			// Add environments
			for _, env := range tgsConfig.Subscriptions[subName].Environments {
				envNode := fmt.Sprintf("env_%s_%s_%s", subName, region, env.Name)
				diagram.WriteString(fmt.Sprintf("\n  %s[%s]", envNode, env.Name))
				diagram.WriteString(fmt.Sprintf("\n  %s --> %s", regionNode, envNode))

				// Add components in this environment
				for _, comp := range mainConfig.Stack.Architecture.Regions[region] {
					compNode := fmt.Sprintf("comp_%s_%s_%s_%s", subName, region, env.Name, comp.Component)
					diagram.WriteString(fmt.Sprintf("\n  %s[%s]", compNode, comp.Component))
					diagram.WriteString(fmt.Sprintf("\n  %s --> %s", envNode, compNode))

					// Add apps if any
					for _, app := range comp.Apps {
						appNode := fmt.Sprintf("app_%s_%s_%s_%s_%s", subName, region, env.Name, comp.Component, app)
						diagram.WriteString(fmt.Sprintf("\n  %s[%s]", appNode, app))
						diagram.WriteString(fmt.Sprintf("\n  %s --> %s", compNode, appNode))
					}
				}
			}
		}
	}

	// Add component dependencies
	diagram.WriteString("\n\n  %% Component Dependencies")
	for compName, comp := range mainConfig.Stack.Components {
		for _, dep := range comp.Deps {
			parts := strings.Split(dep, ".")
			if len(parts) >= 2 {
				region := parts[0]
				depComp := parts[1]
				app := ""
				if len(parts) > 2 {
					app = parts[2]
				}

				// Replace {region} placeholder
				if region == "{region}" {
					// Add dependency for each region
					for region := range mainConfig.Stack.Architecture.Regions {
						addDependencyToMermaid(&diagram, region, compName, depComp, app)
					}
				} else {
					addDependencyToMermaid(&diagram, region, compName, depComp, app)
				}
			}
		}
	}

	diagram.WriteString("\n```")

	// Write the diagram to a file
	outputPath := filepath.Join("diagrams", fmt.Sprintf("%s.md", stackName))
	if err := os.WriteFile(outputPath, []byte(diagram.String()), 0644); err != nil {
		return fmt.Errorf("failed to write diagram file: %w", err)
	}

	return nil
}

func addDependencyToMermaid(diagram *strings.Builder, region, component, depComponent, app string) {
	// Create nodes for the dependency relationship
	fromNode := fmt.Sprintf("comp_nonprod_%s_dev_%s", region, component)
	toNode := fmt.Sprintf("comp_nonprod_%s_dev_%s", region, depComponent)

	if app != "" && app != "{app}" {
		toNode = fmt.Sprintf("app_nonprod_%s_dev_%s_%s", region, depComponent, app)
	}

	diagram.WriteString(fmt.Sprintf("\n  %s -.-> %s", fromNode, toNode))
}

// readStackConfig reads a specific stack configuration
func readStackConfig(stackName string) (*config.MainConfig, error) {
	stacksDir := filepath.Join(".tgs", "stacks")
	stackPath := filepath.Join(stacksDir, fmt.Sprintf("%s.yaml", stackName))

	data, err := os.ReadFile(stackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read stack file %s: %w", stackPath, err)
	}

	var cfg config.MainConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse stack file %s: %w", stackPath, err)
	}

	return &cfg, nil
}

// readTGSConfig reads the tgs.yaml configuration
func readTGSConfig() (*config.TGSConfig, error) {
	configDir := ".tgs"
	configPath := filepath.Join(configDir, "tgs.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tgs.yaml: %w", err)
	}

	var cfg config.TGSConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse tgs.yaml: %w", err)
	}

	return &cfg, nil
}

// generateSimplifiedDiagram creates a clean, focused diagram for the dev environment
func generateSimplifiedDiagram(stackName string, tgsConfig *config.TGSConfig) error {
	logger.Info("Generating simplified diagram for stack: %s", stackName)

	// Read stack config
	mainConfig, err := readStackConfig(stackName)
	if err != nil {
		return fmt.Errorf("failed to read stack config: %w", err)
	}

	// Start building the Mermaid diagram
	var diagram strings.Builder
	diagram.WriteString("```mermaid\nflowchart LR\n")
	diagram.WriteString("  %% Styling\n")
	diagram.WriteString("  classDef app fill:#f9f,stroke:#333,stroke-width:2px\n")
	diagram.WriteString("  classDef component fill:#bbf,stroke:#333,stroke-width:2px\n")
	diagram.WriteString("  classDef infra fill:#ddd,stroke:#333,stroke-width:2px\n\n")

	// Create subgraph for each region in nonprod/dev
	for region := range mainConfig.Stack.Architecture.Regions {
		diagram.WriteString(fmt.Sprintf("  subgraph %s[%s Region]\n", region, region))
		diagram.WriteString("    direction TB\n")

		// Build dependency graph
		dependencies := make(map[string][]string)
		components := make(map[string]bool)

		// First pass: collect all components and their dependencies
		for _, comp := range mainConfig.Stack.Architecture.Regions[region] {
			components[comp.Component] = true
			if _, exists := dependencies[comp.Component]; !exists {
				dependencies[comp.Component] = []string{}
			}
		}

		// Second pass: build dependency graph
		for compName, comp := range mainConfig.Stack.Components {
			if !components[compName] {
				continue
			}

			for _, dep := range comp.Deps {
				parts := strings.Split(dep, ".")
				if len(parts) >= 2 {
					depRegion := parts[0]
					depComp := parts[1]

					// Only process dependencies within the same region
					if (depRegion == "{region}" || depRegion == region) && components[depComp] {
						dependencies[compName] = append(dependencies[compName], depComp)
					}
				}
			}
		}

		// Find components with no dependencies (roots)
		var roots []string
		for comp := range components {
			if len(dependencies[comp]) == 0 {
				roots = append(roots, comp)
			}
		}

		// Topologically sort components
		var sortedComps []string
		visited := make(map[string]bool)
		var visit func(string)
		visit = func(comp string) {
			if visited[comp] {
				return
			}
			visited[comp] = true
			for _, dep := range dependencies[comp] {
				visit(dep)
			}
			sortedComps = append(sortedComps, comp)
		}

		for _, root := range roots {
			visit(root)
		}

		// Add remaining components (in case of cycles)
		for comp := range components {
			if !visited[comp] {
				sortedComps = append(sortedComps, comp)
			}
		}

		// Reverse the sorted list to get dependency order
		for i := len(sortedComps)/2 - 1; i >= 0; i-- {
			opp := len(sortedComps) - 1 - i
			sortedComps[i], sortedComps[opp] = sortedComps[opp], sortedComps[i]
		}

		// Add components in dependency order with their apps
		prevLevel := make(map[string]int)
		maxLevel := 0

		// First determine levels based on dependencies
		for _, comp := range sortedComps {
			level := 0
			for _, dep := range dependencies[comp] {
				if prevLevel[dep] >= level {
					level = prevLevel[dep] + 1
				}
			}
			prevLevel[comp] = level
			if level > maxLevel {
				maxLevel = level
			}
		}

		// Create ranks to enforce ordering
		for level := 0; level <= maxLevel; level++ {
			diagram.WriteString(fmt.Sprintf("    subgraph level_%d[\" \"]\n", level))
			for _, comp := range sortedComps {
				if prevLevel[comp] == level {
					// Create subgraph for component and its apps
					compId := fmt.Sprintf("%s_%s", region, comp)
					diagram.WriteString(fmt.Sprintf("      subgraph %s[%s]\n", compId, comp))
					diagram.WriteString("        direction TB\n")

					// Find apps for this component
					for _, archComp := range mainConfig.Stack.Architecture.Regions[region] {
						if archComp.Component == comp {
							for _, app := range archComp.Apps {
								appId := fmt.Sprintf("%s_%s_%s", region, comp, app)
								diagram.WriteString(fmt.Sprintf("        %s[%s]:::app\n", appId, app))
							}
							break
						}
					}

					diagram.WriteString("      end\n")
					diagram.WriteString(fmt.Sprintf("      style %s fill:#bbf,stroke:#333,stroke-width:2px\n", compId))
				}
			}
			diagram.WriteString("    end\n")
		}

		// Add dependencies between components
		for compName, deps := range dependencies {
			fromId := fmt.Sprintf("%s_%s", region, compName)
			for _, dep := range deps {
				toId := fmt.Sprintf("%s_%s", region, dep)
				diagram.WriteString(fmt.Sprintf("    %s --> %s\n", toId, fromId))
			}
		}

		// Add data flow dependencies for apps
		for compName, comp := range mainConfig.Stack.Components {
			if !components[compName] {
				continue
			}

			for _, dep := range comp.Deps {
				parts := strings.Split(dep, ".")
				if len(parts) >= 2 {
					depRegion := parts[0]
					depComp := parts[1]
					depApp := ""
					if len(parts) > 2 {
						depApp = parts[2]
					}

					if depRegion == "{region}" || depRegion == region {
						if depApp == "{app}" {
							// Find apps in the source component
							for _, archComp := range mainConfig.Stack.Architecture.Regions[region] {
								if archComp.Component == compName {
									for _, app := range archComp.Apps {
										fromId := fmt.Sprintf("%s_%s_%s", region, compName, app)
										toId := fmt.Sprintf("%s_%s_%s", region, depComp, app)
										diagram.WriteString(fmt.Sprintf("    %s -.->|data flow| %s\n", fromId, toId))
									}
								}
							}
						} else if depApp != "" {
							fromId := fmt.Sprintf("%s_%s", region, compName)
							toId := fmt.Sprintf("%s_%s_%s", region, depComp, depApp)
							diagram.WriteString(fmt.Sprintf("    %s -.->|data flow| %s\n", fromId, toId))
						}
					}
				}
			}
		}

		diagram.WriteString("  end\n\n")
	}

	diagram.WriteString("\n```\n\n")

	// Add legend
	diagram.WriteString("### Legend\n\n")
	diagram.WriteString("- 🟦 Blue boxes represent infrastructure components\n")
	diagram.WriteString("- 🟪 Purple boxes represent applications within components\n")
	diagram.WriteString("- ➡️ Solid arrows show provisioning order (A → B means A must be provisioned before B)\n")
	diagram.WriteString("- 〰️ Dotted arrows show data flow between applications\n")

	// Write the diagram to a file
	outputPath := filepath.Join("diagrams", fmt.Sprintf("%s_simplified.md", stackName))
	if err := os.WriteFile(outputPath, []byte(diagram.String()), 0644); err != nil {
		return fmt.Errorf("failed to write simplified diagram: %w", err)
	}

	return nil
}
