package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPath defines the structure for test path validation
type TestPath struct {
	path     string
	isDir    bool
	required bool
}

func TestGenerateCommand(t *testing.T) {
	testCases := []struct {
		name        string
		tgsConfig   string
		stackConfig string
		wantErr     bool
		errContains string
	}{
		{
			name: "Valid configuration",
			tgsConfig: `name: projecta
subscriptions:
  nonprod:
    remotestate:
      name: stprojectanonprodtf
      resource_group: rg-projecta-nonprod-tf
    environments:
      - name: dev
        stack: main`,
			stackConfig: `stack:
  name: main
  version: "1.0.0"
  description: "Test stack"
  components:
    redis:
      source: azurerm_redis_cache
      provider: azurerm
      version: 4.22.0
      description: "Redis cache for caching"
    appservice:
      source: azurerm_app_service
      provider: azurerm
      version: 4.22.0
      description: "App service for API"
      deps:
        - "{region}.redis"
  architecture:
    regions:
      eastus2:
        - component: redis
          apps: []
        - component: appservice
          apps:
            - api`,
			wantErr: false,
		},
		{
			name: "Missing project name in tgs.yaml",
			tgsConfig: `subscriptions:
  nonprod:
    remotestate:
      name: stprojectanonprodtf
      resource_group: rg-projecta-nonprod-tf
    environments:
      - name: dev
        stack: main`,
			stackConfig: `stack:
  name: main
  version: "1.0.0"
  description: "Test stack"
  components:
    redis:
      source: azurerm_redis_cache
      provider: azurerm
      version: 4.22.0
      description: "Redis cache"`,
			wantErr:     true,
			errContains: "project name cannot be empty",
		},
		{
			name: "Missing remotestate in subscription",
			tgsConfig: `name: projecta
subscriptions:
  nonprod:
    environments:
      - name: dev
        stack: main`,
			stackConfig: `stack:
  name: main
  version: "1.0.0"
  description: "Test stack"
  components:
    redis:
      source: azurerm_redis_cache
      provider: azurerm
      version: 4.22.0
      description: "Redis cache"`,
			wantErr:     true,
			errContains: "remotestate.name property must be filled",
		},
		{
			name: "Missing stack version",
			tgsConfig: `name: projecta
subscriptions:
  nonprod:
    remotestate:
      name: stprojectanonprodtf
      resource_group: rg-projecta-nonprod-tf
    environments:
      - name: dev
        stack: main`,
			stackConfig: `stack:
  name: main
  description: "Test stack"
  components:
    redis:
      source: azurerm_redis_cache
      provider: azurerm
      version: 4.22.0
      description: "Redis cache"`,
			wantErr:     true,
			errContains: "version property must be filled",
		},
		{
			name: "Missing component description",
			tgsConfig: `name: projecta
subscriptions:
  nonprod:
    remotestate:
      name: stprojectanonprodtf
      resource_group: rg-projecta-nonprod-tf
    environments:
      - name: dev
        stack: main`,
			stackConfig: `stack:
  name: main
  version: "1.0.0"
  description: "Test stack"
  components:
    redis:
      source: azurerm_redis_cache
      provider: azurerm
      version: 4.22.0`,
			wantErr:     true,
			errContains: "description property must be filled",
		},
		{
			name: "Invalid dependency format",
			tgsConfig: `name: projecta
subscriptions:
  nonprod:
    remotestate:
      name: stprojectanonprodtf
      resource_group: rg-projecta-nonprod-tf
    environments:
      - name: dev
        stack: main`,
			stackConfig: `stack:
  name: main
  version: "1.0.0"
  description: "Test stack"
  components:
    appservice:
      source: azurerm_app_service
      provider: azurerm
      version: 4.22.0
      description: "App service"
      deps:
        - "invalid_dependency"`,
			wantErr:     true,
			errContains: "invalid dependency format",
		},
		{
			name: "Component referenced in architecture but not defined",
			tgsConfig: `name: projecta
subscriptions:
  nonprod:
    remotestate:
      name: stprojectanonprodtf
      resource_group: rg-projecta-nonprod-tf
    environments:
      - name: dev
        stack: main`,
			stackConfig: `stack:
  name: main
  version: "1.0.0"
  description: "Test stack"
  components:
    redis:
      source: azurerm_redis_cache
      provider: azurerm
      version: 4.22.0
      description: "Redis cache"
  architecture:
    regions:
      eastus2:
        - component: undefined_component
          apps: []`,
			wantErr:     true,
			errContains: "component 'undefined_component' referenced in architecture but not defined",
		},
		{
			name: "Invalid region in dependency",
			tgsConfig: `name: projecta
subscriptions:
  nonprod:
    remotestate:
      name: stprojectanonprodtf
      resource_group: rg-projecta-nonprod-tf
    environments:
      - name: dev
        stack: main`,
			stackConfig: `stack:
  name: main
  version: "1.0.0"
  description: "Test stack"
  components:
    redis:
      source: azurerm_redis_cache
      provider: azurerm
      version: 4.22.0
      description: "Redis cache"
    appservice:
      source: azurerm_app_service
      provider: azurerm
      version: 4.22.0
      description: "App service"
      deps:
        - "invalid_region.redis"`,
			wantErr:     true,
			errContains: "invalid region in dependency",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "tgs-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Save current directory and change to temp directory
			currentDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}
			defer os.Chdir(currentDir)

			// Create .tgs directory and config files
			tgsDir := filepath.Join(tmpDir, ".tgs")
			stacksDir := filepath.Join(tgsDir, "stacks")
			if err := os.MkdirAll(stacksDir, 0755); err != nil {
				t.Fatalf("Failed to create .tgs/stacks directory: %v", err)
			}

			// Write tgs.yaml
			if err := os.WriteFile(filepath.Join(tgsDir, "tgs.yaml"), []byte(tc.tgsConfig), 0644); err != nil {
				t.Fatalf("Failed to write tgs.yaml: %v", err)
			}

			// Write main.yaml stack file
			if err := os.WriteFile(filepath.Join(stacksDir, "main.yaml"), []byte(tc.stackConfig), 0644); err != nil {
				t.Fatalf("Failed to write main.yaml: %v", err)
			}

			// Run generate command
			err = Generate()

			// Check if error matches expectation
			if tc.wantErr {
				if err == nil {
					t.Errorf("Generate() expected error containing %q, got no error", tc.errContains)
				} else if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("Generate() error = %v, want error containing %q", err, tc.errContains)
				}
			} else if err != nil {
				t.Errorf("Generate() unexpected error: %v", err)
			}
		})
	}
}
