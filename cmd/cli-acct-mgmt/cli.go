package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

const CONFIG_FILE_PATH = "/Users/andrewserra/GithubProjects/kitchen/config.toml"

const CONTEXT_CONFIG_KEY = "config"
const CONTEXT_API_SERVICE_KEY = "apiService"
const CONTEXT_LOCAL_CREDS_KEY = "localCredentials"
const CONTEXT_LOCAL_ACCOUNT_DATA_CACHE_KEY = "localAccountDataCache"

func loadConfig() (Config, error) {
	inData, err := os.ReadFile(CONFIG_FILE_PATH)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = toml.Unmarshal(inData, &config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

var rootCmd = &cobra.Command{
	Use:   "cli-acct-mgmt",
	Short: "CLI for account management",
	Long:  "CLI for account management",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		config, err := loadConfig()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return err
		}

		localCredentials, err := loadLocalCredentials()
		if err != nil {
			fmt.Println("Error loading local data:", err)
			return err
		}

		apiService := NewAPIService(config.Server.URL)

		cmd.SetContext(context.WithValue(cmd.Context(), CONTEXT_CONFIG_KEY, config))
		cmd.SetContext(context.WithValue(cmd.Context(), CONTEXT_API_SERVICE_KEY, apiService))
		cmd.SetContext(context.WithValue(cmd.Context(), CONTEXT_LOCAL_CREDS_KEY, localCredentials))
		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing root command: %v", err)
		os.Exit(1)
	}
}
