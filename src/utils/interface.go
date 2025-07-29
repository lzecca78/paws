package utils

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	localconfig "github.com/lzecca78/paws/src/config"
	"github.com/lzecca78/paws/src/logger"
	"github.com/spf13/afero"
	"gopkg.in/ini.v1"
	"os/exec"
)

type Utils interface {
	SetProfile(profile string)
	GetProfiles() []string
	GetPromptProfiles(elements []string) (string, error)
	GetSSOStartURL() error
	SSOLogin() error
	PulumiSetup() error
	WriteFile(loc string) error
	NewShellCommand(name string, args ...string) IShellCommand
	GetCallerIdentity() (localconfig.AwsGetCallerIdentitySpec, error)
}

type Spec struct {
	Loader                   func(string) (*ini.File, error)
	Profile                  string
	Fs                       afero.Fs
	AwsGetCallerIdentitySpec localconfig.AwsGetCallerIdentitySpec
	SSOStartURL              string
}

func (u *Spec) SetProfile(profile string) {
	u.Profile = profile
}

func (u *Spec) GetProfiles() []string {
	return u.GetAWSProfiles()
}

func (u *Spec) GetPromptProfiles(elements []string) (string, error) {
	return CreatePrompt(elements)
}

func (u *Spec) GetSSOStartURL() error {
	ssu, err := u.GetSSOStart()
	u.SSOStartURL = ssu

	return err
}

func (u *Spec) SSOLogin() error {
	awsSpec, err := u.SSO(u.SSOStartURL)

	if err != nil {
		logger.Errorf("Failed to login to SSO for profile %s: %v", u.Profile, err)
		return err
	}

	u.AwsGetCallerIdentitySpec = awsSpec

	return err
}

func (u *Spec) PulumiSetup() error {
	return PulumiSetup(u.Fs, u.AwsGetCallerIdentitySpec)
}

func (u *Spec) WriteFile(loc string) error {
	return WriteFile(u.Fs, u.Profile, loc)
}

func (u *Spec) NewShellCommand(name string, args ...string) IShellCommand {
	cmd := exec.Command(name, args...)
	return &execShellCommand{Cmd: cmd}
}

func (u *Spec) GetCallerIdentity() (localconfig.AwsGetCallerIdentitySpec, error) {
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedConfigProfile(u.Profile),
	)
	if err != nil {
		return localconfig.AwsGetCallerIdentitySpec{}, err
	}

	client := sts.NewFromConfig(cfg)
	identity, err := client.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return localconfig.AwsGetCallerIdentitySpec{}, err
	}

	return localconfig.AwsGetCallerIdentitySpec{
		Account: aws.ToString(identity.Account),
		UserID:  aws.ToString(identity.UserId),
		ARN:     aws.ToString(identity.Arn),
	}, nil
}
