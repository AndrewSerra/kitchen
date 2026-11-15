package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:  "list",
	Args: cobra.MinimumNArgs(1),
	ValidArgs: []string{
		"accounts",
	},
	Run: listHandler,
}

type ListFunc func()

func listHandler(cmd *cobra.Command, args []string) {
	localData, ok := cmd.Context().Value("localData").(LocalData)
	if !ok {
		fmt.Println("Error getting local data from context")
		return
	}

	funcs := []ListFunc{}
	invalidArgs := []string{}

	for _, arg := range args {
		switch arg {
		case "accounts":
			funcs = append(funcs, func() { printSavedAccounts(localData) })
		default:
			invalidArgs = append(invalidArgs, arg)
			return
		}
	}

	if len(invalidArgs) > 0 {
		fmt.Printf("Invalid arguments: %s\n", strings.Join(invalidArgs, ", "))
		return
	}

	for _, fn := range funcs {
		fn()
	}
}
