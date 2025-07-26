package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
)

func InitConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Default config file location: $HOME/.myapp.yaml
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not determine home directory: %v\n", err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".pulumi_config")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
	}
}
