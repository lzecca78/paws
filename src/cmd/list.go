package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/lzecca78/paws/src/utils"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List AWS profiles command.",
	Aliases: []string{"l"},
	Long:    "This lists all your AWS profiles.",
	Run: func(cmd *cobra.Command, args []string) {
		err := runProfileLister()
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runProfileListerToWriter(w io.Writer) error {
	spec := utils.NewSpec("") // profile not needed for listing
	profiles, err := spec.GetAWSProfiles()
	if err != nil {
		return err
	}
	for _, p := range profiles {
		_, err := fmt.Fprintln(w, p)
		if err != nil {
			return err
		}
	}
	return nil
}

var runProfileLister = func() error {
	return runProfileListerToWriter(os.Stdout)
}
