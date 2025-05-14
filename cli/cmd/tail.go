/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var follow bool // Flag for -f option

// tailCmd represents the tail command
var tailCmd = &cobra.Command{
	Use:   "tail [-f] <path>",
	Short: "Displays the last part of a file (room messages).",
	Long: `Displays messages from the end of a specified path (room) on the chatsh server.
With -f, appends data as the file grows.`,
	Args: cobra.ExactArgs(1),
	// Add ValidArgsFunction for path completion
	ValidArgsFunction: PathCompletionFunc,
	Run: func(cmd *cobra.Command, args []string) {
		pathArg := args[0]

		currentBaseDir := viper.GetString(currentDirectoryKey)
		if currentBaseDir == "" {
			currentBaseDir = viper.GetString(homeDirectoryKey)
		}

		var targetPath string
		if filepath.IsAbs(pathArg) {
			targetPath = pathArg
		} else {
			targetPath = filepath.Join(currentBaseDir, pathArg)
		}

		// For tail, the context might need to be long-lived if -f is used.
		// However, individual gRPC stream calls might have their own timeouts or keep-alives.
		// For now, we'll use a cancellable context.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // Ensure cancellation on exit, e.g., Ctrl+C

		stream, err := chatshClient.StreamMessage(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling StreamMessage for %s: %v\n", targetPath, err)
			return
		}

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				// Stream ended (e.g. if not following, or server closes it)
				break
			}
			if err != nil {
				// Check if context was cancelled (e.g. Ctrl+C)
				if ctx.Err() == context.Canceled {
					// fmt.Fprintln(os.Stderr, "Stream cancelled by client.")
					break
				}
				fmt.Fprintf(os.Stderr, "Error receiving message stream for %s: %v\n", targetPath, err)
				break
			}

			fmt.Print(chunk.GetText()) // Assuming Line includes newline if it's a full line
		}
	},
}

func init() {
	rootCmd.AddCommand(tailCmd)
	tailCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow the content of the file")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tailCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tailCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
