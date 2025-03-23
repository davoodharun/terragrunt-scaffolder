package scaffold

import (
	"os"
	"path/filepath"
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
	}{
		{
			name: "Simple single environment",
			tgsConfig: `name: CUSTTP
subscriptions:
  nonprod:
    remotestate:
      name: custstfstatessta000
      resource_group: CUSTTP-E-N-TFSTATE-RGP
    environments:
      - name: dev
        stack: main`,
			stackConfig: `stack:
  components:
    appservice:
      source: azurerm_app_service
      provider: azurerm
      version: 4.22.0
  architecture:
    regions:
      eastus2:
        - component: appservice
          apps: []`,
		},
		{
			name: "Multiple environments and regions",
			tgsConfig: `name: CUSTTP
subscriptions:
  nonprod:
    remotestate:
      name: custstfstatessta000
      resource_group: CUSTTP-E-N-TFSTATE-RGP
    environments:
      - name: dev
        stack: main
      - name: test
        stack: main
  prod:
    remotestate:
      name: custstfstatessta000
      resource_group: CUSTTP-E-P-TFSTATE-RGP
    environments:
      - name: prod
        stack: main`,
			stackConfig: `stack:
  components:
    rediscache:
      source: azurerm_redis_cache
      provider: azurerm
      version: 4.22.0
      deps: []
    appservice:
      source: azurerm_app_service
      provider: azurerm
      version: 4.22.0
      deps:
        - "eastus2.redis"
        - "{region}.serviceplan.{app}"
    serviceplan:
      source: azurerm_service_plan
      provider: azurerm
      version: 4.22.0 
      deps: []
  architecture:
    regions:
      eastus2:
        - component: rediscache
          apps: []
        - component: serviceplan
          apps: 
            - api
            - web
        - component: appservice
          apps:
            - api
            - web
      westus:
        - component: serviceplan
          apps: 
            - api
            - web
        - component: appservice
          apps:
            - api
            - web`,
		},
		{
			name: "Component with dependencies",
			tgsConfig: `name: CUSTTP
subscriptions:
  nonprod:
    remotestate:
      name: custstfstatessta000
      resource_group: CUSTTP-E-N-TFSTATE-RGP
    environments:
      - name: dev
        stack: main`,
			stackConfig: `stack:
  components:
    redis:
      source: azurerm_redis_cache
      provider: azurerm
      version: 4.22.0
    appservice:
      source: azurerm_app_service
      provider: azurerm
      version: 4.22.0
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
			if err := Generate(); err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Define test paths based on the configuration
			var testPaths []TestPath

			// Add common paths
			testPaths = append(testPaths,
				TestPath{
					path:     ".infrastructure",
					isDir:    true,
					required: true,
				},
				TestPath{
					path:     ".infrastructure/config",
					isDir:    true,
					required: true,
				},
				TestPath{
					path:     ".infrastructure/config/main",
					isDir:    true,
					required: true,
				},
				TestPath{
					path:     ".infrastructure/_components",
					isDir:    true,
					required: true,
				},
				TestPath{
					path:     ".infrastructure/_components/main",
					isDir:    true,
					required: true,
				},
			)

			// Add paths based on configuration
			if tc.name == "Simple single environment" {
				testPaths = append(testPaths,
					TestPath{
						path:     ".infrastructure/nonprod/eastus2/dev/appservice",
						isDir:    true,
						required: true,
					},
				)
			} else if tc.name == "Multiple environments and regions" {
				testPaths = append(testPaths,
					TestPath{
						path:     ".infrastructure/nonprod/eastus2/dev/appservice/api",
						isDir:    true,
						required: true,
					},
					TestPath{
						path:     ".infrastructure/prod/eastus2/prod/appservice/web",
						isDir:    true,
						required: true,
					},
					TestPath{
						path:     ".infrastructure/nonprod/westus/test/appservice/api",
						isDir:    true,
						required: true,
					},
				)
			} else if tc.name == "Component with dependencies" {
				testPaths = append(testPaths,
					TestPath{
						path:     ".infrastructure/nonprod/eastus2/dev/redis",
						isDir:    true,
						required: true,
					},
					TestPath{
						path:     ".infrastructure/nonprod/eastus2/dev/appservice/api",
						isDir:    true,
						required: true,
					},
				)
			}

			// Test all paths
			for _, tp := range testPaths {
				path := filepath.Join(tmpDir, tp.path)
				info, err := os.Stat(path)
				if tp.required {
					if err != nil {
						t.Errorf("Required path %s does not exist: %v", tp.path, err)
						continue
					}
					if info.IsDir() != tp.isDir {
						t.Errorf("Path %s: expected isDir=%v, got isDir=%v", tp.path, tp.isDir, info.IsDir())
					}
				} else if err == nil {
					if info.IsDir() != tp.isDir {
						t.Errorf("Optional path %s: expected isDir=%v, got isDir=%v", tp.path, tp.isDir, info.IsDir())
					}
				}
			}
		})
	}
}
