package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(logoutCmd)

	logoutCmd.Flags().BoolP("all", "a", false, "Logout from all devices")
	logoutCmd.Flags().StringSliceP("email", "e", []string{}, "Email of the account to logout")
	logoutCmd.MarkFlagsMutuallyExclusive("all", "email")
	logoutCmd.MarkFlagsOneRequired("all", "email")
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from the account",
	Long:  "Logout from the account",
	Run:   logoutHandler,
}

func logoutHandler(cmd *cobra.Command, args []string) {
	localData, ok := cmd.Context().Value("localData").(LocalData)
	if !ok {
		fmt.Println("Error getting local data from context")
		return
	}

	apiService, ok := cmd.Context().Value("apiService").(*APIService)
	if !ok {
		fmt.Println("Error getting api service from context")
		return
	}

	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		fmt.Println("Error getting all flag:", err)
		return
	}
	email, err := cmd.Flags().GetStringSlice("email")
	if err != nil {
		fmt.Println("Error getting email flag:", err)
		return
	}

	emailsToLogout := []string{}

	if all {
		for k := range localData {
			emailsToLogout = append(emailsToLogout, k)
		}
	} else {
		for _, e := range email {
			emailsToLogout = append(emailsToLogout, e)
		}
	}

	var wg sync.WaitGroup
	var errorEmailsChan = make(chan string, len(emailsToLogout))
	var localDataMutex sync.Mutex

	for _, email := range emailsToLogout {

		if _, ok := localData[email]; !ok {
			errorEmailsChan <- email
			continue
		}

		wg.Go(func() {
			logoutResponse, err := apiService.SendLogoutRequest(localData[email].RefreshToken)
			if err != nil {
				fmt.Println("Error sending logout request:", err)
				errorEmailsChan <- email
				return
			}

			if !logoutResponse.Success {
				fmt.Println("Error sending logout request:", logoutResponse.Message)
				errorEmailsChan <- email
				return
			} else {
				localDataMutex.Lock()
				delete(localData, email)
				localDataMutex.Unlock()
			}

		})
	}

	wg.Wait()
	close(errorEmailsChan)
	saveLocalData(localData)

	var errorEmails []string

	for email := range errorEmailsChan {
		errorEmails = append(errorEmails, email)
	}

	if len(errorEmails) > 0 {
		errorEmailsStr := strings.Join(errorEmails, ", ")
		fmt.Println("Error logging out emails:", errorEmailsStr)
	} else {
		fmt.Println("Successfully logged out all accounts")
	}
}
