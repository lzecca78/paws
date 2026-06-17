package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is set at build time using -ldflags
// Example: go build -ldflags "-X github.com/lzecca78/paws/src/cmd.version=v1.0.0"
var version = "dev"

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
