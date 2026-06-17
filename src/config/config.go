package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func InitConfig(cfgFile string) error {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Default config file location: $HOME/.myapp.yaml
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %w", err)
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".pulumi_config")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	fmt.Println("Using config file:", viper.ConfigFileUsed())
	return nil
}
