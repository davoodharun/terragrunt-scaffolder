package validate

import (
	"fmt"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/config"
)

// ValidAzureRegions is a map of valid Azure regions
var ValidAzureRegions = map[string]bool{
	"eastus":             true,
	"eastus2":            true,
	"westus":             true,
	"westus2":            true,
	"centralus":          true,
	"northeurope":        true,
	"westeurope":         true,
	"southeastasia":      true,
	"eastasia":           true,
	"japaneast":          true,
	"japanwest":          true,
	"australiaeast":      true,
	"australiasoutheast": true,
	"southindia":         true,
	"centralindia":       true,
	"westindia":          true,
	"canadacentral":      true,
	"canadaeast":         true,
	"uksouth":            true,
	"ukwest":             true,
	"francecentral":      true,
	"francesouth":        true,
	"germanywestcentral": true,
	"norwayeast":         true,
	"switzerlandnorth":   true,
	"uaenorth":           true,
	"brazilsouth":        true,
	"southafricanorth":   true,
}

// ValidAzureResourceTypes is a map of valid Azure resource types
var ValidAzureResourceTypes = map[string]bool{
	"azurerm_service_plan":           true,
	"azurerm_linux_web_app":          true,
	"azurerm_windows_web_app":        true,
	"azurerm_api_management":         true,
	"azurerm_servicebus_namespace":   true,
	"azurerm_cosmosdb_account":       true,
	"azurerm_cosmosdb_sql_database":  true,
	"azurerm_redis_cache":            true,
	"azurerm_key_vault":              true,
	"azurerm_storage_account":        true,
	"azurerm_container_registry":     true,
	"azurerm_kubernetes_cluster":     true,
	"azurerm_application_gateway":    true,
	"azurerm_virtual_network":        true,
	"azurerm_subnet":                 true,
	"azurerm_public_ip":              true,
	"azurerm_network_security_group": true,
}

// ValidationError represents a validation error with context
type ValidationError struct {
	Context string
	Message string
}

func (e ValidationError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s: %s", e.Context, e.Message)
	}
	return e.Message
}

// ValidateStack validates a stack configuration
func ValidateStack(stack *config.MainConfig) []error {
	var errors []error

	// Validate components
	if len(stack.Stack.Components) == 0 {
		errors = append(errors, ValidationError{
			Context: "Stack",
			Message: "no components defined in stack",
		})
	}

	for compName, comp := range stack.Stack.Components {
		compErrors := validateComponent(compName, comp)
		errors = append(errors, compErrors...)
	}

	// Validate architecture
	if len(stack.Stack.Architecture.Regions) == 0 {
		errors = append(errors, ValidationError{
			Context: "Architecture",
			Message: "no regions defined in architecture",
		})
	}

	for region := range stack.Stack.Architecture.Regions {
		if !ValidAzureRegions[region] {
			errors = append(errors, ValidationError{
				Context: fmt.Sprintf("Region '%s'", region),
				Message: "invalid Azure region",
			})
		}
	}

	// Validate component references in architecture
	errors = append(errors, validateArchitectureComponents(stack)...)

	return errors
}

// validateComponent validates a single component configuration
func validateComponent(name string, comp config.Component) []error {
	var errors []error
	context := fmt.Sprintf("Component '%s'", name)

	// Validate required fields
	if comp.Source == "" {
		errors = append(errors, ValidationError{
			Context: context,
			Message: "source is required",
		})
	}

	if comp.Provider == "" {
		errors = append(errors, ValidationError{
			Context: context,
			Message: "provider is required",
		})
	}

	if comp.Version == "" {
		errors = append(errors, ValidationError{
			Context: context,
			Message: "version is required",
		})
	}

	// Validate source is a valid Azure resource type
	if comp.Source != "" && !ValidAzureResourceTypes[comp.Source] {
		errors = append(errors, ValidationError{
			Context: context,
			Message: fmt.Sprintf("invalid Azure resource type: %s", comp.Source),
		})
	}

	// Validate dependencies format
	for _, dep := range comp.Deps {
		parts := strings.Split(dep, ".")
		if len(parts) < 2 {
			errors = append(errors, ValidationError{
				Context: context,
				Message: fmt.Sprintf("invalid dependency format: %s (should be 'region.component' or 'region.component.app')", dep),
			})
			continue
		}

		// Check if the region part is valid (could be a placeholder {region})
		if parts[0] != "{region}" && !ValidAzureRegions[parts[0]] {
			errors = append(errors, ValidationError{
				Context: context,
				Message: fmt.Sprintf("invalid region in dependency: %s", parts[0]),
			})
		}
	}

	return errors
}

// validateArchitectureComponents validates that all components referenced in the architecture exist in the components section
func validateArchitectureComponents(stack *config.MainConfig) []error {
	var errors []error

	for region, components := range stack.Stack.Architecture.Regions {
		for _, comp := range components {
			if _, exists := stack.Stack.Components[comp.Component]; !exists {
				errors = append(errors, ValidationError{
					Context: fmt.Sprintf("Region '%s'", region),
					Message: fmt.Sprintf("component '%s' referenced in architecture but not defined in components", comp.Component),
				})
			}
		}
	}

	return errors
}

// ValidateTGSConfig validates the TGS configuration file
func ValidateTGSConfig(cfg *config.TGSConfig) []error {
	var errors []error

	// Validate project name
	if cfg.Name == "" {
		errors = append(errors, ValidationError{
			Context: "Project Name",
			Message: "project name cannot be empty",
		})
	}

	// Validate subscriptions
	if len(cfg.Subscriptions) == 0 {
		errors = append(errors, ValidationError{
			Context: "Subscriptions",
			Message: "at least one subscription must be defined",
		})
	}

	// Validate each subscription
	for subName, sub := range cfg.Subscriptions {
		// Validate remote state configuration
		if sub.RemoteState.Name == "" {
			errors = append(errors, ValidationError{
				Context: fmt.Sprintf("Subscription '%s' Remote State", subName),
				Message: "storage account name cannot be empty",
			})
		}

		if sub.RemoteState.ResourceGroup == "" {
			errors = append(errors, ValidationError{
				Context: fmt.Sprintf("Subscription '%s' Remote State", subName),
				Message: "resource group name cannot be empty",
			})
		}

		// Validate environments
		if len(sub.Environments) == 0 {
			errors = append(errors, ValidationError{
				Context: fmt.Sprintf("Subscription '%s'", subName),
				Message: "at least one environment must be defined",
			})
		}

		// Validate each environment
		for _, env := range sub.Environments {
			if env.Name == "" {
				errors = append(errors, ValidationError{
					Context: fmt.Sprintf("Subscription '%s' Environment", subName),
					Message: "environment name cannot be empty",
				})
			}
		}
	}

	return errors
}
