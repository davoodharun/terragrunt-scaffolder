package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/davoodharun/tgs/internal/config"
	"github.com/davoodharun/tgs/internal/logger"
)

func generateEnvironment(subName, region string, env config.Environment, mainConfig *config.MainConfig) error {
	logger.Info("Generating environment: %s/%s/%s", subName, region, env.Name)

	// Create environment base path
	basePath := filepath.Join(".infrastructure", subName, region, env.Name)
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create environment directory: %w", err)
	}

	// Create environment-level config
	envConfig := fmt.Sprintf(`locals {
  environment_name = "%s"
  environment_path = "${get_parent_terragrunt_dir()}"
}`, env.Name)

	if err := createFile(filepath.Join(basePath, "environment.hcl"), envConfig); err != nil {
		return err
	}

	// Get components for the region
	components := mainConfig.Stack.Architecture.Regions[region]

	// Generate component directories and their apps
	for _, comp := range components {
		compPath := filepath.Join(basePath, comp.Component)
		if err := os.MkdirAll(compPath, 0755); err != nil {
			return fmt.Errorf("failed to create component directory: %w", err)
		}

		if len(comp.Apps) > 0 {
			logger.Info("Generating apps for component %s", comp.Component)
			// Create app-specific folders and terragrunt files
			for _, app := range comp.Apps {
				appPath := filepath.Join(compPath, app)
				if err := os.MkdirAll(appPath, 0755); err != nil {
					return fmt.Errorf("failed to create app directory: %w", err)
				}

				// Create app-specific terragrunt.hcl
				terragruntContent := fmt.Sprintf(`include "root" {
  path = find_in_parent_folders("root.hcl")
}

include "component" {
  path = "${get_repo_root()}/.infrastructure/_components/%s/component.hcl"
}

locals {
  app_name = "%s"
}`, comp.Component, app)

				if err := createFile(filepath.Join(appPath, "terragrunt.hcl"), terragruntContent); err != nil {
					return fmt.Errorf("failed to create terragrunt.hcl for app: %w", err)
				}
			}
		} else {
			logger.Info("Generating single component %s", comp.Component)
			// Create single terragrunt.hcl for components without apps
			terragruntContent := fmt.Sprintf(`include "root" {
  path = find_in_parent_folders("root.hcl")
}

include "component" {
  path = "${get_repo_root()}/.infrastructure/_components/%s/component.hcl"
}`, comp.Component)

			if err := createFile(filepath.Join(compPath, "terragrunt.hcl"), terragruntContent); err != nil {
				return fmt.Errorf("failed to create terragrunt.hcl for component: %w", err)
			}
		}
	}

	return nil
}
