# terragrunt-scaffolder

A tool for scaffolding Terraform and Terragrunt projects with standardized structure and naming conventions.

## Overview

Terragrunt-scaffolder (tgs) helps you create and manage infrastructure-as-code projects using Terraform and Terragrunt. It generates a consistent directory structure, configuration files, and naming conventions based on your project specifications.

## Configuration Files

### tgs.yaml

The `tgs.yaml` file is the primary configuration file that defines your project's subscriptions, environments, and remote state configuration.

#### Schema

```yaml
name: <project_name>                      # Name of your project
subscriptions:                            # Map of subscription configurations
  <subscription_name>:                    # Name of the subscription (e.g., nonprod, prod)
    remotestate:                          # Remote state configuration
      name: <storage_account_name>        # Name of the storage account for remote state
      resource_group: <resource_group>    # Resource group containing the storage account
    environments:                         # List of environments in this subscription
      - name: <environment_name>          # Name of the environment (e.g., dev, test, prod)
        stack: <stack_name>               # Name of the stack to use for this environment
```

#### Example

```yaml
name: CUSTTP
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
        stack: main
```

### main.yaml (Stack Configuration)

The `main.yaml` file defines a stack, which includes components and their architecture. A stack represents a collection of infrastructure components and their deployment configuration.

#### Schema

```yaml
stack:
  components:                             # Map of components to be deployed
    <component_name>:                     # Name of the component (e.g., appservice, rediscache)
      source: <terraform_source>          # Terraform module source
      provider: <provider_name>           # Optional: Provider name (e.g., azurerm)
      version: <provider_version>         # Optional: Provider version
      deps:                               # Optional: List of dependencies
        - <dependency_path>               # Dependency path in format: region.component[.app]
  architecture:                           # Deployment architecture
    regions:                              # Map of regions
      <region_name>:                      # Name of the region (e.g., eastus2, westus)
        - component: <component_name>     # Component to deploy in this region
          apps:                           # Optional: List of apps for this component
            - <app_name>                  # Name of the app
```

#### Example

```yaml
stack:
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
            - web
```

## Dependency Notation

The `deps` field in the component configuration uses a special notation to define dependencies between components. This notation follows the format:

```
[region].[component].[app]
```

Where:
- `region`: The region where the dependency is deployed (e.g., eastus2, westus)
- `component`: The name of the component that is a dependency (e.g., serviceplan, rediscache)
- `app`: Optional. The specific app instance of the component (e.g., api, web)

### Special Placeholders

The dependency notation supports special placeholders that are replaced at generation time:

- `{region}`: Replaced with the current region being processed
- `{app}`: Replaced with the current app being processed

### Examples

1. **Fixed Region and Component**:
   ```yaml
   deps:
     - "eastus2.redis"
   ```
   This creates a dependency on the redis component in the eastus2 region.

2. **Current Region with Fixed Component**:
   ```yaml
   deps:
     - "{region}.serviceplan"
   ```
   This creates a dependency on the serviceplan component in the same region as the current component.

3. **Current Region, Fixed Component, and Current App**:
   ```yaml
   deps:
     - "{region}.serviceplan.{app}"
   ```
   This creates a dependency on the serviceplan component for the same app in the same region. For example, if processing the "api" app in "westus", this would resolve to "westus.serviceplan.api".

4. **Fixed Region, Component, and App**:
   ```yaml
   deps:
     - "eastus2.cosmos_db.api"
   ```
   This creates a dependency on the specific "api" instance of the cosmos_db component in eastus2.

### Dependency Resolution

When generating the Terragrunt configuration, these dependencies are converted into Terragrunt dependency blocks. For example:

```hcl
dependency "serviceplan" {
  config_path = "${get_repo_root()}/${local.infrastructure_path}/westus/dev/serviceplan/api"
}
```

This allows components to reference outputs from their dependencies using the `dependency.serviceplan.outputs` syntax in Terragrunt.

## Naming Conventions

Resources are named using the following convention:
```
{project_name}-{region_prefix}{environment_prefix}-{app_name}
```

For example:
- `CUSTTP-ED-api` for an API app in eastus dev environment
- `CUSTTP-WP-web` for a web app in westus prod environment

Region and environment prefixes are single-letter codes:
- Regions: E (eastus), W (westus), E2 (eastus2), etc.
- Environments: D (dev), T (test), P (prod), etc.

## Usage

See the CLI commands section for details on how to use the tool.

## CLI Commands

```
tgs init                  # Initialize a new project with tgs.yaml
tgs create stack          # Create a new stack configuration (main.yaml)
tgs generate              # Generate Terragrunt configuration based on tgs.yaml and main.yaml
```
