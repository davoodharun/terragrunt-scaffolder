# Configuration Guide

This guide provides detailed information about configuring your terragrunt-scaffolder project.

## Table of Contents
- [TGS Configuration](#tgs-configuration)
- [Stack Configuration](#stack-configuration)
- [Environment-Specific Configuration](#environment-specific-configuration)
- [Configuration Fields Reference](#configuration-fields-reference)
- [Dependency Notation](#dependency-notation)
- [Resource Naming Configuration](#resource-naming-configuration)

## TGS Configuration

The `tgs.yaml` file in the `.tgs` directory defines your project settings, Azure subscription details, and resource naming conventions. Here's an example:

```yaml
name: MyProject  # Your project name

# Resource naming configuration
naming:
  format: "${project}-${region}${env}-${type}"  # Default format
  separator: "-"  # Default separator between name parts
  
  # Resource type prefixes
  resource_prefixes:
    appservice: "app"
    serviceplan: "asp"
    rediscache: "redis"
    keyvault: "kv"
    sqlserver: "sql"
    storage: "st"
    cosmosdb: "cosmos"
    servicebus: "sb"
    eventhub: "eh"
    functionapp: "func"
  
  # Optional: Custom formats for specific components
  component_formats:
    keyvault:
      format: "${project}-${type}-${env}"
    storage:
      format: "${project}${type}${env}"
      separator: ""  # No separator for Storage Account names

subscriptions:
  nonprod:
    remotestate:
      name: myprojecttfstatessta000
      resource_group: MyProject-E-N-TFSTATE-RGP
    environments:
      - name: dev
        stack: main
      - name: test
        stack: main
  prod:
    remotestate:
      name: myprojecttfstatesstp000
      resource_group: MyProject-E-P-TFSTATE-RGP
    environments:
      - name: stage
        stack: main
      - name: prod
        stack: main
```

## Stack Configuration

Stack files (e.g., `.tgs/stacks/main.yaml`) define your infrastructure components and their relationships. Here's an example:

```yaml
stack:
  name: main
  version: "1.0.0"
  description: "Main infrastructure stack"
  components:
    redis:
      source: azurerm_redis_cache
      provider: azurerm
      version: "4.22.0"
      description: "Redis cache for caching"
    appservice:
      source: azurerm_app_service
      provider: azurerm
      version: "4.22.0"
      description: "App service for API"
      deps:
        - "{region}.redis"  # Depends on redis in the same region
  architecture:
    regions:
      eastus2:  # Primary region
        - component: redis
          apps: []  # No app-specific instances
        - component: appservice
          apps:
            - api  # Creates an app-specific instance
      westus2:  # Secondary region
        - component: redis
          apps: []
        - component: appservice
          apps:
            - api
```

## Environment-Specific Configuration

The scaffolder generates environment-specific configuration files in `.infrastructure/config/<stack>/<environment>.hcl`. These files allow you to customize component settings per environment:

```hcl
# Example dev.hcl
locals {
  redis = {
    sku_name = "Basic"  # Dev environment uses Basic SKU
    family = "C"
  }
  appservice = {
    sku_name = "B1"  # Dev environment uses Basic tier
    os_type = "Linux"
  }
}
```

For production environments:
```hcl
# Example prod.hcl
locals {
  redis = {
    sku_name = "Premium"  # Prod environment uses Premium SKU
    family = "P"
  }
  appservice = {
    sku_name = "P1v2"  # Prod environment uses Premium v2 tier
    os_type = "Linux"
  }
}
```

## Configuration Fields Reference

### TGS Configuration Fields
- `name`: Project identifier used in resource naming
- `naming`: Resource naming configuration
  - `format`: Default naming format using variables
  - `separator`: Default separator between name parts
  - `resource_prefixes`: Map of resource type abbreviations
  - `component_formats`: Custom formats for specific components
- `subscriptions`: Map of Azure subscriptions
  - `remotestate`: Terraform state storage configuration
    - `name`: Azure Storage Account name
    - `resource_group`: Resource group name
  - `environments`: List of environments in this subscription
    - `name`: Environment name (e.g., dev, test, prod)
    - `stack`: Reference to a stack file (defaults to "main")

### Stack Configuration Fields
- `name`: Stack identifier
- `version`: Stack version for tracking changes
- `description`: Stack purpose description
- `components`: Map of infrastructure components
  - `source`: Azure resource type
  - `provider`: Cloud provider (e.g., azurerm)
  - `version`: Provider version
  - `description`: Component purpose
  - `deps`: List of dependencies (format: "{region}.component[.app]")
- `architecture`: Regional component deployment
  - `regions`: Map of Azure regions
    - `component`: Component to deploy
    - `apps`: List of app-specific instances

## Dependency Notation

Dependencies are specified using the format: `[region].[component].[app]`

### Special Placeholders
- `{region}`: Replaced with the current region being processed
- `{app}`: Replaced with the current app being processed

### Examples
1. Fixed region and component:
   ```yaml
   deps:
     - "eastus2.redis"
   ```

2. Current region with fixed component:
   ```yaml
   deps:
     - "{region}.serviceplan"
   ```

3. Current region, fixed component, and current app:
   ```yaml
   deps:
     - "{region}.serviceplan.{app}"
   ```

## Resource Naming Configuration

The `naming` section in `tgs.yaml` allows you to customize how resources are named across your infrastructure.

### Default Naming Format
```yaml
naming:
  format: "${project}-${region}${env}-${type}"
  separator: "-"
```

Available variables:
- `${project}`: Project name from the root level
- `${region}`: Region prefix (e.g., e2 for eastus2)
- `${env}`: Environment prefix (e.g., d for dev)
- `${type}`: Resource type prefix from resource_prefixes
- `${app}`: Application name (if applicable)

### Resource Type Prefixes
Define standard abbreviations for each resource type:
```yaml
resource_prefixes:
  appservice: "app"
  serviceplan: "asp"
  rediscache: "redis"
  keyvault: "kv"
  sqlserver: "sql"
  storage: "st"
  cosmosdb: "cosmos"
  servicebus: "sb"
  eventhub: "eh"
  functionapp: "func"
```

### Component-Specific Formats
Override the default format for specific components:
```yaml
component_formats:
  keyvault:
    format: "${project}-${type}-${env}"  # Custom format for Key Vault
  storage:
    format: "${project}${type}${env}"    # Custom format for Storage Account
    separator: ""                         # No separator for Storage Account names
```

### Examples

1. **Default Format**:
   - Configuration: `${project}-${region}${env}-${type}`
   - Result: `MyProject-e2d-app` (for an App Service in eastus2 dev)

2. **Storage Account** (custom format):
   - Configuration: `${project}${type}${env}`
   - Result: `myprojectstdev` (no separators for storage account compatibility)

3. **Key Vault** (custom format):
   - Configuration: `${project}-${type}-${env}`
   - Result: `MyProject-kv-dev`

4. **App-specific Resource**:
   - Configuration: `${project}-${region}${env}-${type}-${app}`
   - Result: `MyProject-e2d-app-api` (for an API App Service)

### Naming Best Practices

1. **Storage Accounts**:
   - Use no separators
   - Keep names lowercase
   - Maximum 24 characters
   ```yaml
   component_formats:
     storage:
       format: "${project}${type}${env}"
       separator: ""
   ```

2. **Key Vaults**:
   - Use separators for readability
   - Include environment for security boundaries
   ```yaml
   component_formats:
     keyvault:
       format: "${project}-${type}-${env}"
   ```

3. **General Resources**:
   - Use consistent separators
   - Include region and environment for clear identification
   - Keep the format readable and meaningful
   ```yaml
   format: "${project}-${region}${env}-${type}"
   ``` 