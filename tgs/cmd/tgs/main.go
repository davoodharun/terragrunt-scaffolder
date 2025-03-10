package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/davoodharun/tgs/internal/scaffold"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "tgs",
		Short: "TGS - Terraform Generator Scaffold",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	scaffoldCmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Generate infrastructure scaffold",
		RunE: func(cmd *cobra.Command, args []string) error {
			return scaffold.Generate()
		},
	}

	rootCmd.AddCommand(scaffoldCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
} 