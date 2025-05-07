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
	// viper is already used in root.go for ownerToken
)

// idCmd represents the id command
var idCmd = &cobra.Command{
	Use:   "id",
	Short: "Prints user configuration information.",
	Long:  `Prints user-specific configuration information, such as the display name, retrieved from the server.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		req := &pb.GetConfigRequest{
			OwnerToken: ownerToken, // ownerToken is loaded in root.go
		}

		res, err := chatshClient.GetConfig(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting user config (id): %v\n", err)
			return
		}

		// Standard `id` command prints uid, gid, groups.
		// Here we print the display name as a form of user identification within this app.
		fmt.Printf("DisplayName: %s\n", res.DisplayName)
		// If OwnerToken itself is considered an ID, it could be printed too,
		// but that might be a security concern depending on its nature.
		// fmt.Printf("OwnerToken: %s\n", ownerToken) // Example, if appropriate
	},
}

func init() {
	rootCmd.AddCommand(idCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// idCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// idCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
