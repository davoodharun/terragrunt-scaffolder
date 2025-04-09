# Terragrunt Scaffolder Generation Process

## Overview
The Terragrunt Scaffolder (TGS) is a tool that generates a standardized infrastructure code structure for Azure environments. This document outlines the generation process and code flow.

## Command Flow
1. **Main Command (`cmd/tgs/main.go`)**
   - Entry point for the application
   - Defines root command and subcommands
   - Handles command-line arguments and flags
   - Initializes logger and configuration

2. **Generate Command**
   - Reads TGS configuration from file
   - Validates configuration structure
   - Calls scaffold.Generate() to start generation process

## Generation Process

### 1. Initial Setup (`internal/scaffold/scaffold.go`)
- Creates base infrastructure directory structure
- Initializes progress bar for tracking generation
- Sets up template renderer for file generation

### 2. Configuration Generation
- **Root Configuration (`internal/scaffold/root.go`)**
  - Generates root.hcl with global settings
  - Sets up remote state configuration
  - Configures backend settings

- **Environment Configuration (`internal/scaffold/environment.go`)**
  - Creates environment-specific configurations
  - Sets up subscription and region settings
  - Configures environment variables

### 3. Component Generation (`internal/scaffold/components.go`)
- Processes components in dependency order
- Creates component-specific directories
- Generates component configuration files
- Handles component dependencies

### 4. Architecture Generation (`internal/scaffold/architecture.go`)
- Creates architecture-specific configurations
- Sets up component relationships
- Configures networking and security

## File Structure
The generated infrastructure follows this structure:
```
.infrastructure/
├── root/
│   └── root.hcl
├── config/
│   └── {subscription}/
│       └── {environment}/
│           └── environment.hcl
├── _components/
│   └── {component}/
│       ├── component.hcl
│       └── terraform/
└── _architecture/
    └── {architecture}/
        └── architecture.hcl
```

## Key Functions
1. `Generate()` - Main entry point for generation process
2. `generateRootHCL()` - Creates root configuration
3. `generateEnvironmentConfigs()` - Sets up environment configurations
4. `generateComponentFiles()` - Handles component generation
5. `generateArchitectureFiles()` - Creates architecture configurations

## Progress Tracking
The generation process includes progress tracking:
- Initializes progress bar at start
- Updates progress after each major step
- Shows completion status

## Error Handling
- Validates configuration before generation
- Checks for required directories and files
- Handles template rendering errors
- Provides detailed error messages

## Testing
- Unit tests verify generation process
- Integration tests check file creation
- Dependency handling tests ensure correct order
- Configuration validation tests 