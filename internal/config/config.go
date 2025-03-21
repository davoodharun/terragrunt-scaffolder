package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TGSConfig represents the main TGS configuration
type TGSConfig struct {
	Name          string                  `yaml:"name"`
	Subscriptions map[string]Subscription `yaml:"subscriptions"`
}

// Subscription represents an Azure subscription configuration
type Subscription struct {
	RemoteState  RemoteState   `yaml:"remotestate"`
	Environments []Environment `yaml:"environments"`
}

// RemoteState represents the remote state configuration
type RemoteState struct {
	Name          string `yaml:"name"`
	ResourceGroup string `yaml:"resource_group"`
}

// Environment represents an environment configuration
type Environment struct {
	Name  string `yaml:"name"`
	Stack string `yaml:"stack,omitempty"`
}

// MainConfig represents the main stack configuration
type MainConfig struct {
	Stack StackConfig `yaml:"stack"`
}

// StackConfig represents the stack configuration
type StackConfig struct {
	Name         string               `yaml:"name"`
	Version      string               `yaml:"version"`
	Description  string               `yaml:"description"`
	Architecture ArchitectureConfig   `yaml:"architecture"`
	Components   map[string]Component `yaml:"components"`
}

// ArchitectureConfig represents the architecture configuration
type ArchitectureConfig struct {
	Regions map[string][]RegionComponent `yaml:"regions"`
}

// RegionComponent represents a component in a region
type RegionComponent struct {
	Component string   `yaml:"component"`
	Apps      []string `yaml:"apps,omitempty"`
}

// Component represents a component configuration
type Component struct {
	Source      string   `yaml:"source"`
	Deps        []string `yaml:"deps"`
	Provider    string   `yaml:"provider"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description"`
}

// ReadTGSConfig reads the TGS configuration file
func ReadTGSConfig() (*TGSConfig, error) {
	data, err := os.ReadFile(".tgs/tgs.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read TGS config: %w", err)
	}

	var config TGSConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse TGS config: %w", err)
	}

	return &config, nil
}

// ReadMainConfig reads the main stack configuration file
func ReadMainConfig(stackName string) (*MainConfig, error) {
	data, err := os.ReadFile(filepath.Join(".tgs/stacks", stackName+".yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read stack config: %w", err)
	}

	var config MainConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse stack config: %w", err)
	}

	return &config, nil
}
