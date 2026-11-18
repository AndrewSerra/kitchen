package main

import (
	"fmt"

	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/spf13/cobra"
)

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Stream management commands",
	Long:  "Stream management commands",
}

var createStreamSubCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new stream",
	Long:  "Create a new stream",
	Run:   addNewStream,
}

var getStreamSubCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a stream",
	Long:  "Get a stream",
	Run:   getStreamByName,
}

var getAllStreamsSubCmd = &cobra.Command{
	Use:   "list",
	Short: "List all streams",
	Long:  "List all streams",
	Run:   listStreams,
}

var deleteStreamSubCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a stream",
	Long:  "Delete a stream",
	Run:   deleteStream,
}

func init() {
	rootCmd.AddCommand(streamCmd)
	streamCmd.PersistentFlags().StringP("email", "e", "", "Email of the account")

	streamCmd.AddCommand(createStreamSubCmd)
	createStreamSubCmd.Flags().StringP("name", "n", "", "Name of the stream")
	createStreamSubCmd.MarkFlagRequired("name")

	streamCmd.AddCommand(getStreamSubCmd)
	getStreamSubCmd.Flags().StringP("name", "n", "", "Name of the stream")
	getStreamSubCmd.MarkFlagRequired("name")

	streamCmd.AddCommand(getAllStreamsSubCmd)

	streamCmd.AddCommand(deleteStreamSubCmd)
	deleteStreamSubCmd.Flags().StringP("name", "n", "", "Name of the stream")
	deleteStreamSubCmd.MarkFlagRequired("name")
}

func addNewStream(cmd *cobra.Command, args []string) {
	apiService, ok := cmd.Context().Value(CONTEXT_API_SERVICE_KEY).(*APIService)
	if !ok {
		fmt.Println("api service not found")
		return
	}
	localCreds, ok := cmd.Context().Value(CONTEXT_LOCAL_CREDS_KEY).(LocalCredentials)
	if !ok {
		fmt.Println("local credentials not found")
		return
	}

	email, err := cmd.Flags().GetString("email")
	if err != nil {
		fmt.Println("error getting email:", err)
		return
	}
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		fmt.Println("error getting name:", err)
		return
	}

	accessToken, err := getAccessTokenForEmail(localCreds, email)
	if err != nil {
		fmt.Println("error getting access token:", err)
		return
	}

	stream, err := apiService.SendCreateStreamRequest(name, accessToken)
	if err != nil {
		fmt.Println("error creating stream:", err)
		return
	}

	fmt.Println("stream created:", stream.Name)
}

func getStreamByName(cmd *cobra.Command, args []string) {
	apiService, ok := cmd.Context().Value(CONTEXT_API_SERVICE_KEY).(*APIService)
	if !ok {
		fmt.Println("api service not found")
		return
	}
	localCreds, ok := cmd.Context().Value(CONTEXT_LOCAL_CREDS_KEY).(LocalCredentials)
	if !ok {
		fmt.Println("local credentials not found")
		return
	}

	email, err := cmd.Flags().GetString("email")
	if err != nil {
		fmt.Println("error getting email:", err)
		return
	}

	name, err := cmd.Flags().GetString("name")
	if err != nil {
		fmt.Println("error getting name:", err)
		return
	}

	accessToken, err := getAccessTokenForEmail(localCreds, email)
	if err != nil {
		fmt.Println("error getting access token:", err)
		return
	}

	stream, err := apiService.SendGetStreamByNameRequest(name, accessToken)
	if err != nil {
		fmt.Println("error getting stream:", err)
		return
	}
	printStreams(models.Streams{stream})
}

func listStreams(cmd *cobra.Command, args []string) {
	apiService, ok := cmd.Context().Value(CONTEXT_API_SERVICE_KEY).(*APIService)
	if !ok {
		fmt.Println("api service not found")
		return
	}
	localCreds, ok := cmd.Context().Value(CONTEXT_LOCAL_CREDS_KEY).(LocalCredentials)
	if !ok {
		fmt.Println("local credentials not found")
		return
	}

	email, err := cmd.Flags().GetString("email")
	if err != nil {
		fmt.Println("error getting email:", err)
		return
	}

	accessToken, err := getAccessTokenForEmail(localCreds, email)
	if err != nil {
		fmt.Println("error getting access token:", err)
		return
	}

	streams, err := apiService.SendGetAllStreamsRequest(accessToken)
	if err != nil {
		fmt.Println("error getting streams:", err)
		return
	}

	fmt.Printf("Streams of email %s:\n", email)
	printStreams(*streams)
}

func deleteStream(cmd *cobra.Command, args []string) {
	apiService, ok := cmd.Context().Value(CONTEXT_API_SERVICE_KEY).(*APIService)
	if !ok {
		fmt.Println("api service not found")
		return
	}
	localCreds, ok := cmd.Context().Value(CONTEXT_LOCAL_CREDS_KEY).(LocalCredentials)
	if !ok {
		fmt.Println("local credentials not found")
		return
	}

	email, err := cmd.Flags().GetString("email")
	if err != nil {
		fmt.Println("error getting email:", err)
		return
	}

	name, err := cmd.Flags().GetString("name")
	if err != nil {
		fmt.Println("error getting name:", err)
		return
	}

	accessToken, err := getAccessTokenForEmail(localCreds, email)
	if err != nil {
		fmt.Println("error getting access token:", err)
		return
	}

	if err := apiService.SendDeleteStreamRequest(name, accessToken); err != nil {
		fmt.Println("error deleting stream:", err)
		return
	}

	fmt.Println("stream deleted")
}

func printStreams(streams models.Streams) {
	idWidth := 10
	nameWidth := 20

	fmt.Printf("%-*s %-*s\n", idWidth, "ID", nameWidth, "Name")
	fmt.Printf("%-*s %-*s\n", idWidth, "----", nameWidth, "----")
	for _, stream := range streams {
		fmt.Printf("%-*d %-*s\n", idWidth, stream.ID, nameWidth, stream.Name)
	}
}
