package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

func generateComponents(mainConfig *config.MainConfig) error {
	logger.Info("Generating components")

	// Get the infrastructure path
	infraPath := getInfrastructurePath()

	// Create components directory
	baseDir := filepath.Base(infraPath)
	componentsDir := filepath.Join(baseDir, "_components")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create components directory: %w", err)
	}

	// Generate each component
	for compName, comp := range mainConfig.Stack.Components {
		logger.Info("Generating component: %s", compName)

		// Create component directory
		componentPath := filepath.Join(componentsDir, compName)
		if err := os.MkdirAll(componentPath, 0755); err != nil {
			return err
		}

		// Generate Terraform files from provider schema if available
		if err := generateTerraformFiles(componentPath, comp); err != nil {
			return err
		}

		// Create component.hcl with dependency blocks
		componentHcl := fmt.Sprintf(`
locals {
  subscription_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl"))
  region_vars = read_terragrunt_config(find_in_parent_folders("region.hcl"))
  environment_vars = read_terragrunt_config(find_in_parent_folders("environment.hcl"))

  # Infrastructure path relative to repo root
  infrastructure_path = get_env("TG_INFRASTRUCTURE_PATH", "%s")

  subscription_name = local.subscription_vars.locals.subscription_name
  region_name = local.region_vars.locals.region_name
  environment_name = local.environment_vars.locals.environment_name
  
  # Get the directory name as the app name, defaulting to empty string if at component root
  app_name = try(basename(dirname(get_terragrunt_dir())), basename(get_terragrunt_dir()), "")
}

terraform {
  source = "${get_repo_root()}/${local.infrastructure_path}/_components/%s"
}

%s

inputs = {
  subscription_name = local.subscription_name
  region_name = local.region_name
  environment_name = local.environment_vars.locals.environment_name
  app_name = local.app_name
  name = coalesce(try("${local.app_name}-${local.environment_name}", ""), local.environment_name)
  resource_group_name = coalesce(try("rg-${local.app_name}-${local.environment_name}", ""), "rg-${local.environment_name}")
  location = local.region_name
  tags = {
    Environment = local.environment_name
    Application = local.app_name
  }
}`, infraPath, compName, generateDependencyBlocks(comp.Deps, infraPath))

		if err := createFile(filepath.Join(componentPath, "component.hcl"), componentHcl); err != nil {
			return err
		}
	}
	return nil
}
