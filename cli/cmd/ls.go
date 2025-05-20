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
		current := viper.GetString(currentDirectoryKey)
		var targetPath string
		if len(args) == 0 {
			targetPath = current
		} else {
			sourceArg := args[0]
			if filepath.IsAbs(sourceArg) {
				targetPath = sourceArg
			} else {
				targetPath = filepath.Join(current, sourceArg)
			}
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

			formattedTime := "           "
			if entry.Modified != nil {
				t := entry.Modified.AsTime()
				monthStr := t.Format("1")
				dayStr := fmt.Sprintf("%2d", t.Day())
				timeStr := t.Format("15:04")
				formattedTime = fmt.Sprintf("%s %s %s", monthStr, dayStr, timeStr)
			}
			fmt.Printf("%-4s  %s %s\n", nodeType, formattedTime, entry.Name)
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
