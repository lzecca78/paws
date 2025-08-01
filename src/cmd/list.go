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

//	func runProfileLister() error {
//		profiles := utils.GetProfiles()
//		for _, p := range profiles {
//			fmt.Println(p)
//		}
//		return nil
//	}
func runProfileListerToWriter(w io.Writer) error {
	spec := utils.Spec{
		Loader: utils.LoadINIFromPath,
	}
	profiles := spec.GetAWSProfiles()
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
