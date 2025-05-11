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

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm <path...>",
	Short: "Removes files or directories.",
	Long:  `Removes one or more files or directories on the chatsh server.`,
	Args:  cobra.MinimumNArgs(1),
	// Add ValidArgsFunction for path completion
	ValidArgsFunction: PathCompletionFunc,
	Run: func(cmd *cobra.Command, args []string) {
		currentBaseDir := viper.GetString(currentDirectoryKey)
		if currentBaseDir == "" {
			currentBaseDir = viper.GetString(homeDirectoryKey)
		}

		for _, itemPath := range args {
			var targetPath string
			if filepath.IsAbs(itemPath) {
				targetPath = itemPath
			} else {
				targetPath = filepath.Join(currentBaseDir, itemPath)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			req := &pb.DeletePathRequest{
				Path:       targetPath,
				OwnerToken: ownerToken, // ownerToken is loaded in root.go
			}

			res, err := chatshClient.DeletePath(ctx, req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error calling DeletePath for %s: %v\n", targetPath, err)
				continue
			}

			if res.Status.Ok {
				fmt.Printf("Removed: %s\n", targetPath)
			} else {
				fmt.Fprintf(os.Stderr, "Failed to remove %s: %s\n", targetPath, res.Status.Message)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// rmCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// rmCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
