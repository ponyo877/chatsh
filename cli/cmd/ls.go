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
	"github.com/spf13/viper"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls [path]",
	Short: "Lists directory contents or room information.",
	Long: `Lists the contents of a specified directory or information about a room
on the chatsh server. If no path is provided, it lists the contents
of the current directory managed by this CLI.`,
	Args: cobra.MaximumNArgs(1),
	// Add ValidArgsFunction for path completion
	ValidArgsFunction: PathCompletionFunc,
	Run: func(cmd *cobra.Command, args []string) {
		var targetPath string
		if len(args) == 0 {
			targetPath = viper.GetString(currentDirectoryKey)
			if targetPath == "" {
				targetPath = viper.GetString(homeDirectoryKey)
			}
		} else {
			targetPath = args[0]
			// Note: For gRPC calls, we might not need to resolve to absolute path on client side
			// if the server handles relative paths based on its own context or a user session.
			// However, if the server expects absolute paths, or paths relative to a user-specific root,
			// this might need adjustment similar to the 'cd' command.
			// For now, we'll pass the path as is, assuming the server handles it.
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		req := &pb.ListNodesRequest{
			Path: targetPath,
		}

		res, err := chatshClient.ListNodes(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling ListNodes: %v\n", err)
			return
		}

		if len(res.Entries) == 0 {
			fmt.Println("Directory is empty or path does not exist.")
			return
		}

		for _, entry := range res.Entries {
			nodeType := ""
			switch entry.Type {
			case pb.NodeType_ROOM:
				nodeType = "ROOM"
			case pb.NodeType_DIRECTORY:
				nodeType = "DIR "
			default:
				nodeType = "UNKN"
			}
			// Basic output, can be enhanced with more details like owner, modified time
			fmt.Printf("%s\t%s\n", nodeType, entry.Name)
		}
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// lsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// lsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
