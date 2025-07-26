package utils

import (
	"encoding/json"
	localconfig "github.com/lzecca78/awsd/src/config"
	"github.com/lzecca78/awsd/src/logger"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"path/filepath"
)

type PulumiConfig struct {
	PulumiProjects map[string]string                    `mapstructure:"pulumi_projects"`
	AwsSpec        localconfig.AwsGetCallerIdentitySpec `mapstructure:"aws_spec"`
}
type PulumiStack struct {
	Name string `json:"name"`
}

func PulumiSetup(awsSpec localconfig.AwsGetCallerIdentitySpec) error {
	var pConfig PulumiConfig
	pConfig.AwsSpec = awsSpec
	if ok, err := pConfig.checkYaml(); !ok || err != nil {
		return err
	}

	if err := pConfig.getConfig(); err != nil {
		return err
	}

	if err := pConfig.s3Login(); err != nil {
		logger.Errorf("Failed to login to S3 bucket: %v", err)
		return err
	}

	stacks, err := pConfig.Stacks()
	if err != nil {
		logger.Errorf("Failed to list Pulumi stacks: %v", err)
	}
	if len(stacks) > 0 {
		stack, err := CreatePrompt(stacks)
		if err != nil {
			logger.Errorf("Failed to create prompt for stacks: %v", err)
			return err
		}
		_, err = executePulumiCommand("stack", "select", stack)
		if err != nil {
			logger.Errorf("Failed to select Pulumi stack: %v", err)

		}

	}

	return nil

}

func (p *PulumiConfig) checkYaml() (ok bool, err error) {
	currentDir, err := os.Getwd()
	if err != nil {
		logger.Errorf("Failed to get current directory: %v", err)
		return false, err
	}
	pulumiFilePath := filepath.Join(currentDir, "Pulumi.yaml")
	if _, err := os.Stat(pulumiFilePath); err != nil {
		logger.Warnf("Pulumi.yaml file not found in current directory: %s", currentDir)
		return false, nil
	}

	return true, nil

}

func (p *PulumiConfig) getConfig() error {
	if err := viper.Unmarshal(p); err != nil {
		logger.Errorf("Failed to unmarshal Pulumi config: %v", err)
		return err
	}
	return nil
}

func (p *PulumiConfig) s3Login() error {
	bucketName := p.PulumiProjects[p.AwsSpec.Account]
	logger.Infof("Pulumi bucket name for account %s: %s", p.AwsSpec.Account, bucketName)
	if bucketName == "" {
		logger.Errorf("No s3 Bucket configured for the selected AWS profile %s", os.Getenv("AWS_PROFILE"))
	}

	_, err := executePulumiCommand("login", "s3://"+bucketName)
	return err
}

func (p *PulumiConfig) Stacks() ([]string, error) {
	output, err := executePulumiCommand("stack", "ls", "--json")

	var stacks []PulumiStack

	err = json.Unmarshal(output, &stacks)
	if err != nil {
		logger.Errorf("Failed to parse pulumi stack output: %v\nOutput: %s", err, output)
		return nil, err
	}
	currentStacks := make([]string, 0, len(stacks))

	for _, stack := range stacks {
		currentStacks = append(currentStacks, stack.Name)
	}

	return currentStacks, nil
}

func executePulumiCommand(args ...string) ([]byte, error) {
	cmd := exec.Command("pulumi", args...)
	logger.Infof("Running pulumi %s command: %s", args[0], cmd.String())
	currentDir, err := os.Getwd()
	if err != nil {
		logger.Errorf("Failed to get current directory: %v", err)
		return nil, err
	}
	cmd.Dir = currentDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf("Error running pulumi %s command: %v\nOutput: %s", args[0], err, output)
		return nil, err
	}

	return output, nil
}
