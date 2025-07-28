package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	localconfig "github.com/lzecca78/paws/src/config"
	"github.com/lzecca78/paws/src/logger"
	"github.com/spf13/afero"
	"gopkg.in/ini.v1"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	profilePrefix  = "profile"
	defaultProfile = "default"
)

func LoadINIFromPath(path string) (*ini.File, error) {
	return ini.Load(path)
}

func GetProfiles(loadFunc func(string) (*ini.File, error)) []string {
	profileFileLocation := GetCurrentProfileFile()
	cfg, err := loadFunc(profileFileLocation)
	if err != nil {
		log.Fatalf("Failed to load profiles: %v", err)
	}

	sections := cfg.SectionStrings()
	profiles := make([]string, 0, len(sections)+1)
	for _, section := range sections {
		if strings.HasPrefix(section, profilePrefix) {
			trimmedProfile := strings.TrimPrefix(section, profilePrefix)
			trimmedProfile = strings.TrimSpace(trimmedProfile)
			profiles = append(profiles, trimmedProfile)
		}
	}
	profiles = AppendIfNotExists(profiles, defaultProfile)
	sort.Strings(profiles)
	return profiles
}

func GetSSOStartURLWithLoader(
	profile string,
	loadFunc func(string) (*ini.File, error),
) (string, error) {
	profileFileLocation := GetCurrentProfileFile()
	cfg, err := loadFunc(profileFileLocation)
	if err != nil {
		return "", fmt.Errorf("failed to load profile file: %w", err)
	}

	sectionName := fmt.Sprintf("%s %s", profilePrefix, profile)
	section, err := cfg.GetSection(sectionName)
	if err != nil {
		return "", fmt.Errorf("failed to get section for profile %s: %w", profile, err)
	}

	ssoStartURL := section.Key("sso_start_url").String()
	if ssoStartURL == "" {
		return "", fmt.Errorf("SSO start URL not found for profile %s", profile)
	}

	return ssoStartURL, nil
}

func SSOLogin(fs afero.Fs, profile, ssoStartUri string) (awsSpec localconfig.AwsGetCallerIdentitySpec, err error) {
	logger.Info("Running AWS SSO login...")
	ok, err := IsSSOTokenValid(fs, ssoStartUri, 15)
	if err != nil {
		logger.Errorf("Failed to check SSO token validity: %v", err)
	}
	if !ok {
		logger.Warnf("SSO token is invalid or expired for profile %s, running login...", profile)
		awsSpec, err = RunSSOLogin(profile, true)
		if err != nil {
			logger.Errorf("Failed to run AWS SSO login: %v", err)
			return localconfig.AwsGetCallerIdentitySpec{}, err
		}
	} else {
		logger.Info("SSO token is valid, skipping login.")
		awsSpec, err = RunSSOLogin(profile, false)
		if err != nil {
			logger.Errorf("Failed to run AWS SSO login: %v", err)
			return localconfig.AwsGetCallerIdentitySpec{}, err
		}
	}
	logger.Info("AWS SSO login completed.")
	return awsSpec, nil
}

func RunSSOLogin(profile string, cli bool) (localconfig.AwsGetCallerIdentitySpec, error) {
	logger.Infof("Running SSO login for profile: %s", profile)

	if cli {
		logger.Infof("Executing AWS SSO login command for profile: %s", profile)
		command := ExecuteAwsSSOCommander(profile)
		output, err := ExecuteAwsSSoCommand(command)
		if err != nil {
			logger.Errorf("Failed to execute AWS SSO command: %v\nOutput: %s", err, output)
			return localconfig.AwsGetCallerIdentitySpec{}, fmt.Errorf("failed to execute AWS SSO command: %w", err)
		}
	}

	//	cmd := exec.Command("aws", "sso", "login", "--profile", profile)
	//	cmd.Stdout = os.Stdout
	//	cmd.Stderr = os.Stderr
	//	cmd.Stdin = os.Stdin
	//	if err := cmd.Run(); err != nil {
	//		logger.Errorf("Failed ")
	//		return localconfig.AwsGetCallerIdentitySpec{}, fmt.Errorf("failed to run aws sso login: %w", err)
	//	}
	//}

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		fmt.Println("error:", err)
		return localconfig.AwsGetCallerIdentitySpec{}, err
	}

	client := sts.NewFromConfig(cfg)

	identity, err := client.GetCallerIdentity(
		context.TODO(),
		&sts.GetCallerIdentityInput{},
	)
	if err != nil {
		fmt.Println("error:", err)
		return localconfig.AwsGetCallerIdentitySpec{}, err
	}

	logger.Infof("Account: %s", aws.ToString(identity.Account))
	logger.Infof("UserID: %s", aws.ToString(identity.UserId))
	logger.Infof("ARN: %s", aws.ToString(identity.Arn))

	err = os.Setenv("AWS_PROFILE", profile)
	if err != nil {
		logger.Errorf("Failed to set AWS_PROFILE environment variable: %v", err)
		return localconfig.AwsGetCallerIdentitySpec{}, err
	}

	return localconfig.AwsGetCallerIdentitySpec{
		Account: aws.ToString(identity.Account),
		UserID:  aws.ToString(identity.UserId),
		ARN:     aws.ToString(identity.Arn),
	}, nil

}

func ExecuteAwsSSOCommander(profile string) IShellCommand {
	cmd := exec.Command("aws", "sso", "login", "--profile", profile)
	logger.Infof("Running AWS SSO command: %s", cmd.String())
	return &execShellCommand{Cmd: cmd}
}

func ExecuteAwsSSoCommand(command IShellCommand) ([]byte, error) {
	output, err := command.CombinedOutput()
	if err != nil {
		logger.Errorf("Failed to execute AWS SSO command: %v\nOutput: %s", err, output)
		return nil, fmt.Errorf("failed to execute AWS SSO command: %w", err)
	}
	logger.Infof("AWS SSO command output: %s", output)
	return output, nil
}

type SSOToken struct {
	StartURL  string `json:"startUrl"`
	ExpiresAt string `json:"expiresAt"`
}

func IsSSOTokenValid(fs afero.Fs, ssoStartUri string, threshold time.Duration) (bool, error) {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
	entries, err := afero.ReadDir(fs, cacheDir)
	if err != nil {
		return false, fmt.Errorf("failed to read SSO cache dir: %w", err)
	}

	now := time.Now().UTC()
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		fullPath := filepath.Join(cacheDir, entry.Name())

		data, err := afero.ReadFile(fs, fullPath)
		if err != nil {
			continue
		}

		var token SSOToken
		if err := json.Unmarshal(data, &token); err != nil {
			continue
		}

		if token.StartURL == "" || token.ExpiresAt == "" {
			continue
		}

		if token.StartURL != ssoStartUri {
			continue
		}

		expiresAt, err := time.Parse(time.RFC3339, token.ExpiresAt)
		if err != nil {
			continue
		}

		if expiresAt.After(now.Add(threshold)) {
			logger.Infof("Found valid SSO token in file %s with expiration at %s", entry.Name(), expiresAt)
			return true, nil
		}
	}

	logger.Infof("No valid SSO tokens found for start URL: %s", ssoStartUri)
	return false, nil
}
