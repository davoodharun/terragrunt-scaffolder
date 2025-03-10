package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/davoodharun/tgs/internal/config"
	"gopkg.in/yaml.v3"
)

func Generate() error {
	// Read configurations
	tgsConfig, err := readTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read tgs config: %w", err)
	}

	mainConfig, err := readMainConfig()
	if err != nil {
		return fmt.Errorf("failed to read main config: %w", err)
	}

	// Create base infrastructure directory
	if err := os.MkdirAll(".infrastructure", 0755); err != nil {
		return fmt.Errorf("failed to create .infrastructure directory: %w", err)
	}

	// Generate scaffold for each subscription and environment
	for subName, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			if err := generateEnvironment(subName, env, mainConfig); err != nil {
				return fmt.Errorf("failed to generate environment %s/%s: %w", subName, env.Name, err)
			}
		}
	}

	return nil
}

func generateEnvironment(subName string, env config.Environment, mainConfig *config.MainConfig) error {
	basePath := filepath.Join(".infrastructure", subName, env.Name)
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return err
	}

	// Generate component directories for each region
	for region, components := range mainConfig.Stack.Architecture.Regions {
		for _, comp := range components {
			// Create component directory
			compPath := filepath.Join(basePath, region, comp.Component)
			if err := os.MkdirAll(compPath, 0755); err != nil {
				return err
			}

			// Create main.tf placeholder
			if err := createFile(filepath.Join(compPath, "main.tf"), ""); err != nil {
				return err
			}

			// Create variables.tf placeholder
			if err := createFile(filepath.Join(compPath, "variables.tf"), ""); err != nil {
				return err
			}
		}
	}

	return nil
}

func readTGSConfig() (*config.TGSConfig, error) {
	// Get the executable's directory
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Read from the tgs directory
	data, err := os.ReadFile(filepath.Join(execDir, "tgs.yaml"))
	if err != nil {
		// Try current directory as fallback
		data, err = os.ReadFile("tgs.yaml")
		if err != nil {
			return nil, err
		}
	}

	var cfg config.TGSConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func readMainConfig() (*config.MainConfig, error) {
	// Get the executable's directory
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Read from the tgs directory
	data, err := os.ReadFile(filepath.Join(execDir, "main.yaml"))
	if err != nil {
		// Try current directory as fallback
		data, err = os.ReadFile("main.yaml")
		if err != nil {
			return nil, err
		}
	}

	var cfg config.MainConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal main.yaml: %w", err)
	}

	return &cfg, nil
}

func createFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
