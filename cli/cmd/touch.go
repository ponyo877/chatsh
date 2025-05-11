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

// touchCmd represents the touch command
var touchCmd = &cobra.Command{
	Use:   "touch <file_path...>",
	Short: "Creates new empty files (rooms) or updates timestamps (not implemented via API).",
	Long: `Attempts to create new empty files (rooms) on the chatsh server.
If the file already exists, the current API (CreateRoom) might return an error or do nothing.
Standard touch behavior of updating timestamps for existing files is not directly supported by CreateRoom.`,
	Args: cobra.MinimumNArgs(1),
	// Add ValidArgsFunction for path completion
	ValidArgsFunction: PathCompletionFunc,
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

			req := &pb.CreateRoomRequest{
				Path:       targetPath,
				OwnerToken: ownerToken, // ownerToken is loaded in root.go
			}

			res, err := chatshClient.CreateRoom(ctx, req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error calling CreateRoom for %s: %v\n", targetPath, err)
				continue
			}

			if res.Status.Ok {
				fmt.Printf("Room created or already exists: %s\n", targetPath)
				// Standard touch doesn't output on success.
			} else {
				// If server indicates "already exists" as not an error, this message might be misleading.
				// However, if it's a genuine failure to create, it's appropriate.
				fmt.Fprintf(os.Stderr, "Failed to touch (create room) %s: %s\n", targetPath, res.Status.Message)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(touchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// touchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// touchCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
