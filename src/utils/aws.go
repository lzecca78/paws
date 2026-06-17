package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	localconfig "github.com/lzecca78/paws/src/config"
	"github.com/lzecca78/paws/src/logger"
	"github.com/spf13/afero"
	"gopkg.in/ini.v1"
)

const (
	profilePrefix  = "profile"
	defaultProfile = "default"
	// SSOTokenExpirationThreshold is the minimum time before expiration
	// that we consider an SSO token still valid (in minutes)
	SSOTokenExpirationThreshold = 15 * time.Minute
)

func LoadINIFromPath(path string) (*ini.File, error) {
	return ini.Load(path)
}

func (u *Spec) GetAWSProfiles() ([]string, error) {
	profileFileLocation := GetCurrentProfileFile()
	cfg, err := u.Loader(profileFileLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to load profiles: %w", err)
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
	return profiles, nil
}

func (u *Spec) GetSSOStart() (string, error) {
	profileFileLocation := GetCurrentProfileFile()
	cfg, err := u.Loader(profileFileLocation)
	if err != nil {
		return "", fmt.Errorf("failed to load profile file: %w", err)
	}

	sectionName := fmt.Sprintf("%s %s", profilePrefix, u.Profile)
	section, err := cfg.GetSection(sectionName)
	if err != nil {
		return "", fmt.Errorf("failed to get section for profile %s: %w", u.Profile, err)
	}

	ssoStartURL := section.Key("sso_start_url").String()
	if ssoStartURL == "" {
		return "", fmt.Errorf("SSO start URL not found for profile %s", u.Profile)
	}

	return ssoStartURL, nil
}

func (u *Spec) SSO(ssoStartURL string) (awsSpec localconfig.AwsGetCallerIdentitySpec, err error) {
	logger.Info("Running AWS SSO login...")
	ok, err := IsSSOTokenValid(u.Fs, ssoStartURL, SSOTokenExpirationThreshold)
	if err != nil {
		logger.Errorf("Failed to check SSO token validity: %v", err)
	}
	if !ok {
		logger.Warnf("SSO token is invalid or expired for profile %s, running login...", u.Profile)
		awsSpec, err = u.RunSSOLogin(true)
		if err != nil {
			logger.Errorf("Failed to run AWS SSO login: %v", err)
			return localconfig.AwsGetCallerIdentitySpec{}, err
		}
	} else {
		logger.Info("SSO token is valid, skipping login.")
		awsSpec, err = u.RunSSOLogin(false)
		if err != nil {
			logger.Errorf("Failed to run AWS SSO login: %v", err)
			return localconfig.AwsGetCallerIdentitySpec{}, err
		}
	}
	logger.Info("AWS SSO login completed.")
	return awsSpec, nil
}

func (u *Spec) RunSSOLogin(cli bool) (localconfig.AwsGetCallerIdentitySpec, error) {
	logger.Infof("Running SSO login for profile: %s", u.Profile)

	if cli {
		logger.Infof("Executing AWS SSO login command for profile: %s", u.Profile)
		command := u.ExecuteAwsSSOCommander()
		output, err := ExecuteSSOCommand(command)
		if err != nil {
			logger.Errorf("Failed to execute AWS SSO command: %v\nOutput: %s", err, output)
			return localconfig.AwsGetCallerIdentitySpec{}, fmt.Errorf("failed to execute AWS SSO command: %w", err)
		}
	}

	identity, err := u.GetCallerIdentity()
	if err != nil {
		logger.Errorf("Failed to get caller identity: %v", err)
		return localconfig.AwsGetCallerIdentitySpec{}, err
	}

	logger.Infof("Account: %s", identity.Account)
	logger.Infof("UserID: %s", identity.UserID)
	logger.Infof("ARN: %s", identity.ARN)

	err = os.Setenv("AWS_PROFILE", u.Profile)
	if err != nil {
		logger.Errorf("Failed to set AWS_PROFILE environment variable: %v", err)
		return localconfig.AwsGetCallerIdentitySpec{}, err
	}

	return identity, nil
}

func (u *Spec) ExecuteAwsSSOCommander() IShellCommand {
	return u.NewShellCommand("aws", "sso", "login", "--profile", u.Profile)
}

func ExecuteSSOCommand(command IShellCommand) ([]byte, error) {
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

func IsSSOTokenValid(fs afero.Fs, ssoStartURL string, threshold time.Duration) (bool, error) {
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

		if token.StartURL != ssoStartURL {
			continue
		}

		expiresAt, err := time.Parse(time.RFC3339, token.ExpiresAt)
		if err != nil {
			continue
		}

		// Convert to local time
		localExpiresAt := expiresAt.Local()

		if expiresAt.After(now.Add(threshold)) {
			logger.Infof("Found valid SSO token in file %s with expiration at %s", entry.Name(), localExpiresAt.Format(time.RFC1123))
			return true, nil
		}
	}

	logger.Infof("No valid SSO tokens found for start URL: %s", ssoStartURL)
	return false, nil
}
