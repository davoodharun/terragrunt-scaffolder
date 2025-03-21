package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
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

// GeneratePipelineTemplates generates pipeline templates for each environment
func GeneratePipelineTemplates() error {
	// Create .azuredevops directory
	if err := os.MkdirAll(".azuredevops", 0755); err != nil {
		return fmt.Errorf("failed to create .azuredevops directory: %w", err)
	}

	// Create .azuredevops/scripts directory
	if err := os.MkdirAll(".azuredevops/scripts", 0755); err != nil {
		return fmt.Errorf("failed to create .azuredevops/scripts directory: %w", err)
	}

	// Create deploy.sh script
	deployScript := `#!/bin/bash
set -e

# Set the working directory
if [ -n "$1" ]; then
  cd .infrastructure/$2/$3/$4/$5/$1
else
  cd .infrastructure/$2/$3/$4/$5
fi

# Always run init
terragrunt init

# Run the appropriate command based on runMode
case "$6" in
  "plan")
    terragrunt plan
    ;;
  "apply")
    terragrunt plan --auto-approve
    terragrunt apply --auto-approve
    ;;
  "destroy")
    terragrunt destroy --auto-approve
    ;;
  *)
    echo "Invalid runMode: $6"
    exit 1
    ;;
esac`

	if err := os.WriteFile(".azuredevops/scripts/deploy.sh", []byte(deployScript), 0755); err != nil {
		return fmt.Errorf("failed to create deploy.sh script: %w", err)
	}

	// Analyze infrastructure
	envComponents, err := AnalyzeInfrastructure()
	if err != nil {
		return fmt.Errorf("failed to analyze infrastructure: %w", err)
	}

	// Generate deployment template
	if err := generateDeploymentTemplate(); err != nil {
		return fmt.Errorf("failed to generate deployment template: %w", err)
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
	template := `parameters:
  - name: component
    type: string
  - name: region
    type: string
  - name: env
    type: string
  - name: sub
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
    default: 'apply'
    values:
      - plan
      - apply
      - destroy

steps:
  - task: UsePythonVersion@0
    inputs:
      versionSpec: '3.9'
      addToPath: true

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
      chmod +x .azuredevops/scripts/deploy.sh
      .azuredevops/scripts/deploy.sh "${{ parameters.app }}" "${{ parameters.sub }}" "${{ parameters.region }}" "${{ parameters.env }}" "${{ parameters.component }}" "${{ parameters.runMode }}"
    displayName: Deploy Infrastructure
    env:
      ARM_CLIENT_ID: $(ARM_CLIENT_ID)
      ARM_CLIENT_SECRET: $(ARM_CLIENT_SECRET)
      ARM_SUBSCRIPTION_ID: $(ARM_SUBSCRIPTION_ID)
      ARM_TENANT_ID: $(ARM_TENANT_ID)
`

	return os.WriteFile(".azuredevops/component-deploy.yml", []byte(template), 0644)
}

// generateEnvironmentPipeline generates the pipeline YAML for an environment
func generateEnvironmentPipeline(envName string, components []Component) error {
	// Build dependency chain
	stages := BuildDependencyChain(components)

	// Generate pipeline YAML
	var pipeline strings.Builder
	pipeline.WriteString(fmt.Sprintf(`parameters:
  - name: runMode
    displayName: Run Mode
    type: string
    default: 'apply'
    values:
      - plan
      - apply
      - destroy

trigger:
  branches:
    include:
      - main
  paths:
    include:
      - .infrastructure/**
      - .azuredevops/**

variables:
  - group: terraform-variables
  - name: terraform_version
    value: '1.11.2'
  - name: terragrunt_version
    value: 'v0.69.10'

stages:
`))

	// Add stages
	for _, stage := range stages {
		// Format dependsOn as an array
		dependsOn := "[]"
		if len(stage.DependsOn) > 0 {
			dependsOn = fmt.Sprintf("[%s]", strings.Join(stage.DependsOn, ", "))
		}

		pipeline.WriteString(fmt.Sprintf(`  - stage: %s
    displayName: Deploy %s
    dependsOn: %s
    jobs:
      - job: Deploy
        displayName: Deploy Infrastructure
        pool:
          vmImage: ubuntu-latest
        steps:
          - template: component-deploy.yml
            parameters:
              component: %s
              region: %s
              env: %s
              sub: %s
              terraform_version: $(terraform_version)
              terragrunt_version: $(terragrunt_version)
              runMode: ${{ parameters.runMode }}
`, stage.Name, stage.Name, dependsOn,
			stage.Parameters["component"], stage.Parameters["region"],
			stage.Parameters["env"], stage.Parameters["sub"]))

		if app, ok := stage.Parameters["app"].(string); ok && app != "" {
			pipeline.WriteString(fmt.Sprintf(`              app: %s
`, app))
		}

		pipeline.WriteString("\n")
	}

	// Write pipeline file
	filename := fmt.Sprintf(".azuredevops/%s-pipeline.yml", envName)
	return os.WriteFile(filename, []byte(pipeline.String()), 0644)
}
