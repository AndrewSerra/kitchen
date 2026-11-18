package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func init() {
	rootCmd.AddCommand(registerCmd)

	registerCmd.Flags().StringP("email", "e", "", "Email address")
}

var registerCmd = &cobra.Command{
	Use:     "register",
	Short:   "Register a new account",
	Long:    "Register a new account",
	Run:     registerHandler,
	Example: `cli-acct-mgmt register --email test@example.com`,
}

func registerHandler(cmd *cobra.Command, args []string) {
	fmt.Print("Password: ")
	passwordIn, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("Error reading password:", err)
		return
	}
	fmt.Println()
	fmt.Print("Confirm Password: ")
	confirmPasswordIn, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("Error reading confirm password:", err)
		return
	}
	fmt.Println()
	if string(passwordIn) != string(confirmPasswordIn) {
		fmt.Println("Passwords do not match")
		return
	}

	fmt.Println("Registering...")

	email, _ := cmd.Flags().GetString("email")
	password := string(passwordIn)

	apiService, ok := cmd.Context().Value(CONTEXT_API_SERVICE_KEY).(*APIService)
	if !ok {
		fmt.Println("Error getting api service from context")
		return
	}

	localCredentials, ok := cmd.Context().Value(CONTEXT_LOCAL_CREDS_KEY).(LocalCredentials)
	if !ok {
		fmt.Println("Error getting local data from context")
		return
	}

	registerResponse, err := apiService.SendRegisterRequest(email, password)
	if err != nil {
		fmt.Println("Error registering:", err)
		return
	}

	localCredentials[Email(email)] = LocalAccountData{
		ID:           registerResponse.User.ID,
		AccessToken:  registerResponse.AccessToken,
		RefreshToken: registerResponse.RefreshToken,
		SavedAt:      time.Now(),
	}

	err = saveLocalAccountData(localCredentials)
	if err != nil {
		fmt.Println("Error saving local data:", err)
		return
	}

	fmt.Println("Registered successfully")
}
