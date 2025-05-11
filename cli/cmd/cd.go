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

// cdCmd represents the cd command
var cdCmd = &cobra.Command{
	Use:   "cd [directory]",
	Short: "Changes the current working directory.",
	Long: `Changes the current working directory managed by this CLI.
If no directory is specified, it changes to the user's home directory.
This command updates the directory path stored in the configuration file.`,
	Args: cobra.MaximumNArgs(1), // Allow 0 or 1 argument
	// Add ValidArgsFunction for path completion
	ValidArgsFunction: DirectoryPathCompletionFunc,
	Run: func(cmd *cobra.Command, args []string) {
		var targetDir string
		currentDir := viper.GetString(currentDirectoryKey)
		if currentDir == "" {
			// If no current directory is set, default to home directory
			currentDir = viper.GetString(homeDirectoryKey)
		}

		if len(args) == 0 {
			// No argument, cd to home directory
			targetDir = viper.GetString(homeDirectoryKey)
		} else {
			targetDir = args[0]
			// Handle relative paths
			if !filepath.IsAbs(targetDir) {
				targetDir = filepath.Join(currentDir, targetDir)
			}
		}

		// Clean the path (e.g., resolve "..")
		absTargetDir, err := filepath.Abs(targetDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error resolving path:", err)
			return
		}

		// Check if the target directory exists and is a directory
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		req := &pb.CheckDirectoryExistsRequest{
			Path: absTargetDir,
		}

		res, err := chatshClient.CheckDirectoryExists(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling ListNodes: %v\n", err)
			return
		}
		if !res.Exists {
			fmt.Fprintf(os.Stderr, "Directory does not exist: %s\n", absTargetDir)
			return
		}

		// Save the new current directory
		viper.Set(currentDirectoryKey, absTargetDir)
		// Attempt to write the config file.
		// If the config file doesn't exist, try to create it.
		// If it exists, overwrite it.
		if err := viper.WriteConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found, try to write a new one
				if err := viper.WriteConfigAs(viper.ConfigFileUsed()); err != nil {
					fmt.Fprintln(os.Stderr, "Error creating config file:", err)
				}
			} else {
				// Some other error occurred
				fmt.Fprintln(os.Stderr, "Error writing config file:", err)
			}
		}
		// fmt.Println("Current directory set to:", absTargetDir) // Optional: print confirmation
	},
}

func init() {
	rootCmd.AddCommand(cdCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cdCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cdCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
