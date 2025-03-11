package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/davoodharun/terragrunt-scaffolder/internal/azure"
	"github.com/davoodharun/terragrunt-scaffolder/internal/diagram"
	"github.com/davoodharun/terragrunt-scaffolder/internal/scaffold"
	"github.com/davoodharun/terragrunt-scaffolder/internal/template"
	"github.com/spf13/cobra"
)

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

	// Generate scaffold command
	scaffoldCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate infrastructure scaffold",
		RunE: func(cmd *cobra.Command, args []string) error {
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

	// Add subcommands to create command
	createCmd.AddCommand(createStackCmd)
	createCmd.AddCommand(createContainerCmd)

	// Add commands to root command
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(scaffoldCmd)
	rootCmd.AddCommand(listStacksCmd)
	rootCmd.AddCommand(diagramCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
