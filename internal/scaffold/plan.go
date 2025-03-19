package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

type Change struct {
	Type         string // "add", "remove", "modify"
	Category     string // "component", "app", "config", "subscription", "environment"
	Component    string
	App          string
	Region       string
	Environment  string
	Subscription string
	Details      string
}

// Plan analyzes changes that would be applied to the infrastructure
func Plan() error {
	logger.Info("Analyzing infrastructure changes...")

	// Check if .infrastructure directory exists
	if _, err := os.Stat(".infrastructure"); os.IsNotExist(err) {
		return fmt.Errorf("no existing infrastructure found. Use 'generate' to create initial infrastructure")
	}

	// Read TGS config
	tgsConfig, err := ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Track all changes
	var changes []Change

	// Get existing subscriptions from .infrastructure directory
	entries, err := os.ReadDir(".infrastructure")
	if err != nil {
		return fmt.Errorf("failed to read infrastructure directory: %w", err)
	}

	// Track existing and planned subscriptions
	existingSubs := make(map[string]bool)
	plannedSubs := make(map[string]bool)

	// Find existing subscriptions (excluding special directories)
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), "_") && !strings.HasPrefix(entry.Name(), ".") && entry.Name() != "config" && entry.Name() != "diagrams" {
			existingSubs[entry.Name()] = true
		}
	}

	// Process planned subscriptions and their contents
	for subName, sub := range tgsConfig.Subscriptions {
		plannedSubs[subName] = true

		// Check if this is a new subscription
		if !existingSubs[subName] {
			changes = append(changes, Change{
				Type:         "add",
				Category:     "subscription",
				Subscription: subName,
				Details:      fmt.Sprintf("New subscription will be created"),
			})
			continue
		}

		// Get existing environments for this subscription
		subPath := filepath.Join(".infrastructure", subName)
		existingEnvs := make(map[string]map[string]bool) // map[region]map[env]bool

		// Read regions in the subscription
		regions, err := os.ReadDir(subPath)
		if err == nil {
			for _, region := range regions {
				if region.IsDir() {
					regionPath := filepath.Join(subPath, region.Name())
					envs, err := os.ReadDir(regionPath)
					if err == nil {
						if existingEnvs[region.Name()] == nil {
							existingEnvs[region.Name()] = make(map[string]bool)
						}
						for _, env := range envs {
							if env.IsDir() {
								existingEnvs[region.Name()][env.Name()] = true
							}
						}
					}
				}
			}
		}

		// Process each environment with its specified stack
		for _, env := range sub.Environments {
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}

			// Read the stack configuration
			mainConfig, err := readMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			// Compare components and apps in each region
			for region, components := range mainConfig.Stack.Architecture.Regions {
				// Remove environment from existing map to track removals
				if existingEnvs[region] != nil {
					delete(existingEnvs[region], env.Name)
				}

				// Check if this environment exists in this region
				envPath := filepath.Join(".infrastructure", subName, region, env.Name)
				if _, err := os.Stat(envPath); os.IsNotExist(err) {
					changes = append(changes, Change{
						Type:         "add",
						Category:     "environment",
						Subscription: subName,
						Environment:  env.Name,
						Region:       region,
						Details:      fmt.Sprintf("New environment will be created"),
					})
					continue
				}

				// Track planned components
				plannedComponents := make(map[string]bool)

				// Check for new or modified components
				for _, comp := range components {
					plannedComponents[comp.Component] = true

					// Check if component directory exists
					componentPath := filepath.Join(envPath, comp.Component)
					componentExists := false
					if _, err := os.Stat(componentPath); err == nil {
						componentExists = true
					}

					if !componentExists {
						changes = append(changes, Change{
							Type:         "add",
							Category:     "component",
							Component:    comp.Component,
							Region:       region,
							Environment:  env.Name,
							Subscription: subName,
							Details:      fmt.Sprintf("New component will be created"),
						})
						continue
					}

					// Compare apps if component exists
					if len(comp.Apps) > 0 {
						existingApps := make(map[string]bool)
						// Read existing app directories
						entries, err := os.ReadDir(componentPath)
						if err == nil {
							for _, entry := range entries {
								if entry.IsDir() {
									existingApps[entry.Name()] = true
								}
							}
						}

						// Check for new apps
						for _, app := range comp.Apps {
							if !existingApps[app] {
								changes = append(changes, Change{
									Type:         "add",
									Category:     "app",
									Component:    comp.Component,
									App:          app,
									Region:       region,
									Environment:  env.Name,
									Subscription: subName,
									Details:      fmt.Sprintf("New application instance will be created"),
								})
							}
						}

						// Check for removed apps
						for existingApp := range existingApps {
							found := false
							for _, plannedApp := range comp.Apps {
								if plannedApp == existingApp {
									found = true
									break
								}
							}
							if !found {
								changes = append(changes, Change{
									Type:         "remove",
									Category:     "app",
									Component:    comp.Component,
									App:          existingApp,
									Region:       region,
									Environment:  env.Name,
									Subscription: subName,
									Details:      fmt.Sprintf("Application instance will be removed"),
								})
							}
						}
					}

					// Check for configuration changes
					if configChanges := checkComponentConfigChanges(mainConfig.Stack.Components[comp.Component], componentPath); len(configChanges) > 0 {
						for _, detail := range configChanges {
							changes = append(changes, Change{
								Type:         "modify",
								Category:     "config",
								Component:    comp.Component,
								Region:       region,
								Environment:  env.Name,
								Subscription: subName,
								Details:      detail,
							})
						}
					}
				}

				// Check for removed components
				entries, err := os.ReadDir(envPath)
				if err == nil {
					for _, entry := range entries {
						if entry.IsDir() && !plannedComponents[entry.Name()] {
							changes = append(changes, Change{
								Type:         "remove",
								Category:     "component",
								Component:    entry.Name(),
								Region:       region,
								Environment:  env.Name,
								Subscription: subName,
								Details:      fmt.Sprintf("Component will be removed"),
							})
						}
					}
				}
			}
		}

		// Add changes for environments that will be removed
		for region, envs := range existingEnvs {
			for env := range envs {
				changes = append(changes, Change{
					Type:         "remove",
					Category:     "environment",
					Subscription: subName,
					Environment:  env,
					Region:       region,
					Details:      fmt.Sprintf("Environment will be removed"),
				})
			}
		}
	}

	// Check for removed subscriptions
	for existingSub := range existingSubs {
		if !plannedSubs[existingSub] {
			changes = append(changes, Change{
				Type:         "remove",
				Category:     "subscription",
				Subscription: existingSub,
				Details:      fmt.Sprintf("Subscription will be removed"),
			})
		}
	}

	// Print changes
	if len(changes) == 0 {
		fmt.Println("\nNo changes detected. Infrastructure is up to date.")
		return nil
	}

	fmt.Println("\nPlanned changes:")
	fmt.Println("================")

	// Group changes by subscription and environment
	bySubEnvRegion := make(map[string][]Change)
	for _, change := range changes {
		var key string
		if change.Category == "subscription" {
			key = change.Subscription
		} else {
			key = fmt.Sprintf("%s/%s/%s", change.Subscription, change.Environment, change.Region)
		}
		bySubEnvRegion[key] = append(bySubEnvRegion[key], change)
	}

	// Print changes organized by subscription, environment, and region
	for key, changes := range bySubEnvRegion {
		parts := strings.Split(key, "/")
		if len(parts) == 1 {
			// Subscription-level changes
			fmt.Printf("\nSubscription: %s\n", parts[0])
			fmt.Println(strings.Repeat("-", 40))
		} else {
			// Environment-level changes
			sub, env, region := parts[0], parts[1], parts[2]
			fmt.Printf("\nSubscription: %s, Environment: %s, Region: %s\n", sub, env, region)
			fmt.Println(strings.Repeat("-", 40))
		}

		// Group by change type
		for _, changeType := range []string{"add", "remove", "modify"} {
			var typeChanges []Change
			for _, change := range changes {
				if change.Type == changeType {
					typeChanges = append(typeChanges, change)
				}
			}

			if len(typeChanges) > 0 {
				switch changeType {
				case "add":
					fmt.Println("\n  + Additions:")
				case "remove":
					fmt.Println("\n  - Removals:")
				case "modify":
					fmt.Println("\n  ~ Modifications:")
				}

				for _, change := range typeChanges {
					switch change.Category {
					case "subscription":
						fmt.Printf("    Subscription %s: %s\n", change.Subscription, change.Details)
					case "environment":
						fmt.Printf("    Environment %s: %s\n", change.Environment, change.Details)
					case "component":
						fmt.Printf("    %s: %s\n", change.Component, change.Details)
					case "app":
						fmt.Printf("    %s/%s: %s\n", change.Component, change.App, change.Details)
					case "config":
						fmt.Printf("    %s: %s\n", change.Component, change.Details)
					}
				}
			}
		}
	}

	return nil
}

// checkComponentConfigChanges checks for configuration changes in component.hcl and terragrunt.hcl files
func checkComponentConfigChanges(comp config.Component, componentPath string) []string {
	var changes []string

	// Read the existing component.hcl file
	componentHclPath := filepath.Join(componentPath, "component.hcl")
	content, err := os.ReadFile(componentHclPath)
	if err != nil {
		return changes
	}

	currentContent := string(content)

	// Check for version changes
	if comp.Version != "" && !strings.Contains(currentContent, fmt.Sprintf(`version = "%s"`, comp.Version)) {
		changes = append(changes, fmt.Sprintf("Provider version will be updated to %s", comp.Version))
	}

	// Check for dependency changes
	if len(comp.Deps) > 0 {
		missingDeps := false
		for _, dep := range comp.Deps {
			if !strings.Contains(currentContent, fmt.Sprintf(`dependency "%s"`, strings.Split(dep, ".")[1])) {
				missingDeps = true
				break
			}
		}
		if missingDeps {
			changes = append(changes, "Component dependencies will be updated")
		}
	}

	return changes
}
