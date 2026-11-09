package main

import (
	"fmt"
	"os"

	"github.com/AndrewSerra/kitchen/internal/utils"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

type ServerConfig struct {
	URL string `toml:"url"`
}

type Config struct {
	Server ServerConfig `toml:"server"`
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(serverUrlConfigSubCmd)

	serverUrlConfigSubCmd.Flags().StringP("url", "u", "", "Server URL")
	serverUrlConfigSubCmd.MarkFlagRequired("url")
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure the CLI",
	Long:  "Configure the CLI",
}

var serverUrlConfigSubCmd = &cobra.Command{
	Use:   "server-url",
	Short: "Configure the server URL",
	Long:  "Configure the server URL",
	Run:   serverUrlConfigHandler,
}

func serverUrlConfigHandler(cmd *cobra.Command, args []string) {
	config, ok := cmd.Context().Value("config").(Config)

	if !ok {
		fmt.Println("Error getting config from context")
		return
	}

	fmt.Println("Configuring server URL...")
	url, _ := cmd.Flags().GetString("url")

	validatedURL, err := utils.ValidateServerURL(url)
	if err != nil {
		fmt.Println("Error validating server URL:", err)
		return
	}

	config.Server.URL = validatedURL

	data, err := toml.Marshal(config)
	if err != nil {
		fmt.Println("Error marshalling config:", err)
		return
	}

	err = os.WriteFile(CONFIG_FILE_PATH, data, 0o644)
	if err != nil {
		fmt.Println("Error writing config:", err)
	}

	fmt.Println("Server URL configured successfully")
}
