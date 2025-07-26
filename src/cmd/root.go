package cmd

import (
	"fmt"
	"github.com/lzecca78/awsd/src/config"
	"github.com/lzecca78/awsd/src/logger"
	"github.com/lzecca78/awsd/src/utils"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "awsd",
	Short: "awsd - switch between AWS profiles.",
	Long:  "Allows for switching AWS profiles files.",
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

type PulumiConfig struct {
	PulumiProjects map[string]string `mapstructure:"pulumi_projects"`
}
type PulumiStack struct {
	Name string `json:"name"`
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
	awsProfileSpec, err := runProfileSwitcher()
	if err != nil {
		return err
	}
	// execute aws sso login
	awsSpec, err := utils.SSOLogin(awsProfileSpec.Profile, awsProfileSpec.SsoStartURL)
	if err != nil {
		logger.Errorf("Failed to run AWS SSO login: %v", err)
		return err
	}

	return utils.PulumiSetup(awsSpec)
}

func runProfileSwitcher() (awsProfileSpec AwsProfileSpec, error error) {
	profiles := utils.GetProfiles()
	fmt.Printf(utils.NoticeColor, "AWS Profile Switcher\n")
	profile, err := utils.CreatePrompt(profiles)
	if err != nil {
		return AwsProfileSpec{}, err
	}
	fmt.Printf(utils.PromptColor, "Choose a profile")
	fmt.Printf(utils.NoticeColor, "? ")
	fmt.Printf(utils.CyanColor, profile)
	fmt.Println()
	ssoStartURI, err := utils.GetSSOStartURL(profile)
	if err != nil {
		logger.Errorf("Failed to get SSO start URL for profile %s: %v", profile, err)
		return AwsProfileSpec{}, err
	}
	return AwsProfileSpec{Profile: profile, SsoStartURL: ssoStartURI}, utils.WriteFile(profile, utils.GetHomeDir())
}

func shouldRunDirectProfileSwitch() bool {
	invalidProfiles := []string{"l", "list", "completion", "help", "--help", "-h", "v", "version"}
	return len(os.Args) > 1 && !utils.Contains(invalidProfiles, os.Args[1])
}

func directProfileSwitch(desiredProfile string) error {
	profiles := utils.GetProfiles()
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
