/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pb "github.com/ponyo877/chatsh/grpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// echoCmd represents the echo command
var echoCmd = &cobra.Command{
	Use:   "echo <text_to_write> <destination_path>",
	Short: "Writes text to a destination path (room).",
	Long:  `Writes the given text content to the specified destination path (room) on the chatsh server.`,
	Args:  cobra.ExactArgs(2),
	// Add ValidArgsFunction for path completion on the destination_path argument
	ValidArgsFunction: PathCompletionFunc,
	Run: func(cmd *cobra.Command, args []string) {
		textContent := args[0]
		destinationPathArg := args[1]

		currentBaseDir := viper.GetString(currentDirectoryKey)
		if currentBaseDir == "" {
			currentBaseDir = viper.GetString(homeDirectoryKey)
		}

		var targetPath string
		if filepath.IsAbs(destinationPathArg) {
			targetPath = destinationPathArg
		} else {
			targetPath = filepath.Join(currentBaseDir, destinationPathArg)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		req := &pb.WriteMessageRequest{
			TextContent:     textContent,
			DestinationPath: targetPath,
			OwnerToken:      ownerToken, // ownerToken is loaded in root.go
		}

		res, err := chatshClient.WriteMessage(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling WriteMessage to %s: %v\n", targetPath, err)
			return
		}

		if res.Status.Ok {
			// Standard echo doesn't print anything other than the echoed string.
			// Here, we are writing to a server, so a confirmation might be useful.
			// However, to mimic `echo > file` behavior, we might not print anything on success.
			// For now, let's print a confirmation.
			fmt.Printf("Text written to %s\n", targetPath)
		} else {
			fmt.Fprintf(os.Stderr, "Failed to write text to %s: %s\n", targetPath, res.Status.Message)
		}
	},
}

func init() {
	rootCmd.AddCommand(echoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// echoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// echoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
