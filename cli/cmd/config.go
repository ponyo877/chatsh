/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	pb "github.com/ponyo877/chatsh/grpc"
	"github.com/spf13/cobra"
	// viper is already used in root.go for ownerToken, no need to import here unless for other specific configs
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config [new_display_name]",
	Short: "Gets or sets the display name.",
	Long: `Manages configuration for the chatsh client.
If called without arguments, it displays the current configuration (e.g., display name).
If called with an argument, it sets the display name to the provided value.`,
	Args: cobra.MaximumNArgs(1), // 0 or 1 argument
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if len(args) == 0 {
			// Get current config
			req := &pb.GetConfigRequest{
				OwnerToken: ownerToken, // ownerToken is loaded in root.go
			}
			res, err := chatshClient.GetConfig(ctx, req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting config: %v\n", err)
				return
			}
			fmt.Printf("Display Name: %s\n", res.DisplayName)
			// Potentially display other config values here if the API supports them
		} else {
			// Set new config (display name)
			newDisplayName := args[0]
			req := &pb.SetConfigRequest{
				OwnerToken:  ownerToken,
				DisplayName: newDisplayName,
			}
			res, err := chatshClient.SetConfig(ctx, req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error setting config: %v\n", err)
				return
			}
			if res.Status.Ok {
				fmt.Printf("Display name set to: %s\n", newDisplayName)
				// Also update local viper config if this CLI uses it for display name elsewhere
				// viper.Set("display_name", newDisplayName) // Example, if "display_name" is a key
				// viper.WriteConfig()
			} else {
				fmt.Fprintf(os.Stderr, "Failed to set display name: %s\n", res.Status.Message)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
