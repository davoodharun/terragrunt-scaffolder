package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateCommand(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "tgs-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create minimal tgs.yaml for testing
	tgsConfig := `
name: test-project
subscriptions:
  test-sub:
    remote_state:
      resource_group: test-rg
      name: test-storage
    environments:
      - name: dev
        stack: main
      - name: test
        stack: main
      - name: stage
        stack: main
      - name: prod
        stack: main
`
	if err := os.WriteFile("tgs.yaml", []byte(tgsConfig), 0644); err != nil {
		t.Fatalf("Failed to create tgs.yaml: %v", err)
	}

	// Create minimal main.yaml for testing
	mainConfig := `
stack:
  components:
    test_component:
      source: azurerm_resource_group
      provider: azurerm
      version: "3.0.0"
  architecture:
    regions:
      eastus2:
        - component: test_component
`
	stacksDir := filepath.Join(".tgs", "stacks")
	if err := os.MkdirAll(stacksDir, 0755); err != nil {
		t.Fatalf("Failed to create stacks directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stacksDir, "main.yaml"), []byte(mainConfig), 0644); err != nil {
		t.Fatalf("Failed to create main.yaml: %v", err)
	}

	// Run generate command
	if err := Generate(); err != nil {
		t.Fatalf("Generate command failed: %v", err)
	}

	// Test cases for directory and file existence
	testCases := []struct {
		name     string
		path     string
		isDir    bool
		required bool
	}{
		// Infrastructure directory
		{"Infrastructure directory", ".infrastructure", true, true},

		// Config directory and files
		{"Config directory", filepath.Join(".infrastructure", "config"), true, true},
		{"Config stack directory", filepath.Join(".infrastructure", "config", "main"), true, true},
		{"Global config", filepath.Join(".infrastructure", "config", "global.hcl"), false, true},
		{"Dev config", filepath.Join(".infrastructure", "config", "main", "dev.hcl"), false, true},
		{"Test config", filepath.Join(".infrastructure", "config", "main", "test.hcl"), false, true},
		{"Stage config", filepath.Join(".infrastructure", "config", "main", "stage.hcl"), false, true},
		{"Prod config", filepath.Join(".infrastructure", "config", "main", "prod.hcl"), false, true},

		// Components directory
		{"Components directory", filepath.Join(".infrastructure", "_components"), true, true},
		{"Components stack directory", filepath.Join(".infrastructure", "_components", "main"), true, true},
		{"Component directory", filepath.Join(".infrastructure", "_components", "main", "test_component"), true, true},
		{"Component files", filepath.Join(".infrastructure", "_components", "main", "test_component", "component.hcl"), false, true},
		{"Main TF", filepath.Join(".infrastructure", "_components", "main", "test_component", "main.tf"), false, true},
		{"Variables TF", filepath.Join(".infrastructure", "_components", "main", "test_component", "variables.tf"), false, true},
		{"Provider TF", filepath.Join(".infrastructure", "_components", "main", "test_component", "provider.tf"), false, true},

		// Subscription structure
		{"Subscription directory", filepath.Join(".infrastructure", "test-sub"), true, true},
		{"Subscription config", filepath.Join(".infrastructure", "test-sub", "subscription.hcl"), false, true},

		// Region structure
		{"Region directory", filepath.Join(".infrastructure", "test-sub", "eastus2"), true, true},
		{"Region config", filepath.Join(".infrastructure", "test-sub", "eastus2", "region.hcl"), false, true},

		// Environment structure
		{"Dev environment directory", filepath.Join(".infrastructure", "test-sub", "eastus2", "dev"), true, true},
		{"Test environment directory", filepath.Join(".infrastructure", "test-sub", "eastus2", "test"), true, true},
		{"Stage environment directory", filepath.Join(".infrastructure", "test-sub", "eastus2", "stage"), true, true},
		{"Prod environment directory", filepath.Join(".infrastructure", "test-sub", "eastus2", "prod"), true, true},

		// Component in environment
		{"Component in dev", filepath.Join(".infrastructure", "test-sub", "eastus2", "dev", "test_component"), true, true},
		{"Terragrunt config in dev", filepath.Join(".infrastructure", "test-sub", "eastus2", "dev", "test_component", "terragrunt.hcl"), false, true},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := os.Stat(tc.path)
			if err != nil {
				if tc.required {
					t.Errorf("Required path %s does not exist: %v", tc.path, err)
				}
				return
			}

			if tc.isDir && !info.IsDir() {
				t.Errorf("Expected %s to be a directory", tc.path)
			}
			if !tc.isDir && info.IsDir() {
				t.Errorf("Expected %s to be a file", tc.path)
			}
		})
	}
}
