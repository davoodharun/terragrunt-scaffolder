# terragrunt-scaffolder

A tool for scaffolding Terraform and Terragrunt projects with standardized structure and naming conventions.

## Overview

Terragrunt-scaffolder (tgs) helps you create and manage infrastructure-as-code projects using Terraform and Terragrunt. It generates a consistent directory structure, configuration files, and naming conventions based on your project specifications.

> **Note**: Currently, this tool only supports Azure cloud provider. Support for other cloud providers may be added in future releases.

## Prerequisites

Before using this tool, ensure you have the following prerequisites installed and configured:

1. **Terraform** (v1.0.0 or later)
   - Download and install from [Terraform's official website](https://www.terraform.io/downloads.html)
   - Verify installation: `terraform version`

2. **Terragrunt** (v0.45.0 or later)
   - Download and install from [Terragrunt's releases page](https://github.com/gruntwork-io/terragrunt/releases)
   - Verify installation: `terragrunt --version`

3. **Azure Subscription**
   - An active Azure subscription with appropriate permissions
   - Azure CLI installed and configured with your subscription
   - Verify configuration: `az account show`

## Provider Setup

The tool requires proper configuration of the Azure provider to interact with Azure resources. Here's how to set it up:

### Azure Provider Configuration

1. **Install Azure CLI** (if not already installed):
   ```bash
   # Windows (PowerShell)
   winget install -e --id Microsoft.AzureCLI

   # macOS (Homebrew)
   brew install azure-cli

   # Linux (Ubuntu/Debian)
   curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
   ```

2. **Login to Azure**:
   ```bash
   az login
   ```
   This will open a browser window for authentication.

3. **Set the correct subscription**:
   ```bash
   # List available subscriptions
   az account list --output table

   # Set the active subscription
   az account set --subscription "<subscription-name-or-id>"
   ```

4. **Create a Service Principal** (for CI/CD or non-interactive use):
   ```bash
   # Create the service principal
   az ad sp create-for-rbac --name "tgs-sp" --role contributor \
     --scopes /subscriptions/<subscription-id> \
     --sdk-auth

   # The output will look like this:
   {
     "clientId": "<client-id>",
     "clientSecret": "<client-secret>",
     "subscriptionId": "<subscription-id>",
     "tenantId": "<tenant-id>",
     "activeDirectoryEndpointUrl": "https://login.microsoftonline.com",
     "resourceManagerEndpointUrl": "https://management.azure.com/",
     "activeDirectoryGraphResourceId": "https://graph.windows.net/",
     "sqlManagementEndpointUrl": "https://management.core.windows.net:8443/",
     "galleryEndpointUrl": "https://gallery.azure.com/",
     "managementEndpointUrl": "https://management.core.windows.net/"
   }
   ```

5. **Configure Environment Variables** (if using service principal):
   ```bash
   # Set environment variables
   export ARM_CLIENT_ID="<client-id>"
   export ARM_CLIENT_SECRET="<client-secret>"
   export ARM_SUBSCRIPTION_ID="<subscription-id>"
   export ARM_TENANT_ID="<tenant-id>"
   ```

6. **Verify Provider Access**:
   ```bash
   # Test Azure CLI authentication
   az account show

   # Test Terraform provider access
   terraform init
   ```

### Provider Version Requirements

The tool uses the following provider versions by default:
- Azure Provider: `~> 4.22.0`

You can specify a different version in your stack configuration if needed:

```yaml
stack:
  components:
    appservice:
      source: azurerm_app_service
      provider: azurerm
      version: "~> 4.22.0"  # Specify your desired version
```

## Directory Structure

The tool uses the following directory structure:

```
.
├── .tgs/                       # Configuration directory
│   ├── tgs.yaml                # Main configuration file
│   └── stacks/                 # Stack configurations
│       ├── main.yaml           # Default stack
│       ├── dev.yaml            # Dev stack
│       └── prod.yaml           # Prod stack
│
└── .infrastructure/            # Generated infrastructure code
    ├── config/                 # Global configuration
    │   └── global.hcl          # Global variables
    ├── root.hcl                # Root Terragrunt configuration
    ├── _components/            # Component templates
    │   ├── appservice/         # App Service component
    │   │   ├── component.hcl   # Component-level configuration
    │   │   ├── main.tf         # Main Terraform configuration
    │   │   ├── variables.tf    # Input variables
    │   │   └── provider.tf     # Provider configuration
    │   ├── rediscache/         # Redis Cache component
    │   │   ├── component.hcl   # Component-level configuration
    │   │   ├── main.tf         # Main Terraform configuration
    │   │   ├── variables.tf    # Input variables
    │   │   └── provider.tf     # Provider configuration
    │   └── ...                 # Other components
    ├── nonprod/                # Non-production subscription
    │   ├── subscription.hcl    # Subscription-level configuration
    │   ├── eastus2/            # Region
    │   │   ├── region.hcl      # Region-level configuration
    │   │   ├── dev/            # Environment
    │   │   │   ├── environment.hcl  # Environment-level configuration
    │   │   │   ├── appservice/ # Component
    │   │   │   │   ├── terragrunt.hcl  # App-specific configuration
    │   │   │   │   ├── api/    # App
    │   │   │   │   │   └── terragrunt.hcl  # App-specific configuration
    │   │   │   │   └── web/    # App
    │   │   │   │       └── terragrunt.hcl  # App-specific configuration
    │   │   │   └── ...         # Other components
    │   │   └── test/           # Environment
    │   │       ├── environment.hcl  # Environment-level configuration
    │   │       └── ...         # Components and apps
    │   └── westus/             # Region
    │       ├── region.hcl      # Region-level configuration
    │       └── ...             # Environments and components
    └── prod/                   # Production subscription
        ├── subscription.hcl    # Subscription-level configuration
        └── ...                 # Similar structure
```

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
  config_path = "${get_repo_root()}/.infrastructure/${local.subscription_name}/westus/${local.environment_name}/serviceplan/api"
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
tgs init                  # Initialize a new project with tgs.yaml and main.yaml in .tgs directory
tgs create stack [name]   # Create a new stack configuration in .tgs/stacks directory
tgs create container      # Create a container in the storage account for remote state
tgs list                  # List available stacks in .tgs/stacks directory
tgs generate              # Generate Terragrunt configuration based on tgs.yaml and main.yaml
tgs plan                  # Show changes that will be applied to the infrastructure
tgs validate [stack]      # Validate a stack configuration (defaults to main stack)
tgs validate-config       # Validate the tgs.yaml configuration file
tgs details [stack]       # Show detailed information about a stack configuration
tgs diagram              # Generate a Mermaid diagram of the infrastructure layout
```

### Workflow

1. **Initialize the project**:
   ```
   tgs init
   ```
   This creates the `.tgs` directory with a default `tgs.yaml` file and a default `main.yaml` stack in the `.tgs/stacks` directory.

2. **Create additional stacks** (optional):
   ```
   tgs create stack dev
   ```
   This creates a new stack configuration file at `.tgs/stacks/dev.yaml`.

3. **Create storage container** (required for remote state):
   ```
   tgs create container
   ```
   This creates a container in the storage account specified in your `tgs.yaml` for storing Terraform state.

4. **List available stacks**:
   ```
   tgs list
   ```
   This lists all available stacks in the `.tgs/stacks` directory.

5. **Validate configurations**:
   ```
   tgs validate-config    # Validate tgs.yaml
   tgs validate dev      # Validate a specific stack
   ```
   This ensures your configurations are valid before generating infrastructure.

6. **View stack details**:
   ```
   tgs details dev
   ```
   This shows detailed information about a stack's components, architecture, and dependencies.

7. **Generate infrastructure**:
   ```
   tgs generate
   ```
   This generates the Terragrunt configuration based on the configuration files.

8. **Plan changes**:
   ```
   tgs plan
   ```
   This shows what changes would be applied to your infrastructure, including additions, removals, and modifications.

9. **Generate diagrams**:
   ```
   tgs diagram
   ```
   This generates Mermaid diagrams showing the infrastructure layout in the `.infrastructure/diagrams` directory.
