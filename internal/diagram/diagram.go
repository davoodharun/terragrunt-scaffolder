package diagram

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
	"gopkg.in/yaml.v3"
)

// GenerateDiagram generates PlantUML diagrams for all stacks
func GenerateDiagram() error {
	logger.Info("Generating infrastructure diagrams")

	// Read TGS config to get subscription and environment structure
	tgsConfig, err := readTGSConfig()
	if err != nil {
		return fmt.Errorf("failed to read TGS config: %w", err)
	}

	// Create diagrams directory in .infrastructure if it doesn't exist
	outputDir := filepath.Join(".infrastructure", "diagrams")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create diagrams directory: %w", err)
	}

	// Track which stacks we've processed to avoid duplicates
	processedStacks := make(map[string]bool)

	// Generate diagrams for each environment using its specified stack
	for _, sub := range tgsConfig.Subscriptions {
		for _, env := range sub.Environments {
			// Use the stack specified in the environment config, default to "main" if not specified
			stackName := "main"
			if env.Stack != "" {
				stackName = env.Stack
			}

			// Skip if we've already processed this stack for this environment
			key := fmt.Sprintf("%s_%s", stackName, env.Name)
			if processedStacks[key] {
				continue
			}
			processedStacks[key] = true

			if err := generatePlantUMLDiagram(stackName, tgsConfig, env.Name); err != nil {
				return fmt.Errorf("failed to generate diagram for stack %s, environment %s: %w", stackName, env.Name, err)
			}

			logger.Info("Generated diagram for stack %s, environment %s", stackName, env.Name)
		}
	}

	logger.Info("Generated infrastructure diagrams in .infrastructure/diagrams/ directory")
	return nil
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
