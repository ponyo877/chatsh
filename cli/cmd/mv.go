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

// mvCmd represents the mv command
var mvCmd = &cobra.Command{
	Use:   "mv <source> <destination>",
	Short: "Moves or renames a file or directory.",
	Long:  `Moves (renames) a source file or directory to a destination on the chatsh server.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sourceArg := args[0]
		destinationArg := args[1]

		currentBaseDir := viper.GetString(currentDirectoryKey)
		if currentBaseDir == "" {
			currentBaseDir = viper.GetString(homeDirectoryKey)
		}

		var sourcePath string
		if filepath.IsAbs(sourceArg) {
			sourcePath = sourceArg
		} else {
			sourcePath = filepath.Join(currentBaseDir, sourceArg)
		}

		var destinationPath string
		if filepath.IsAbs(destinationArg) {
			destinationPath = destinationArg
		} else {
			destinationPath = filepath.Join(currentBaseDir, destinationArg)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		req := &pb.MovePathRequest{
			SourcePath:      sourcePath,
			DestinationPath: destinationPath,
			OwnerToken:      ownerToken, // ownerToken is loaded in root.go
		}

		res, err := chatshClient.MovePath(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling MovePath from %s to %s: %v\n", sourcePath, destinationPath, err)
			return
		}

		if res.Status.Ok {
			fmt.Printf("Moved %s to %s\n", sourcePath, destinationPath)
		} else {
			fmt.Fprintf(os.Stderr, "Failed to move %s to %s: %s\n", sourcePath, destinationPath, res.Status.Message)
		}
	},
}

func init() {
	rootCmd.AddCommand(mvCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// mvCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// mvCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
