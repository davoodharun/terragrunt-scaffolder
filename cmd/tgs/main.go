package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/azure"
	"github.com/davoodharun/terragrunt-scaffolder/internal/diagram"
	"github.com/davoodharun/terragrunt-scaffolder/internal/scaffold"
	"github.com/davoodharun/terragrunt-scaffolder/internal/template"
	"github.com/davoodharun/terragrunt-scaffolder/internal/validate"
	"github.com/spf13/cobra"
)

// detailsCmd shows detailed information about a stack
var detailsCmd = &cobra.Command{
	Use:   "details [stack]",
	Short: "Show detailed information about a stack configuration",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stackName := "main"
		if len(args) > 0 {
			stackName = args[0]
		}

		// Read the stack configuration
		mainConfig, err := scaffold.ReadMainConfig(stackName)
		if err != nil {
			return fmt.Errorf("failed to read stack config: %w", err)
		}

		// Print stack details
		fmt.Printf("\nStack: %s\n", mainConfig.Stack.Name)
		fmt.Printf("Version: %s\n", mainConfig.Stack.Version)
		fmt.Printf("Description: %s\n\n", mainConfig.Stack.Description)

		// Group components by type
		componentTypes := make(map[string][]string)
		for name, comp := range mainConfig.Stack.Components {
			resourceType := strings.TrimPrefix(comp.Source, "azurerm_")
			componentTypes[resourceType] = append(componentTypes[resourceType], name)
		}

		// Sort resource types for consistent output
		var types []string
		for t := range componentTypes {
			types = append(types, t)
		}
		sort.Strings(types)

		fmt.Println("Resources:")
		fmt.Println("----------")
		for _, resourceType := range types {
			components := componentTypes[resourceType]
			sort.Strings(components)
			fmt.Printf("\n%s:\n", strings.ReplaceAll(resourceType, "_", " "))
			for _, comp := range components {
				fmt.Printf("  - %s: %s\n", comp, mainConfig.Stack.Components[comp].Description)
			}
		}

		fmt.Println("\nRegions:")
		fmt.Println("--------")
		for region, components := range mainConfig.Stack.Architecture.Regions {
			fmt.Printf("\n%s:\n", region)
			var compNames []string
			for _, comp := range components {
				compNames = append(compNames, comp.Component)
			}
			sort.Strings(compNames)
			for _, comp := range compNames {
				fmt.Printf("  - %s\n", comp)
			}
		}

		return nil
	},
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "tgs",
		Short: "TGS - Terraform Generator Scaffold",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Initialize a new project with tgs.yaml
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new project with tgs.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return template.InitProject()
		},
	}

	// Create command with subcommands
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create various configuration files",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Create stack subcommand
	createStackCmd := &cobra.Command{
		Use:   "stack [name]",
		Short: "Create a new stack configuration (main.yaml)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stackName := "main"
			if len(args) > 0 {
				stackName = args[0]
			}

			return template.CreateStack(stackName)
		},
	}

	// Create container subcommand
	createContainerCmd := &cobra.Command{
		Use:   "container",
		Short: "Create a container in a storage account",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read TGS config to get storage accounts
			tgsConfig, err := scaffold.ReadTGSConfig()
			if err != nil {
				return fmt.Errorf("failed to read TGS config: %w", err)
			}

			// Create a map of available storage accounts
			storageAccounts := make(map[int]struct {
				name string
				sub  string
			})
			i := 1

			fmt.Println("\nAvailable storage accounts:")
			for subName, sub := range tgsConfig.Subscriptions {
				fmt.Printf("%d. %s (Subscription: %s)\n", i, sub.RemoteState.Name, subName)
				storageAccounts[i] = struct {
					name string
					sub  string
				}{
					name: sub.RemoteState.Name,
					sub:  subName,
				}
				i++
			}

			// Get user input
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("\nEnter the number of the storage account to use: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			// Parse user input
			input = strings.TrimSpace(input)
			choice := 0
			_, err = fmt.Sscanf(input, "%d", &choice)
			if err != nil || choice < 1 || choice >= i {
				return fmt.Errorf("invalid selection: please enter a number between 1 and %d", i-1)
			}

			selectedAccount := storageAccounts[choice]
			fmt.Printf("\nCreating container '%s' in storage account '%s' (Subscription: %s)...\n",
				tgsConfig.Name, selectedAccount.name, selectedAccount.sub)

			// Create the container using Azure SDK
			if err := azure.CreateContainer(selectedAccount.name, tgsConfig.Name); err != nil {
				return fmt.Errorf("failed to create container: %w", err)
			}

			fmt.Printf("\nSuccessfully created container '%s' in storage account '%s'\n",
				tgsConfig.Name, selectedAccount.name)

			return nil
		},
	}

	// List stacks command
	listStacksCmd := &cobra.Command{
		Use:   "list",
		Short: "List available stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return template.ListStacks()
		},
	}

	// Validate stack command
	validateCmd := &cobra.Command{
		Use:   "validate [stack]",
		Short: "Validate a stack configuration",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stackName := "main"
			if len(args) > 0 {
				stackName = args[0]
			}

			// Read the stack configuration
			mainConfig, err := scaffold.ReadMainConfig(stackName)
			if err != nil {
				return fmt.Errorf("failed to read stack config: %w", err)
			}

			// Validate the stack
			if errors := validate.ValidateStack(mainConfig); len(errors) > 0 {
				fmt.Println("Stack validation failed:")
				for _, err := range errors {
					fmt.Printf("  - %v\n", err)
				}
				return fmt.Errorf("stack validation failed with %d errors", len(errors))
			}

			fmt.Printf("Stack '%s' validation successful\n", stackName)
			return nil
		},
	}

	// Validate tgs.yaml command
	validateTGSCmd := &cobra.Command{
		Use:   "validate-config",
		Short: "Validate the tgs.yaml configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read TGS config
			tgsConfig, err := scaffold.ReadTGSConfig()
			if err != nil {
				return fmt.Errorf("failed to read TGS config: %w", err)
			}

			// Validate the configuration
			if errors := validate.ValidateTGSConfig(tgsConfig); len(errors) > 0 {
				fmt.Println("TGS configuration validation failed:")
				for _, err := range errors {
					fmt.Printf("  - %v\n", err)
				}
				return fmt.Errorf("tgs.yaml validation failed with %d errors", len(errors))
			}

			fmt.Println("TGS configuration validation successful")
			return nil
		},
	}

	// Generate scaffold command
	scaffoldCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate infrastructure scaffold",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read TGS config first
			tgsConfig, err := scaffold.ReadTGSConfig()
			if err != nil {
				return fmt.Errorf("failed to read TGS config: %w", err)
			}

			// Validate tgs.yaml first
			if errors := validate.ValidateTGSConfig(tgsConfig); len(errors) > 0 {
				fmt.Println("TGS configuration validation failed:")
				for _, err := range errors {
					fmt.Printf("  - %v\n", err)
				}
				return fmt.Errorf("tgs.yaml validation failed with %d errors", len(errors))
			}
			fmt.Println("TGS configuration validation successful")

			// Track processed stacks to avoid duplicate validation
			processedStacks := make(map[string]bool)

			// Validate all stacks referenced in environments
			for _, sub := range tgsConfig.Subscriptions {
				for _, env := range sub.Environments {
					stackName := "main"
					if env.Stack != "" {
						stackName = env.Stack
					}

					// Skip if we've already validated this stack
					if processedStacks[stackName] {
						continue
					}
					processedStacks[stackName] = true

					fmt.Printf("Validating stack '%s'...\n", stackName)

					// Read and validate the stack
					mainConfig, err := scaffold.ReadMainConfig(stackName)
					if err != nil {
						return fmt.Errorf("failed to read stack config %s: %w", stackName, err)
					}

					if errors := validate.ValidateStack(mainConfig); len(errors) > 0 {
						fmt.Printf("Stack '%s' validation failed:\n", stackName)
						for _, err := range errors {
							fmt.Printf("  - %v\n", err)
						}
						return fmt.Errorf("stack '%s' validation failed with %d errors", stackName, len(errors))
					}

					fmt.Printf("Stack '%s' validation successful\n", stackName)
				}
			}

			fmt.Println("All configurations validated successfully, proceeding with generation...")

			// If all validations pass, proceed with generation
			return scaffold.Generate()
		},
	}

	// Generate diagram command
	diagramCmd := &cobra.Command{
		Use:   "diagram",
		Short: "Generate a Mermaid diagram of the infrastructure layout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return diagram.GenerateDiagram()
		},
	}

	// Plan command
	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Show changes that will be applied to the infrastructure",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read TGS config first
			tgsConfig, err := scaffold.ReadTGSConfig()
			if err != nil {
				return fmt.Errorf("failed to read TGS config: %w", err)
			}

			// Validate tgs.yaml first
			if errors := validate.ValidateTGSConfig(tgsConfig); len(errors) > 0 {
				fmt.Println("TGS configuration validation failed:")
				for _, err := range errors {
					fmt.Printf("  - %v\n", err)
				}
				return fmt.Errorf("tgs.yaml validation failed with %d errors", len(errors))
			}

			return scaffold.Plan()
		},
	}

	// Add subcommands to create command
	createCmd.AddCommand(createStackCmd)
	createCmd.AddCommand(createContainerCmd)

	// Add commands to root command
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(scaffoldCmd)
	rootCmd.AddCommand(listStacksCmd)
	rootCmd.AddCommand(diagramCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(validateTGSCmd)
	rootCmd.AddCommand(detailsCmd)
	rootCmd.AddCommand(planCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
