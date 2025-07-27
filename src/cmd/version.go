package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var version string = "v0.1.3"

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "paws version command",
	Aliases: []string{"v"},
	Long:    "Returns the current version of paws",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "paws version:", version); err != nil {
			// You could either log or return a wrapped error here
			cmd.PrintErrln("Failed to print version:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
