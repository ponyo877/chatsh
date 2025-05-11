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

// grepCmd represents the grep command
var grepCmd = &cobra.Command{
	Use:   "grep <pattern> <path>",
	Short: "Searches for a pattern in a specified path (room).",
	Long:  `Searches for a given pattern within the messages of a specified path (room) on the chatsh server.`,
	Args:  cobra.ExactArgs(2),
	// Add ValidArgsFunction for path completion on the path argument
	ValidArgsFunction: PathCompletionFunc,
	Run: func(cmd *cobra.Command, args []string) {
		pattern := args[0]
		pathArg := args[1]

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

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		req := &pb.SearchMessageRequest{
			Pattern: pattern,
			Path:    targetPath,
			// SearchMessageRequest does not have OwnerToken in the proto definition.
		}

		res, err := chatshClient.SearchMessage(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling SearchMessage for pattern '%s' in %s: %v\n", pattern, targetPath, err)
			return
		}

		if len(res.Messages) == 0 {
			// Standard grep exits with 1 if no lines were selected, 0 otherwise.
			// For simplicity, we'll just print nothing.
			// To more closely match grep, we might os.Exit(1) here.
			return
		}

		for _, msg := range res.Messages {
			// Grep typically prints the matching line.
			// Here, msg.TextContent is the entire message that matched.
			fmt.Println(msg.TextContent)
		}
	},
}

func init() {
	rootCmd.AddCommand(grepCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// grepCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// grepCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
