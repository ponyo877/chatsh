/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/ponyo877/chatsh/grpc"

	"context"
	"time"

	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	cfgFile           string
	ownerToken        string
	grpcServerAddress string
	chatshClient      pb.ChatshServiceClient
	grpcConn          *grpc.ClientConn
)

const (
	currentDirectoryKey  = "current_directory"
	homeDirectoryKey     = "home_directory"
	ownerTokenKey        = "owner_token"
	grpcServerAddressKey = "grpc_server_address"
)

var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(grpcServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("did not connect to gRPC server: %w", err)
		}
		grpcConn = conn
		chatshClient = pb.NewChatshServiceClient(conn)
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if grpcConn != nil {
			return grpcConn.Close()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func PathCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completionFuncHelper(cmd, args, toComplete, true)
}

func DirectoryPathCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completionFuncHelper(cmd, args, toComplete, false)
}

func completionFuncHelper(cmd *cobra.Command, args []string, toComplete string, includeRoom bool) ([]string, cobra.ShellCompDirective) {
	debugFile, _ := os.OpenFile("/tmp/chatsh_completion_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if debugFile != nil {
		defer debugFile.Close()
		fmt.Fprintf(debugFile, "--- New Completion Request ---\n")
		fmt.Fprintf(debugFile, "Time: %s\n", time.Now().Format(time.RFC3339Nano))
		fmt.Fprintf(debugFile, "Command: %s\n", cmd.Use)
		fmt.Fprintf(debugFile, "Args: %v\n", args)
		fmt.Fprintf(debugFile, "ToComplete: '%s'\n", toComplete)
	}
	if chatshClient == nil {
		if debugFile != nil {
			fmt.Fprintf(debugFile, "Error: chatshClient is nil\n")
		}
		return nil, cobra.ShellCompDirectiveError
	}

	var dirToList, prefix string
	firstSlash := strings.Index(toComplete, "/")
	lastSlash := strings.LastIndex(toComplete, "/")
	current := viper.GetString(currentDirectoryKey)
	// absolute path
	if firstSlash == 0 {
		dirToList = "/"
		if lastSlash > 0 {
			dirToList = toComplete[:lastSlash]
		}

		prefix = toComplete[lastSlash+1:]
	} else { // relative path
		dirToList = current
		prefix = toComplete
		if lastSlash != -1 {
			dirToList = filepath.Join(dirToList, toComplete[:lastSlash])
			prefix = toComplete[lastSlash+1:]
		}
	}

	if debugFile != nil {
		fmt.Fprintf(debugFile, "Calculated dirToList: '%s', prefix: '%s'\n", dirToList, prefix)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.ListNodesRequest{
		Path: dirToList,
	}

	if debugFile != nil {
		fmt.Fprintf(debugFile, "ListNodes Request: Path='%s'\n", req.Path)
	}

	res, err := chatshClient.ListNodes(ctx, req)
	if err != nil {
		if debugFile != nil {
			fmt.Fprintf(debugFile, "ListNodes Error: %v\n", err)
		}
		return nil, cobra.ShellCompDirectiveError
	}

	var suggestions []string
	for _, entry := range res.Entries {
		if strings.HasPrefix(entry.Name, prefix) {
			suggestion := filepath.Join(toComplete[:lastSlash+1], entry.Name)
			if entry.Type == pb.NodeType_DIRECTORY {
				if !strings.HasSuffix(suggestion, "/") {
					suggestion += "/"
				}
			}
			if entry.Type == pb.NodeType_ROOM && !includeRoom {
				continue
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	if debugFile != nil {
		fmt.Fprintf(debugFile, "Suggestions: %v\n", suggestions)
		fmt.Fprintf(debugFile, "--- End Completion Request ---\n\n")
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}
func Execute() {
	if len(os.Args) > 1 {
		if err := rootCmd.Execute(); err != nil {
			os.Exit(1)
		}

		if len(os.Args) > 1 && os.Args[1] == "completion" {
			return
		}
		return
	}

	fmt.Println("entering interactive mode, type 'exit' to quit")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("❯❯❯ ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}

		args, err := shellwords.Parse(line)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing command:", err)
			continue
		}
		if len(args) == 0 {
			continue
		}

		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
		}
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cli.yaml)")
	rootCmd.PersistentFlags().String("home-directory", "", "Home directory for the CLI")
	rootCmd.PersistentFlags().String("owner-token", "", "Owner token for authentication with the chatsh server")
	rootCmd.PersistentFlags().String("grpc-server", "localhost:50051", "Address of the gRPC chatsh server (e.g., localhost:50051)")

	viper.BindPFlag(homeDirectoryKey, rootCmd.PersistentFlags().Lookup("home-directory"))
	viper.BindPFlag(ownerTokenKey, rootCmd.PersistentFlags().Lookup("owner-token"))
	viper.BindPFlag(grpcServerAddressKey, rootCmd.PersistentFlags().Lookup("grpc-server"))
	viper.SetDefault(homeDirectoryKey, "/home/chatsh")
	viper.SetDefault(currentDirectoryKey, "/home/chatsh")
	viper.SetDefault(ownerTokenKey, "default_token")
	viper.SetDefault(grpcServerAddressKey, "localhost:50051")

}
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cli")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Fprintln(os.Stderr, "Config file not found, using default values and environment variables.")
		} else {
			fmt.Fprintln(os.Stderr, "Error reading config file:", err)
		}
	}

	ownerToken = viper.GetString(ownerTokenKey)
	grpcServerAddress = viper.GetString(grpcServerAddressKey)
}
