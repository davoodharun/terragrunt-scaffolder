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
	"azurerm_service_plan":                          true,
	"azurerm_linux_web_app":                         true,
	"azurerm_windows_web_app":                       true,
	"azurerm_app_service":                           true,
	"azurerm_app_service_plan":                      true,
	"azurerm_api_management":                        true,
	"azurerm_servicebus_namespace":                  true,
	"azurerm_cosmosdb_account":                      true,
	"azurerm_cosmosdb_sql_database":                 true,
	"azurerm_redis_cache":                           true,
	"azurerm_key_vault":                             true,
	"azurerm_storage_account":                       true,
	"azurerm_container_registry":                    true,
	"azurerm_kubernetes_cluster":                    true,
	"azurerm_application_gateway":                   true,
	"azurerm_virtual_network":                       true,
	"azurerm_subnet":                                true,
	"azurerm_public_ip":                             true,
	"azurerm_network_security_group":                true,
	"azurerm_eventhub":                              true,
	"azurerm_eventhub_namespace":                    true,
	"azurerm_linux_function_app":                    true,
	"azurerm_windows_function_app":                  true,
	"azurerm_function_app":                          true,
	"azurerm_log_analytics_workspace":               true,
	"azurerm_sql_server":                            true,
	"azurerm_sql_database":                          true,
	"azurerm_monitor_diagnostic_setting":            true,
	"azurerm_monitor_action_group":                  true,
	"azurerm_monitor_metric_alert":                  true,
	"azurerm_monitor_activity_log_alert":            true,
	"azurerm_private_endpoint":                      true,
	"azurerm_private_dns_zone":                      true,
	"azurerm_private_dns_zone_virtual_network_link": true,
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

// ValidateStack validates a stack configuration according to Testing-Framework.md specifications
func ValidateStack(stack *config.MainConfig) []error {
	var errors []error

	// Validate stack name
	if stack.Stack.Name == "" {
		errors = append(errors, ValidationError{
			Context: "Stack",
			Message: "name property must be filled",
		})
	}

	// Validate version
	if stack.Stack.Version == "" {
		errors = append(errors, ValidationError{
			Context: "Stack",
			Message: "version property must be filled",
		})
	}

	// Validate description
	if stack.Stack.Description == "" {
		errors = append(errors, ValidationError{
			Context: "Stack",
			Message: "description property must be filled",
		})
	}

	// Validate components
	if len(stack.Stack.Components) == 0 {
		errors = append(errors, ValidationError{
			Context: "Stack",
			Message: "at least one component must be defined",
		})
	}

	// Validate each component
	for compName, comp := range stack.Stack.Components {
		compErrors := validateComponent(compName, comp)
		errors = append(errors, compErrors...)
	}

	// Validate architecture
	if len(stack.Stack.Architecture.Regions) == 0 {
		errors = append(errors, ValidationError{
			Context: "Architecture",
			Message: "at least one region must be defined",
		})
	}

	// Validate component references in architecture
	errors = append(errors, validateArchitectureComponents(stack)...)

	// Validate dependencies
	errors = append(errors, validateDependencies(stack)...)

	return errors
}

// validateComponent validates a single component configuration
func validateComponent(name string, comp config.Component) []error {
	var errors []error

	// Validate required fields
	if comp.Source == "" {
		errors = append(errors, ValidationError{
			Context: fmt.Sprintf("Component '%s'", name),
			Message: "source property must be filled",
		})
	}

	if comp.Provider == "" {
		errors = append(errors, ValidationError{
			Context: fmt.Sprintf("Component '%s'", name),
			Message: "provider property must be filled",
		})
	}

	if comp.Version == "" {
		errors = append(errors, ValidationError{
			Context: fmt.Sprintf("Component '%s'", name),
			Message: "version property must be filled",
		})
	}

	if comp.Description == "" {
		errors = append(errors, ValidationError{
			Context: fmt.Sprintf("Component '%s'", name),
			Message: "description property must be filled",
		})
	}

	// Validate source is a valid Azure resource type
	if comp.Source != "" && !ValidAzureResourceTypes[comp.Source] {
		errors = append(errors, ValidationError{
			Context: fmt.Sprintf("Component '%s'", name),
			Message: fmt.Sprintf("invalid Azure resource type: %s", comp.Source),
		})
	}

	// Validate dependencies format
	for _, dep := range comp.Deps {
		parts := strings.Split(dep, ".")
		if len(parts) < 2 {
			errors = append(errors, ValidationError{
				Context: fmt.Sprintf("Component '%s'", name),
				Message: fmt.Sprintf("invalid dependency format: %s (should be 'region.component' or 'region.component.app')", dep),
			})
			continue
		}

		// Check if the region part is valid (could be a placeholder {region})
		if parts[0] != "{region}" && !ValidAzureRegions[parts[0]] {
			errors = append(errors, ValidationError{
				Context: fmt.Sprintf("Component '%s'", name),
				Message: fmt.Sprintf("invalid region in dependency: %s", parts[0]),
			})
		}
	}

	return errors
}

// validateArchitectureComponents validates component references in the architecture
func validateArchitectureComponents(stack *config.MainConfig) []error {
	var errors []error

	for region, components := range stack.Stack.Architecture.Regions {
		for _, comp := range components {
			// Check if component exists in components section
			if _, exists := stack.Stack.Components[comp.Component]; !exists {
				errors = append(errors, ValidationError{
					Context: fmt.Sprintf("Region '%s'", region),
					Message: fmt.Sprintf("component '%s' referenced in architecture but not defined in components section", comp.Component),
				})
			}
		}
	}

	return errors
}

// validateDependencies validates component dependencies
func validateDependencies(stack *config.MainConfig) []error {
	var errors []error

	// First, build a map of components that are actually used in the architecture
	usedComponents := make(map[string]bool)
	for _, components := range stack.Stack.Architecture.Regions {
		for _, comp := range components {
			usedComponents[comp.Component] = true
		}
	}

	for compName, comp := range stack.Stack.Components {
		for _, dep := range comp.Deps {
			// Parse dependency string
			parts := strings.Split(dep, ".")
			if len(parts) < 2 {
				errors = append(errors, ValidationError{
					Context: fmt.Sprintf("Component '%s'", compName),
					Message: fmt.Sprintf("invalid dependency format: %s", dep),
				})
				continue
			}

			// Get the component name from the dependency
			depComponent := parts[1]

			// Check if the component exists in the components section
			if _, exists := stack.Stack.Components[depComponent]; !exists {
				errors = append(errors, ValidationError{
					Context: fmt.Sprintf("Component '%s'", compName),
					Message: fmt.Sprintf("dependency references non-existent component '%s'", depComponent),
				})
				continue
			}

			// Check if the component is actually used in the architecture
			if !usedComponents[depComponent] {
				errors = append(errors, ValidationError{
					Context: fmt.Sprintf("Component '%s'", compName),
					Message: fmt.Sprintf("dependency references component '%s' which is defined but not used in the architecture", depComponent),
				})
				continue
			}

			// If using concrete region (not {region}), validate it exists
			region := parts[0]
			if !strings.Contains(region, "{region}") {
				if _, exists := stack.Stack.Architecture.Regions[region]; !exists {
					errors = append(errors, ValidationError{
						Context: fmt.Sprintf("Component '%s'", compName),
						Message: fmt.Sprintf("dependency references non-existent region: %s", region),
					})
				}
			}

			// If app is specified, validate it exists in the architecture
			if len(parts) > 2 {
				app := parts[2]
				// Skip validation if using {app} template
				if app == "{app}" {
					continue
				}

				// For concrete apps, verify they exist in the architecture
				found := false
				// If using {region}, check all regions
				if region == "{region}" {
					for _, regionComps := range stack.Stack.Architecture.Regions {
						for _, rc := range regionComps {
							if rc.Component == depComponent {
								for _, a := range rc.Apps {
									if a == app {
										found = true
										break
									}
								}
							}
						}
						if found {
							break
						}
					}
				} else if regionComps, exists := stack.Stack.Architecture.Regions[region]; exists {
					// For concrete region, check only that region
					for _, rc := range regionComps {
						if rc.Component == depComponent {
							for _, a := range rc.Apps {
								if a == app {
									found = true
									break
								}
							}
						}
					}
				}

				if !found {
					errors = append(errors, ValidationError{
						Context: fmt.Sprintf("Component '%s'", compName),
						Message: fmt.Sprintf("dependency references non-existent app '%s' for component '%s'", app, depComponent),
					})
				}
			}
		}
	}

	return errors
}

// ValidateTGSConfig validates the TGS configuration file according to Testing-Framework.md specifications
func ValidateTGSConfig(cfg *config.TGSConfig) []error {
	var errors []error

	// Validate project name
	if cfg.Name == "" {
		errors = append(errors, ValidationError{
			Context: "Project Name",
			Message: "name property must be filled",
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
		// Validate remote state
		if sub.RemoteState.Name == "" {
			errors = append(errors, ValidationError{
				Context: fmt.Sprintf("Subscription '%s'", subName),
				Message: "remotestate.name property must be filled",
			})
		}

		if sub.RemoteState.ResourceGroup == "" {
			errors = append(errors, ValidationError{
				Context: fmt.Sprintf("Subscription '%s'", subName),
				Message: "remotestate.resource_group property must be filled",
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
		for i, env := range sub.Environments {
			if env.Name == "" {
				errors = append(errors, ValidationError{
					Context: fmt.Sprintf("Subscription '%s' Environment %d", subName, i+1),
					Message: "environment name must be filled",
				})
			}
		}
	}

	return errors
}
