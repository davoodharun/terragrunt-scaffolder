package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/scaffold"
)

// Component represents a component in the infrastructure
type Component struct {
	Name   string
	Apps   []string
	Region string
	Env    string
	Sub    string
	Deps   []string
	Path   string
}

// Stage represents a pipeline stage
type Stage struct {
	Name       string
	DependsOn  []string
	Template   string
	Parameters map[string]interface{}
}

// Pipeline represents a complete pipeline configuration
type Pipeline struct {
	Name       string
	Stages     []Stage
	Parameters map[string]interface{}
}

// AnalyzeInfrastructure analyzes the .infrastructure folder to build dependency chains
func AnalyzeInfrastructure() (map[string][]Component, error) {
	// Map to store components by environment
	envComponents := make(map[string][]Component)

	// Read TGS config to get subscription and environment structure
	tgsConfig, err := config.ReadTGSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Process each subscription
	for subName, sub := range tgsConfig.Subscriptions {
		// Process each environment
		for _, env := range sub.Environments {
			envName := env.Name
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}

			// Read the stack configuration
			mainConfig, err := config.ReadMainConfig(stackName)
			if err != nil {
				return nil, fmt.Errorf("failed to read stack config %s: %w", stackName, err)
			}

			// Process each region
			for region, components := range mainConfig.Stack.Architecture.Regions {
				for _, comp := range components {
					// Create component instance
					component := Component{
						Name:   comp.Component,
						Apps:   comp.Apps,
						Region: region,
						Env:    envName,
						Sub:    subName,
						Deps:   mainConfig.Stack.Components[comp.Component].Deps,
						Path:   filepath.Join(".infrastructure", subName, region, envName, comp.Component),
					}

					// Add to environment components
					envComponents[envName] = append(envComponents[envName], component)
				}
			}
		}
	}

	return envComponents, nil
}

// BuildDependencyChain builds the dependency chain for components in an environment
func BuildDependencyChain(components []Component) []Stage {
	// Map to store stages by component name
	stages := make(map[string]*Stage)

	// First pass: create stages for all components
	for _, comp := range components {
		// Create stage for each app if specified
		if len(comp.Apps) > 0 {
			for _, app := range comp.Apps {
				stageName := fmt.Sprintf("%s_%s_%s", comp.Region, comp.Name, app)
				stages[stageName] = &Stage{
					Name:      stageName,
					DependsOn: []string{},
					Template:  "component-deploy.yml",
					Parameters: map[string]interface{}{
						"component": comp.Name,
						"app":       app,
						"region":    comp.Region,
						"env":       comp.Env,
						"sub":       comp.Sub,
					},
				}
			}
		} else {
			// Create stage for component without apps
			stageName := fmt.Sprintf("%s_%s", comp.Region, comp.Name)
			stages[stageName] = &Stage{
				Name:      stageName,
				DependsOn: []string{},
				Template:  "component-deploy.yml",
				Parameters: map[string]interface{}{
					"component": comp.Name,
					"region":    comp.Region,
					"env":       comp.Env,
					"sub":       comp.Sub,
				},
			}
		}
	}

	// Second pass: add dependencies
	for _, comp := range components {
		for _, dep := range comp.Deps {
			// Parse dependency path
			parts := strings.Split(dep, ".")
			if len(parts) < 2 {
				continue
			}

			region := parts[0]
			depComp := parts[1]
			app := ""
			if len(parts) > 2 {
				app = parts[2]
			}

			// Handle special placeholders
			if region == "{region}" {
				region = comp.Region
			}
			if app == "{app}" {
				// Add dependency for each app of the component
				for _, compApp := range comp.Apps {
					depStageName := fmt.Sprintf("%s_%s_%s", region, depComp, compApp)
					if depStage, ok := stages[depStageName]; ok {
						stageName := fmt.Sprintf("%s_%s_%s", comp.Region, comp.Name, compApp)
						if stage, ok := stages[stageName]; ok {
							stage.DependsOn = append(stage.DependsOn, depStage.Name)
						}
					}
				}
			} else if app == "" {
				// Add dependency for component without apps
				depStageName := fmt.Sprintf("%s_%s", region, depComp)
				if depStage, ok := stages[depStageName]; ok {
					if len(comp.Apps) > 0 {
						// Add dependency to all app stages
						for _, compApp := range comp.Apps {
							stageName := fmt.Sprintf("%s_%s_%s", comp.Region, comp.Name, compApp)
							if stage, ok := stages[stageName]; ok {
								stage.DependsOn = append(stage.DependsOn, depStage.Name)
							}
						}
					} else {
						// Add dependency to component stage
						stageName := fmt.Sprintf("%s_%s", comp.Region, comp.Name)
						if stage, ok := stages[stageName]; ok {
							stage.DependsOn = append(stage.DependsOn, depStage.Name)
						}
					}
				}
			} else {
				// Add dependency for specific app
				depStageName := fmt.Sprintf("%s_%s_%s", region, depComp, app)
				if depStage, ok := stages[depStageName]; ok {
					if len(comp.Apps) > 0 {
						// Add dependency to all app stages
						for _, compApp := range comp.Apps {
							stageName := fmt.Sprintf("%s_%s_%s", comp.Region, comp.Name, compApp)
							if stage, ok := stages[stageName]; ok {
								stage.DependsOn = append(stage.DependsOn, depStage.Name)
							}
						}
					} else {
						// Add dependency to component stage
						stageName := fmt.Sprintf("%s_%s", comp.Region, comp.Name)
						if stage, ok := stages[stageName]; ok {
							stage.DependsOn = append(stage.DependsOn, depStage.Name)
						}
					}
				}
			}
		}
	}

	// Convert stages map to slice
	var result []Stage
	for _, stage := range stages {
		result = append(result, *stage)
	}

	return result
}

// generateStackTemplate generates a deployment template for a specific stack
func generateStackTemplate(stackName string, mainConfig *config.MainConfig) error {
	// Create templates directory if it doesn't exist
	if err := os.MkdirAll(".azure-pipelines/templates", 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Generate the stack template content
	template := fmt.Sprintf(`# Stack deployment template for %s
parameters:
  - name: environment
    type: string
  - name: subscription
    type: string
  - name: runMode
    type: string
    default: plan
    values:
      - plan
      - apply
      - destroy

stages:
`, stackName)

	// Group components by region
	regionComponents := make(map[string][]string)
	for region, components := range mainConfig.Stack.Architecture.Regions {
		for _, comp := range components {
			regionComponents[region] = append(regionComponents[region], comp.Component)
		}
	}

	// Add stages for each region's components
	for region, components := range regionComponents {
		regionPrefix := scaffold.GetRegionPrefix(region)
		template += fmt.Sprintf(`  # Region: %s (%s)
`, region, regionPrefix)
		for _, comp := range components {
			componentConfig := mainConfig.Stack.Components[comp]

			// Get apps for this component in this region
			var apps []string
			for _, rc := range mainConfig.Stack.Architecture.Regions[region] {
				if rc.Component == comp {
					apps = rc.Apps
					break
				}
			}

			// Helper function to get stage dependencies
			getDependencies := func(depString string, currentApp string) string {
				depParts := strings.Split(depString, ".")
				if len(depParts) < 2 {
					return ""
				}

				depRegion := depParts[0]
				depComp := depParts[1]
				if depRegion == "{region}" {
					depRegion = region
				}

				// Check if the dependency component has apps
				hasApps := false
				var depApp string
				if len(depParts) > 2 {
					depApp = depParts[2]
					if depApp == "{app}" {
						depApp = currentApp
					}
					hasApps = true
				} else {
					// Check if the component has apps in the architecture
					for _, rc := range mainConfig.Stack.Architecture.Regions[depRegion] {
						if rc.Component == depComp && len(rc.Apps) > 0 {
							hasApps = true
							depApp = rc.Apps[0] // Use the first app as default
							break
						}
					}
				}

				if hasApps {
					return fmt.Sprintf("'%s_%s_%s'", depRegion, depComp, depApp)
				}
				return fmt.Sprintf("'%s_%s'", depRegion, depComp)
			}

			// If component has apps, create a stage for each app
			if len(apps) > 0 {
				for _, app := range apps {
					stageName := fmt.Sprintf("%s_%s_%s", region, comp, app)
					displayName := fmt.Sprintf("%s/%s/%s", regionPrefix, comp, app)

					// Add dependencies
					var deps []string
					for _, dep := range componentConfig.Deps {
						if depStage := getDependencies(dep, app); depStage != "" {
							deps = append(deps, depStage)
						}
					}

					template += fmt.Sprintf(`  - stage: '%s'
    displayName: '%s'
`, stageName, displayName)

					// Always add dependsOn section
					if len(deps) > 0 {
						template += "    dependsOn:\n"
						for _, dep := range deps {
							template += fmt.Sprintf("      - %s\n", dep)
						}
					} else {
						template += "    dependsOn: []\n"
					}

					template += fmt.Sprintf(`    jobs:
      - job: Deploy
        displayName: 'Deploy Infrastructure (${{ parameters.runMode }})'
        pool:
          vmImage: ubuntu-latest
        steps:
          - template: component-deploy.yml
            parameters:
              component: '%s'
              region: '%s'
              environment: ${{ parameters.environment }}
              subscription: ${{ parameters.subscription }}
              runMode: ${{ parameters.runMode }}
              app: '%s'

`, comp, region, app)
				}
			} else {
				// Create single stage for component without apps
				stageName := fmt.Sprintf("%s_%s", region, comp)
				displayName := fmt.Sprintf("%s/%s", regionPrefix, comp)

				// Add dependencies
				var deps []string
				for _, dep := range componentConfig.Deps {
					if depStage := getDependencies(dep, ""); depStage != "" {
						deps = append(deps, depStage)
					}
				}

				template += fmt.Sprintf(`  - stage: '%s'
    displayName: '%s'
`, stageName, displayName)

				// Always add dependsOn section
				if len(deps) > 0 {
					template += "    dependsOn:\n"
					for _, dep := range deps {
						template += fmt.Sprintf("      - %s\n", dep)
					}
				} else {
					template += "    dependsOn: []\n"
				}

				template += fmt.Sprintf(`    jobs:
      - job: Deploy
        displayName: 'Deploy Infrastructure (${{ parameters.runMode }})'
        pool:
          vmImage: ubuntu-latest
        steps:
          - template: component-deploy.yml
            parameters:
              component: '%s'
              region: '%s'
              environment: ${{ parameters.environment }}
              subscription: ${{ parameters.subscription }}
              runMode: ${{ parameters.runMode }}

`, comp, region)
			}
		}
	}

	// Write the template file
	templatePath := filepath.Join(".azure-pipelines/templates", fmt.Sprintf("stack-%s.yml", stackName))
	if err := os.WriteFile(templatePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write stack template: %w", err)
	}

	return nil
}

// GeneratePipelineTemplates generates all pipeline templates
func GeneratePipelineTemplates() error {
	// Create .azure-pipelines directory if it doesn't exist
	if err := os.MkdirAll(".azure-pipelines", 0755); err != nil {
		return fmt.Errorf("failed to create pipeline directory: %w", err)
	}

	// Read TGS config
	tgsConfig, err := config.ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Track processed stacks to avoid duplicates
	processedStacks := make(map[string]bool)

	// Generate stack templates for each unique stack
	for _, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}

			if !processedStacks[stackName] {
				mainConfig, err := config.ReadMainConfig(stackName)
				if err != nil {
					return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
				}

				if err := generateStackTemplate(stackName, mainConfig); err != nil {
					return fmt.Errorf("failed to generate stack template for %s: %w", stackName, err)
				}

				processedStacks[stackName] = true
			}
		}
	}

	// Generate the component deployment template
	if err := generateDeploymentTemplate(); err != nil {
		return fmt.Errorf("failed to generate deployment template: %w", err)
	}

	// Analyze infrastructure to get components by environment
	envComponents, err := AnalyzeInfrastructure()
	if err != nil {
		return fmt.Errorf("failed to analyze infrastructure: %w", err)
	}

	// Generate pipeline for each environment
	for envName, components := range envComponents {
		if err := generateEnvironmentPipeline(envName, components); err != nil {
			return fmt.Errorf("failed to generate pipeline for environment %s: %w", envName, err)
		}
	}

	return nil
}

// generateDeploymentTemplate generates the deployment template YAML
func generateDeploymentTemplate() error {
	// Create templates directory if it doesn't exist
	if err := os.MkdirAll(".azure-pipelines/templates", 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Create scripts directory if it doesn't exist
	if err := os.MkdirAll(".azure-pipelines/scripts", 0755); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Generate deploy script
	deployScript := `#!/bin/bash
set -e

# Set the working directory
if [ -n "$1" ]; then
  cd .infrastructure/architecture/$2/$3/$4/$5/$1
else
  cd .infrastructure/architecture/$2/$3/$4/$5
fi

# Always run init
terragrunt init

# Run the appropriate command based on runMode
case "$6" in
  "plan")
    terragrunt plan
    ;;
  "apply")
    terragrunt plan
    terragrunt apply --auto-approve
    terragrunt output
    ;;
  "destroy")
    terragrunt destroy --auto-approve
    ;;
  *)
    echo "Invalid runMode: $6"
    exit 1
    ;;
esac`

	if err := os.WriteFile(".azure-pipelines/scripts/deploy.sh", []byte(deployScript), 0755); err != nil {
		return fmt.Errorf("failed to create deploy script: %w", err)
	}

	// Generate component deployment template
	template := `parameters:
  - name: component
    type: string
  - name: region
    type: string
  - name: environment
    type: string
  - name: subscription
    type: string
  - name: app
    type: string
    default: ''
  - name: terraform_version
    type: string
    default: '1.11.2'
  - name: terragrunt_version
    type: string
    default: 'v0.69.10'
  - name: runMode
    type: string
    default: 'plan'
    values:
      - plan
      - apply
      - destroy

steps:
  - script: |
      # Install Terraform
      wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg
      echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
      sudo apt update && sudo apt install -y terraform=${{ parameters.terraform_version }}

      # Install Terragrunt
      wget https://github.com/gruntwork-io/terragrunt/releases/download/${{ parameters.terragrunt_version }}/terragrunt_linux_amd64
      chmod +x terragrunt_linux_amd64
      sudo mv terragrunt_linux_amd64 /usr/local/bin/terragrunt
    displayName: Install Terraform and Terragrunt

  - script: |
      chmod +x .azure-pipelines/scripts/deploy.sh
      .azure-pipelines/scripts/deploy.sh "${{ parameters.app }}" "${{ parameters.subscription }}" "${{ parameters.region }}" "${{ parameters.environment }}" "${{ parameters.component }}" "${{ parameters.runMode }}"
    displayName: Deploy Infrastructure
    env:
      ARM_CLIENT_ID: $(ARM_CLIENT_ID)
      ARM_CLIENT_SECRET: $(ARM_CLIENT_SECRET)
      ARM_SUBSCRIPTION_ID: $(ARM_SUBSCRIPTION_ID)
      ARM_TENANT_ID: $(ARM_TENANT_ID)
`

	return os.WriteFile(".azure-pipelines/templates/component-deploy.yml", []byte(template), 0644)
}

// generateEnvironmentPipeline generates a pipeline for a specific environment
func generateEnvironmentPipeline(envName string, components []Component) error {
	if len(components) == 0 {
		return nil
	}

	// Get subscription and stack from first component (they should all be the same)
	sub := components[0].Sub

	// Read TGS config to get stack name
	tgsConfig, err := config.ReadTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Find stack name for this environment
	stackName := "main"
	for _, subscription := range tgsConfig.Subscriptions {
		for _, env := range subscription.Environments {
			if env.Name == envName {
				if env.Stack != "" {
					stackName = env.Stack
				}
				break
			}
		}
	}

	// Create pipeline content
	pipeline := fmt.Sprintf(`# Pipeline for %s environment
trigger: none
pr: none

parameters:
  - name: runMode
    type: string
    default: plan
    values:
      - plan
      - apply
      - destroy

variables:
  - name: environment
    value: '%s'
  - name: subscription
    value: '%s'
  - group: terraform-variables
  - name: terraform_version
    value: '1.11.2'
  - name: terragrunt_version
    value: 'v0.69.10'

stages:
  - template: templates/stack-%s.yml
    parameters:
      environment: $(environment)
      subscription: $(subscription)
      runMode: ${{ parameters.runMode }}
`, envName, envName, sub, stackName)

	// Write the pipeline file
	pipelinePath := filepath.Join(".azure-pipelines", fmt.Sprintf("%s-pipeline.yml", envName))
	if err := os.WriteFile(pipelinePath, []byte(pipeline), 0644); err != nil {
		return fmt.Errorf("failed to write pipeline file: %w", err)
	}

	return nil
}
