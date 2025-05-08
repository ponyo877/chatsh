/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	// Adjust this import path if your module name is different.
	pb "github.com/ponyo877/chatsh/grpc" // Import the generated gRPC package

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
	homeDirectoryKey     = "home_directory"
	ownerTokenKey        = "owner_token"
	grpcServerAddressKey = "grpc_server_address"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize gRPC client
		// Ensure grpcServerAddress and ownerToken are loaded by initConfig before this runs
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
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// one‑shot
	if len(os.Args) > 1 {
		if err := rootCmd.Execute(); err != nil {
			os.Exit(1)
			return
		}
	}

	// REPL
	fmt.Println("entering interactive mode, type 'exit' to quit")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("❯❯❯ ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}
		args, _ := shellwords.Parse(line)
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			os.Exit(1)
			return
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
	viper.SetDefault(ownerTokenKey, "default_token") // Consider if a default token is appropriate or if it should always be set
	viper.SetDefault(grpcServerAddressKey, "localhost:50051")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cli") // This will look for .cli.yaml
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if it's optional
			fmt.Fprintln(os.Stderr, "Config file not found, using default values and environment variables.")
		} else {
			// Config file was found but another error was produced
			fmt.Fprintln(os.Stderr, "Error reading config file:", err)
		}
	}

	// Load values after all potential sources (defaults, flags, env, config file)
	ownerToken = viper.GetString(ownerTokenKey)
	grpcServerAddress = viper.GetString(grpcServerAddressKey)

	// For debugging, you can print the loaded values:
	// fmt.Fprintln(os.Stderr, "Loaded owner token:", ownerToken)
	// fmt.Fprintln(os.Stderr, "Loaded gRPC server address:", grpcServerAddress)
}
