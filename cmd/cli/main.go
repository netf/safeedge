package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "safeedge",
		Short: "SafeEdge CLI",
		Long:  "Command-line interface for SafeEdge fleet management platform",
	}

	// Placeholder commands - to be implemented
	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "version",
			Short: "Print version information",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("safeedge version 0.1.0")
			},
		},
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
