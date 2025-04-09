package scaffold

import (
	"os"
	"testing"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

// TestPath defines the structure for test path validation
type TestPath struct {
	path     string
	isDir    bool
	required bool
}

func TestGenerateCommand(t *testing.T) {
	// Enable test mode to suppress logs
	logger.SetTestMode(true)
	defer logger.SetTestMode(false)

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "scaffold-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create necessary directories
	dirs := []string{
		".tgs",
		".tgs/stacks",
		".infrastructure",
		".infrastructure/config",
		".infrastructure/_components",
		".infrastructure/architecture",
		".infrastructure/root",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Write test configuration files
	tgsConfig := `name: test-project
remote_state:
  name: test-state
  resource_group: test-rg
  storage_account: teststorage
  container_name: tfstate
subscriptions:
  sub1:
    environments:
      - name: dev
        stack: main
      - name: prod
        stack: main
    remotestate:
      name: test-state-sub1
      resource_group: test-rg-sub1`

	if err := os.WriteFile(".tgs/tgs.yaml", []byte(tgsConfig), 0644); err != nil {
		t.Fatalf("Failed to write tgs.yaml: %v", err)
	}

	mainConfig := `version: 1.0.0
description: Test stack
stack:
  name: main
  version: 1.0.0
  description: Test stack
  components:
    storage:
      source: azurerm_storage_account
      version: "3.0.0"
      provider: azurerm
      description: "Storage account for test"
    redis:
      source: azurerm_redis_cache
      version: "3.0.0"
      provider: azurerm
      description: "Redis cache for test"
      deps:
        - eastus.storage
  architecture:
    regions:
      eastus:
        - component: storage
          apps: []
        - component: redis
          apps: []
      westus:
        - component: storage
          apps: []
        - component: redis
          apps: []`

	if err := os.WriteFile(".tgs/stacks/main.yaml", []byte(mainConfig), 0644); err != nil {
		t.Fatalf("Failed to write main.yaml: %v", err)
	}

	// Load TGS config
	tgsConfigObj, err := config.ReadTGSConfig()
	if err != nil {
		t.Fatalf("Failed to read TGS config: %v", err)
	}

	// Run tests
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Generate basic infrastructure",
			wantErr: false,
		},
		{
			name:    "Handle dependencies",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Generate(tgsConfigObj)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify generated files
			files := []string{
				".infrastructure/root/root.hcl",
				".infrastructure/config/sub1/dev/environment.hcl",
				".infrastructure/config/sub1/prod/environment.hcl",
				".infrastructure/_components/storage/component.hcl",
				".infrastructure/_components/redis/component.hcl",
			}

			for _, file := range files {
				if _, err := os.Stat(file); os.IsNotExist(err) {
					t.Errorf("Expected file %s was not created", file)
				}
			}
		})
	}
}

func TestScaffold(t *testing.T) {
	// Enable test mode to suppress logs
	logger.SetTestMode(true)
	defer logger.SetTestMode(false)

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "scaffold-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create necessary directories
	dirs := []string{
		".tgs",
		".tgs/stacks",
		".infrastructure",
		".infrastructure/config",
		".infrastructure/_components",
		".infrastructure/architecture",
		".infrastructure/root",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Write test configuration files
	tgsConfig := `name: test-project
remote_state:
  name: test-state
  resource_group: test-rg
  storage_account: teststorage
  container_name: tfstate
subscriptions:
  sub1:
    environments:
      - name: dev
        stack: main
      - name: prod
        stack: main
    remotestate:
      name: test-state-sub1
      resource_group: test-rg-sub1`

	if err := os.WriteFile(".tgs/tgs.yaml", []byte(tgsConfig), 0644); err != nil {
		t.Fatalf("Failed to write tgs.yaml: %v", err)
	}

	mainConfig := `version: 1.0.0
description: Test stack
stack:
  name: main
  version: 1.0.0
  description: Test stack
  components:
    storage:
      source: azurerm_storage_account
      version: "3.0.0"
      provider: azurerm
      description: "Storage account for test"
    redis:
      source: azurerm_redis_cache
      version: "3.0.0"
      provider: azurerm
      description: "Redis cache for test"
      deps:
        - eastus.storage
  architecture:
    regions:
      eastus:
        - component: storage
          apps: []
        - component: redis
          apps: []
      westus:
        - component: storage
          apps: []
        - component: redis
          apps: []`

	if err := os.WriteFile(".tgs/stacks/main.yaml", []byte(mainConfig), 0644); err != nil {
		t.Fatalf("Failed to write main.yaml: %v", err)
	}

	// Load TGS config
	tgsConfigObj, err := config.ReadTGSConfig()
	if err != nil {
		t.Fatalf("Failed to read TGS config: %v", err)
	}

	// Run tests
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Generate basic infrastructure",
			wantErr: false,
		},
		{
			name:    "Handle dependencies",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Generate(tgsConfigObj)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify generated files
			files := []string{
				".infrastructure/root/root.hcl",
				".infrastructure/config/sub1/dev/environment.hcl",
				".infrastructure/config/sub1/prod/environment.hcl",
				".infrastructure/_components/storage/component.hcl",
				".infrastructure/_components/redis/component.hcl",
			}

			for _, file := range files {
				if _, err := os.Stat(file); os.IsNotExist(err) {
					t.Errorf("Expected file %s was not created", file)
				}
			}
		})
	}
}
