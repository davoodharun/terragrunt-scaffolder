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

// GenerateDiagram generates PlantUML diagrams for all stacks
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

			// Generate diagrams for each environment
			for _, sub := range tgsConfig.Subscriptions {
				for _, env := range sub.Environments {
					if err := generatePlantUMLDiagram(stackName, tgsConfig, env.Name); err != nil {
						return fmt.Errorf("failed to generate diagram for stack %s, environment %s: %w", stackName, env.Name, err)
					}
				}
			}
		}
	}

	logger.Info("Generated infrastructure diagrams in diagrams/ directory")
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
