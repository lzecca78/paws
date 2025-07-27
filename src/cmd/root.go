package cmd

import (
	"fmt"
	"github.com/lzecca78/paws/src/config"
	"github.com/lzecca78/paws/src/logger"
	"github.com/lzecca78/paws/src/utils"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
	"log"
	"os"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "paws",
	Short: "paws - switch between AWS profiles and Pulumi stacks.",
	Long:  "Allows for switching AWS profiles files and Pulumi stacks.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.InitConfig(cfgFile)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := primaInitialize(); err != nil {
			log.Fatal(err)
		}
	},
}

// Execute Entry point for the CLI tool
func Execute() {
	if shouldRunDirectProfileSwitch() {
		profile := os.Args[1]
		config.InitConfig(cfgFile)
		if err := directProfileSwitch(profile); err != nil {
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

func primaInitialize() error {
	awsProfileSpec := runProfileSwitcherWithPrompt(
		utils.CreatePrompt,
		utils.LoadINIFromPath,
		utils.LoadINIFromPath)
	// execute aws sso login
	awsSpec, err := utils.SSOLogin(awsProfileSpec.Profile, awsProfileSpec.SsoStartURL)
	if err != nil {
		logger.Errorf("Failed to run AWS SSO login: %v", err)
		return err
	}

	return utils.PulumiSetup(awsSpec)
}

func runProfileSwitcherWithPrompt(
	promptFn func([]string) (string, error),
	profileLoader func(string) (*ini.File, error),
	ssoLoader func(string) (*ini.File, error),
) AwsProfileSpec {
	profiles := utils.GetProfiles(profileLoader)

	fmt.Printf(utils.NoticeColor, "PAWS Profile Switcher\n")
	profile, err := promptFn(profiles)
	if err != nil {
		return AwsProfileSpec{}
	}

	fmt.Printf(utils.PromptColor, "Choose a profile")
	fmt.Printf(utils.NoticeColor, "? ")
	fmt.Printf(utils.CyanColor, profile)
	fmt.Println()

	ssoStartURI, err := utils.GetSSOStartURLWithLoader(profile, ssoLoader)
	if err != nil {
		logger.Errorf("Failed to get SSO start URL for profile %s: %v", profile, err)
		return AwsProfileSpec{}
	}

	return AwsProfileSpec{Profile: profile, SsoStartURL: ssoStartURI}
}

func shouldRunDirectProfileSwitch() bool {
	invalidProfiles := []string{"l", "list", "completion", "help", "--help", "-h", "v", "version"}
	return len(os.Args) > 1 && !utils.Contains(invalidProfiles, os.Args[1])
}

func directProfileSwitch(desiredProfile string) error {
	profiles := utils.GetProfiles(utils.LoadINIFromPath)
	if utils.Contains(profiles, desiredProfile) {
		printColoredMessage("Profile ", utils.PromptColor)
		printColoredMessage(desiredProfile, utils.CyanColor)
		printColoredMessage(" set.\n", utils.PromptColor)
		ssu, err := utils.GetSSOStartURL(desiredProfile)
		if err != nil {
			logger.Errorf("Failed to get SSO start URL for profile %s: %v", desiredProfile, err)
			return err
		}

		awsSpec, err := utils.SSOLogin(desiredProfile, ssu)
		if err != nil {
			logger.Errorf("Failed to run AWS SSO login: %v", err)
			return err
		}
		err = utils.PulumiSetup(awsSpec)
		if err != nil {
			logger.Errorf("Failed to setup Pulumi: %v", err)
			return err
		}
		return utils.WriteFile(desiredProfile, utils.GetHomeDir())
	}
	printColoredMessage("WARNING: Profile ", utils.NoticeColor)
	printColoredMessage(desiredProfile, utils.CyanColor)
	printColoredMessage(" does not exist.\n", utils.PromptColor)
	return nil
}

func printColoredMessage(msg, color string) {
	fmt.Printf(color, msg)
}
