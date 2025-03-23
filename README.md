# terragrunt-scaffolder

A tool for scaffolding Terraform and Terragrunt projects with standardized structure and naming conventions.

## Overview

Terragrunt-scaffolder (tgs) helps you create and manage infrastructure-as-code projects using Terraform and Terragrunt. It generates a consistent directory structure, configuration files, and naming conventions based on your project specifications.

For a detailed understanding of how the tool works:
- [Generation Process Documentation](GENERATION_PROCESS.md) - Learn about the complete generation process and code flow
- [Provider Schema Documentation](PROVIDER_SCHEMA.md) - Understand how the tool interacts with the Azure provider schema

> **Note**: Currently, this tool only supports Azure cloud provider. Support for other cloud providers may be added in future releases.

> **Important**: Before starting, ensure you have an Azure Storage Account created in your subscription. This storage account will be used to store Terraform state files. The storage account should be in a resource group that follows your organization's naming conventions.

## Quick Start

1. **Install the tool**:
   ```bash
   # Download the latest release from GitHub
   # For Windows (PowerShell):
   Invoke-WebRequest -Uri "https://github.com/davoodharun/terragrunt-scaffolder/releases/latest/download/tgs-windows-amd64.exe" -OutFile "tgs.exe"
   Move-Item tgs.exe $env:LOCALAPPDATA\Microsoft\WindowsApps

   # For macOS/Linux:
   curl -L https://github.com/davoodharun/terragrunt-scaffolder/releases/latest/download/tgs-linux-amd64 -o tgs
   chmod +x tgs
   sudo mv tgs /usr/local/bin/
   ```

2. **Authenticate with Azure**:
   ```bash
   # Interactive login (opens browser)
   az login

   # or to use a device code (needed for managing multiple sessions/profiles)
   az login --use-device-code

   # OR using service principal
   az login --service-principal \
     --username <app-id> \
     --password <password-or-cert> \
     --tenant <tenant-id>
   ```

3. **Set the correct subscription**:
   ```bash
   # List available subscriptions
   az account list --output table

   # Set the active subscription
   az account set --subscription "<subscription-name-or-id>"
   ```

4. **Configure Azure Provider Authentication**:
   The tool uses the Azure provider for Terraform. You can authenticate using:
   - Interactive login (default): Uses your Azure CLI credentials
   - Service Principal: Set these environment variables:
     ```bash
     export ARM_CLIENT_ID="<app-id>"
     export ARM_CLIENT_SECRET="<password-or-cert>"
     export ARM_SUBSCRIPTION_ID="<subscription-id>"
     export ARM_TENANT_ID="<tenant-id>"
     ```
   - Managed Identity: No additional configuration needed when running on Azure

5. **Initialize a new project**:
   ```bash
   # Create a new directory for your project
   mkdir my-infrastructure
   cd my-infrastructure

   # Initialize the project with default configuration
   tgs init
   ```
   This creates the `.tgs` directory with a default `tgs.yaml` file and a default `main.yaml` stack.

6. **Configure your project**:
   - Edit `.tgs/tgs.yaml` to set your project name and Azure subscription details
   - Edit `.tgs/stacks/main.yaml` to define your infrastructure components

7. **Generate the infrastructure**:
   ```bash
   tgs generate
   ```
   This creates the Terragrunt configuration in the `.infrastructure` directory. For a detailed explanation of the generation process, see the [Generation Process Documentation](GENERATION_PROCESS.md).

8. **Create the storage container**:
   ```bash
   tgs create container
   ```
   This creates a container in your Azure storage account for storing Terraform state.

9. **Initialize Terragrunt**:
   ```bash
   # Navigate to the infrastructure directory
   cd .infrastructure

   # Initialize Terragrunt for all components
   terragrunt run-all init

   # Or initialize a specific component (e.g., appservice in dev environment)
   cd nonprod/eastus2/dev/appservice
   terragrunt init
   ```
   This initializes the Terraform working directory and downloads required providers.

10. **Plan your changes**:
   ```bash
   # Plan all components
   terragrunt run-all plan

   # Or plan a specific component
   terragrunt plan
   ```
   This shows what changes will be applied to your infrastructure.

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

> **Note**: For detailed information about how the tool interacts with the Azure provider schema, see the [Provider Schema Documentation](PROVIDER_SCHEMA.md).

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