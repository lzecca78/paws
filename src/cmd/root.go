package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/lzecca78/paws/src/config"
	"github.com/lzecca78/paws/src/logger"
	"github.com/lzecca78/paws/src/utils"
	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "paws",
	Short: "paws - switch between AWS profiles and Pulumi stacks.",
	Long:  "Allows for switching AWS profiles files and Pulumi stacks.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for commands that don't need it
		skipConfigCommands := []string{"version", "help", "completion", "list"}
		for _, skip := range skipConfigCommands {
			if cmd.Name() == skip {
				return nil
			}
		}
		return config.InitConfig(cfgFile)
	},
	Run: func(cmd *cobra.Command, args []string) {
		utilsSpec := utils.NewSpec("")
		if err := primaInitialize(utilsSpec); err != nil {
			log.Fatal(err)
		}
	},
}

// Execute Entry point for the CLI tool
func Execute() {
	if shouldRunDirectProfileSwitch() {
		profile := os.Args[1]
		if err := config.InitConfig(cfgFile); err != nil {
			log.Fatal(err)
		}
		utilsSpec := utils.NewSpec(profile)
		if err := directProfileSwitch(profile, utilsSpec); err != nil {
			log.Fatal(err)
		}
		return
	}
	runRootCmd()
}

type AwsProfileSpec struct {
	Profile     string
	SsoStartURL string
}

func runRootCmd() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pulumi_config.yaml)")
}

func primaInitialize(helper utils.Utils) error {
	_, err := runProfileSwitcherWithPrompt(helper)
	if err != nil {
		return err
	}
	// execute aws sso login
	err = helper.SSOLogin()
	if err != nil {
		logger.Errorf("Failed to run AWS SSO login: %v", err)
		return err
	}

	return helper.PulumiSetup()
}

func runProfileSwitcherWithPrompt(
	helper utils.Utils,
) (AwsProfileSpec, error) {
	profiles, err := helper.GetProfiles()
	if err != nil {
		return AwsProfileSpec{}, err
	}

	fmt.Printf(utils.NoticeColor, "PAWS Profile Switcher\n")
	profile, err := helper.GetPromptProfiles(profiles)
	if err != nil {
		return AwsProfileSpec{}, err
	}

	helper.SetProfile(profile)

	fmt.Printf(utils.PromptColor, "Choose a profile")
	fmt.Printf(utils.NoticeColor, "? ")
	fmt.Printf(utils.CyanColor, profile)
	fmt.Println()

	err = helper.GetSSOStartURL()
	if err != nil {
		logger.Errorf("Failed to get SSO start URL for profile %s: %v", profile, err)
		return AwsProfileSpec{}, err
	}

	homeDir, err := utils.GetHomeDir()
	if err != nil {
		return AwsProfileSpec{}, err
	}

	return AwsProfileSpec{Profile: profile, SsoStartURL: helper.GetSSOUrl()}, helper.WriteFile(homeDir)
}

func shouldRunDirectProfileSwitch() bool {
	invalidProfiles := []string{"l", "list", "completion", "help", "--help", "-h", "v", "version"}
	return len(os.Args) > 1 && !utils.Contains(invalidProfiles, os.Args[1])
}

func directProfileSwitch(
	desiredProfile string,
	helper utils.Utils,
) error {
	profiles, err := helper.GetProfiles()
	if err != nil {
		return err
	}
	if utils.Contains(profiles, desiredProfile) {
		printColoredMessage("Profile ", utils.PromptColor)
		printColoredMessage(desiredProfile, utils.CyanColor)
		printColoredMessage(" set.\n", utils.PromptColor)
		helper.SetProfile(desiredProfile)
		err := helper.GetSSOStartURL()
		if err != nil {
			logger.Errorf("Failed to get SSO start URL for profile %s: %v", desiredProfile, err)
			return err
		}
		err = helper.SSOLogin()

		if err != nil {
			logger.Errorf("Failed to run AWS SSO login: %v", err)
			return err
		}
		err = helper.PulumiSetup()
		if err != nil {
			logger.Errorf("Failed to setup Pulumi: %v", err)
			return err
		}

		homeDir, err := utils.GetHomeDir()
		if err != nil {
			return err
		}

		return helper.WriteFile(homeDir)
	}
	printColoredMessage("WARNING: Profile ", utils.NoticeColor)
	printColoredMessage(desiredProfile, utils.CyanColor)
	printColoredMessage(" does not exist.\n", utils.PromptColor)
	return nil
}

func printColoredMessage(msg, color string) {
	fmt.Printf(color, msg)
}
