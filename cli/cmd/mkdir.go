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

// mkdirCmd represents the mkdir command
var mkdirCmd = &cobra.Command{
	Use:   "mkdir <directory_name...>",
	Short: "Creates new directories.",
	Long:  `Creates one or more new directories on the chatsh server.`,
	Args:  cobra.MinimumNArgs(1),
	// Add ValidArgsFunction for path completion
	ValidArgsFunction: PathCompletionFunc,
	Run: func(cmd *cobra.Command, args []string) {
		currentBaseDir := viper.GetString(currentDirectoryKey)
		if currentBaseDir == "" {
			currentBaseDir = viper.GetString(homeDirectoryKey)
		}

		for _, dirName := range args {
			// Construct the full path for the new directory
			// Similar to 'ls', we assume the server handles path resolution.
			// If dirName is absolute, it's used as is. If relative, it's relative to currentBaseDir.
			// This logic might need refinement based on server expectations.
			var targetPath string
			if filepath.IsAbs(dirName) {
				targetPath = dirName
			} else {
				targetPath = filepath.Join(currentBaseDir, dirName)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			req := &pb.CreateDirectoryRequest{
				Path:       targetPath,
				OwnerToken: ownerToken, // ownerToken is loaded in root.go
			}

			res, err := chatshClient.CreateDirectory(ctx, req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error calling CreateDirectory for %s: %v\n", targetPath, err)
				continue // Continue with the next directory if one fails
			}

			if res.Status.Ok {
				fmt.Printf("Directory created: %s\n", targetPath)
			} else {
				fmt.Fprintf(os.Stderr, "Failed to create directory %s: %s\n", targetPath, res.Status.Message)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(mkdirCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// mkdirCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// mkdirCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
