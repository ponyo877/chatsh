/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	pb "github.com/ponyo877/chatsh/grpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// catCmd represents the cat command
var catCmd = &cobra.Command{
	Use:   "cat <file_path...>",
	Short: "Displays the content of files (rooms).",
	Long:  `Displays the content of one or more files (interpreted as rooms) on the chatsh server.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		currentBaseDir := viper.GetString(currentDirectoryKey)
		if currentBaseDir == "" {
			currentBaseDir = viper.GetString(homeDirectoryKey)
		}

		for _, filePathArg := range args {
			var targetPath string
			if filepath.IsAbs(filePathArg) {
				targetPath = filePathArg
			} else {
				targetPath = filepath.Join(currentBaseDir, filePathArg)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			// Using ListMessages to simulate cat.
			// This assumes a "room" path corresponds to a file and its messages are its content.
			req := &pb.ListMessagesRequest{
				Path: targetPath,
				// ListMessagesRequest does not have OwnerToken in the proto definition.
				// If authentication is needed for this, the proto would need to be updated.
			}

			res, err := chatshClient.ListMessages(ctx, req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error calling ListMessages for %s: %v\n", targetPath, err)
				continue
			}

			if len(res.Messages) == 0 {
				// fmt.Fprintf(os.Stderr, "File %s is empty or does not exist.\n", targetPath)
				// To behave like `cat`, print nothing for an empty file.
				// If it doesn't exist, the error above would have caught it if the server returns an error.
				// If the server returns an empty list for non-existent files, this is the behavior.
				continue
			}

			// Concatenate all message contents.
			// This might not be the most efficient way for very large files/many messages.
			// StreamMessage might be better if the server supports streaming a whole "file".
			var contentBuilder strings.Builder
			for _, msg := range res.Messages {
				contentBuilder.WriteString(msg.TextContent)
				// Assuming messages don't inherently have newlines and `cat` should preserve them if they do.
				// If each message is a line, then add a newline:
				// contentBuilder.WriteString("\n")
			}
			fmt.Print(contentBuilder.String())
		}
	},
}

func init() {
	rootCmd.AddCommand(catCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// catCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// catCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
