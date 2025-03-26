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
	Naming        NamingConfig            `yaml:"naming"`
}

// NamingConfig represents the resource naming configuration
type NamingConfig struct {
	Format           string                     `yaml:"format"`
	ResourcePrefixes map[string]string          `yaml:"resource_prefixes"`
	DefaultSeparator string                     `yaml:"separator"`
	ComponentFormats map[string]ComponentFormat `yaml:"component_formats,omitempty"`
}

// ComponentFormat represents a custom format for a specific component
type ComponentFormat struct {
	Format    string `yaml:"format"`
	Separator string `yaml:"separator,omitempty"`
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
	Provider    string   `yaml:"provider"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description"`
	Deps        []string `yaml:"deps"`
	AppSettings bool     `yaml:"app_settings"`
	PolicyFiles bool     `yaml:"policy_files"`
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

	// Validate project name
	if err := validateProjectName(config.Name); err != nil {
		return nil, fmt.Errorf("invalid project name: %w", err)
	}

	// Set default naming configuration if not provided
	if config.Naming.Format == "" {
		config.Naming.Format = "${project}-${region}${env}-${type}"
	}
	if config.Naming.DefaultSeparator == "" {
		config.Naming.DefaultSeparator = "-"
	}

	return &config, nil
}

// validateProjectName ensures the project name follows the required format:
// - Lowercase letters, numbers, and hyphens only
// - Must start with a lowercase letter or number
// - No consecutive hyphens
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Check first character is lowercase letter or number
	firstChar := rune(name[0])
	if !((firstChar >= 'a' && firstChar <= 'z') || (firstChar >= '0' && firstChar <= '9')) {
		return fmt.Errorf("project name must start with a lowercase letter or number")
	}

	// Check for valid characters and consecutive hyphens
	prevHyphen := false
	for _, char := range name {
		if char == '-' {
			if prevHyphen {
				return fmt.Errorf("project name cannot contain consecutive hyphens")
			}
			prevHyphen = true
		} else if char >= 'A' && char <= 'Z' {
			return fmt.Errorf("project name cannot contain uppercase letters")
		} else if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			return fmt.Errorf("project name can only contain lowercase letters, numbers, and hyphens")
		} else {
			prevHyphen = false
		}
	}

	// Check if name ends with hyphen
	if name[len(name)-1] == '-' {
		return fmt.Errorf("project name cannot end with a hyphen")
	}

	return nil
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

	// Log the architecture configuratio

	return &config, nil
}
