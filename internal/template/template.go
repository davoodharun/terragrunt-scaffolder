// Package template provides functionality for generating template files
package template

import (
	"fmt"
	"os"
	"path/filepath"
)

// TGSYamlTemplate is the default template for tgs.yaml
const TGSYamlTemplate = `name: CUSTTP  # Your project name
subscriptions:
  nonprod:
    remotestate:
      name: # Storage account name for remote state
      resource_group: # Resource group for remote state
    environments:
      - name: dev
        stack: main
      - name: test
        stack: main
  prod:
    remotestate:
      name: # Storage account name for remote state
      resource_group: # Resource group for remote state
    environments:
      - name: prod
        stack: main
`

// MainYamlTemplate is the default template for main.yaml (stack configuration)
const MainYamlTemplate = `stack:
  components:
    # Example components
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
        # Dependency notation examples:
        # - "eastus2.redis"                # Fixed region and component
        # - "{region}.serviceplan"         # Current region with fixed component
        # - "{region}.serviceplan.{app}"   # Current region, component, and app
        # - "eastus2.cosmos_db.api"        # Fixed region, component, and app
        - "{region}.serviceplan.{app}"     # Depends on serviceplan in same region for same app
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
            - web
`

// GitignoreTemplate is the default template for .gitignore
const GitignoreTemplate = `# See https://help.github.com/articles/ignoring-files/ for more about ignoring files.

# Code Editor settings
.vscode/
.idea/

# TGS specific folders
.tgs/
.infrastructure/

# Local .terraform directories
**/.terraform/*

# .tfstate files
*.tfstate
*.tfstate.*

# Crash log files
crash.log
crash.*.log

# Exclude all .tfvars files, which are likely to contain sensitive data
*.tfvars
*.tfvars.json

# Ignore override files as they are usually used for local dev
override.tf
override.tf.json
*_override.tf
*_override.tf.json

# Ignore CLI configuration files
.terraformrc
terraform.rc

# Terragrunt cache directories
**/.terragrunt-cache/*

# Terragrunt lock files
.terraform.lock.hcl

# Ignore generated backend files
backend.tf

# Ignore provider generated files
.plugins/
.plugin-cache/

# Ignore temporary files
*.tmp
*.bak
*.swp
*~

# Ignore OS specific files
.DS_Store
Thumbs.db

# Ignore binary files
*.exe
*.dll
*.so
*.dylib

# Ignore log files
*.log
`

// TerraformignoreTemplate is the default template for .terraformignore
const TerraformignoreTemplate = `# TGS specific folders
.tgs/
.infrastructure/

# Local .terraform directories
**/.terraform/*

# .tfstate files
*.tfstate
*.tfstate.*

# Crash log files
crash.log
crash.*.log

# Exclude all .tfvars files, which are likely to contain sensitive data
*.tfvars
*.tfvars.json

# Ignore override files as they are usually used for local dev
override.tf
override.tf.json
*_override.tf
*_override.tf.json

# Ignore CLI configuration files
.terraformrc
terraform.rc

# Terragrunt cache directories
**/.terragrunt-cache/*

# Ignore git directories
.git/
.github/

# Ignore IDE and editor files
.vscode/
.idea/
*.swp
*~

# Ignore OS specific files
.DS_Store
Thumbs.db

# Ignore documentation
*.md
docs/

# Ignore test files
*_test.go
*_test.tf
test/
tests/
`

// TerragruntignoreTemplate is the default template for .terragruntignore
const TerragruntignoreTemplate = `# TGS specific folders
.tgs/
.infrastructure/

# Local .terraform directories
**/.terraform/*

# .tfstate files
*.tfstate
*.tfstate.*

# Crash log files
crash.log
crash.*.log

# Exclude all .tfvars files, which are likely to contain sensitive data
*.tfvars
*.tfvars.json

# Ignore override files as they are usually used for local dev
override.tf
override.tf.json
*_override.tf
*_override.tf.json

# Ignore CLI configuration files
.terraformrc
terraform.rc

# Terragrunt cache directories
**/.terragrunt-cache/*

# Ignore git directories
.git/
.github/

# Ignore IDE and editor files
.vscode/
.idea/
*.swp
*~

# Ignore OS specific files
.DS_Store
Thumbs.db

# Ignore documentation
*.md
docs/

# Ignore test files
*_test.go
*_test.tf
test/
tests/

# Ignore generated backend files
backend.tf
`

// getConfigDir returns the path to the .tgs config directory
func getConfigDir() (string, error) {
	return ".tgs", nil
}

// getStacksDir returns the path to the .tgs/stacks directory
func getStacksDir() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "stacks"), nil
}

// CreateFileIfNotExists creates a file with the given content if it doesn't exist
func CreateFileIfNotExists(path string, content string) error {
	return CreateFileIfNotExistsWithOverwrite(path, content, false)
}

// CreateFileIfNotExistsWithOverwrite creates a file with the given content if it doesn't exist
// If overwriteIfExists is true, it will overwrite the file if it already exists
func CreateFileIfNotExistsWithOverwrite(path string, content string, overwriteIfExists bool) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		if !overwriteIfExists {
			return fmt.Errorf("file %s already exists", path)
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create the file
	return os.WriteFile(path, []byte(content), 0644)
}

// InitProject initializes a new project with tgs.yaml and main.yaml
func InitProject() error {
	fmt.Println("Initializing new project with tgs.yaml...")

	// Create .tgs directory
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	// Create .tgs directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create .tgs directory: %w", err)
	}

	// Create tgs.yaml in .tgs directory
	configPath := filepath.Join(configDir, "tgs.yaml")
	if err := CreateFileIfNotExistsWithOverwrite(configPath, TGSYamlTemplate, false); err != nil {
		// If the file already exists, just log a message and continue
		if os.IsExist(err) || err.Error() == fmt.Sprintf("file %s already exists", configPath) {
			fmt.Printf("File %s already exists, skipping...\n", configPath)
		} else {
			return fmt.Errorf("failed to create tgs.yaml: %w", err)
		}
	} else {
		fmt.Println("Successfully created", configPath)
	}

	// Create .tgs/stacks directory
	stacksDir, err := getStacksDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(stacksDir, 0755); err != nil {
		return fmt.Errorf("failed to create stacks directory %s: %w", stacksDir, err)
	}

	// Create default main.yaml in .tgs/stacks directory
	mainStackPath := filepath.Join(stacksDir, "main.yaml")
	if err := CreateFileIfNotExistsWithOverwrite(mainStackPath, MainYamlTemplate, false); err != nil {
		// If the file already exists, just log a message and continue
		if os.IsExist(err) || err.Error() == fmt.Sprintf("file %s already exists", mainStackPath) {
			fmt.Printf("File %s already exists, skipping...\n", mainStackPath)
		} else {
			return fmt.Errorf("failed to create main.yaml: %w", err)
		}
	} else {
		fmt.Println("Successfully created", mainStackPath)
	}

	// Create .gitignore in project root directory
	gitignorePath := ".gitignore"
	if err := CreateFileIfNotExistsWithOverwrite(gitignorePath, GitignoreTemplate, false); err != nil {
		// If the file already exists, just log a message and continue
		if os.IsExist(err) || err.Error() == fmt.Sprintf("file %s already exists", gitignorePath) {
			fmt.Printf("File %s already exists, skipping...\n", gitignorePath)
		} else {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
	} else {
		fmt.Println("Successfully created", gitignorePath)
	}

	// Create .terraformignore in project root directory
	terraformignorePath := ".terraformignore"
	if err := CreateFileIfNotExistsWithOverwrite(terraformignorePath, TerraformignoreTemplate, false); err != nil {
		// If the file already exists, just log a message and continue
		if os.IsExist(err) || err.Error() == fmt.Sprintf("file %s already exists", terraformignorePath) {
			fmt.Printf("File %s already exists, skipping...\n", terraformignorePath)
		} else {
			return fmt.Errorf("failed to create .terraformignore: %w", err)
		}
	} else {
		fmt.Println("Successfully created", terraformignorePath)
	}

	// Create .terragruntignore in project root directory
	terragruntignorePath := ".terragruntignore"
	if err := CreateFileIfNotExistsWithOverwrite(terragruntignorePath, TerragruntignoreTemplate, false); err != nil {
		// If the file already exists, just log a message and continue
		if os.IsExist(err) || err.Error() == fmt.Sprintf("file %s already exists", terragruntignorePath) {
			fmt.Printf("File %s already exists, skipping...\n", terragruntignorePath)
		} else {
			return fmt.Errorf("failed to create .terragruntignore: %w", err)
		}
	} else {
		fmt.Println("Successfully created", terragruntignorePath)
	}

	fmt.Println("Project initialization complete!")
	return nil
}

// CreateStack creates a new stack configuration file
func CreateStack(name string) error {
	// Create .tgs/stacks directory
	stacksDir, err := getStacksDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(stacksDir, 0755); err != nil {
		return fmt.Errorf("failed to create stacks directory %s: %w", stacksDir, err)
	}

	filename := fmt.Sprintf("%s.yaml", name)
	stackPath := filepath.Join(stacksDir, filename)

	fmt.Printf("Creating new stack configuration: %s...\n", stackPath)

	if err := CreateFileIfNotExists(stackPath, MainYamlTemplate); err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}

	fmt.Printf("Successfully created %s\n", stackPath)
	return nil
}

// ListStacks lists all stack files in the .tgs/stacks directory
func ListStacks() error {
	stacksDir, err := getStacksDir()
	if err != nil {
		return err
	}

	// Check if stacks directory exists
	if _, err := os.Stat(stacksDir); os.IsNotExist(err) {
		fmt.Println("No stacks found. Use 'tgs create stack' to create a stack.")
		return nil
	}

	// Read all files in the stacks directory
	files, err := os.ReadDir(stacksDir)
	if err != nil {
		return fmt.Errorf("failed to read stacks directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No stacks found. Use 'tgs create stack' to create a stack.")
		return nil
	}

	fmt.Println("Available stacks:")
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			stackName := file.Name()[:len(file.Name())-len(".yaml")]
			fmt.Printf("  - %s\n", stackName)
		}
	}

	return nil
}
