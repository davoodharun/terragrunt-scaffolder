package main

import (
	"fmt"
	"os"

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

	// Add subcommands to create command
	createCmd.AddCommand(createStackCmd)

	// Add commands to root command
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(scaffoldCmd)
	rootCmd.AddCommand(listStacksCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
