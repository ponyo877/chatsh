/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/mattn/go-shellwords"
	"github.com/oklog/ulid/v2"
	pb "github.com/ponyo877/chatsh/grpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	cfgFile           string
	ownerToken        string
	grpcServerAddress string
	isSecure          bool
	chatshClient      pb.ChatshServiceClient
	grpcConn          *grpc.ClientConn
	configCreated     bool
)

const (
	currentDirectoryKey  = "current_directory"
	homeDirectoryKey     = "home_directory"
	ownerTokenKey        = "owner_token"
	grpcServerAddressKey = "grpc_server_address"
	isSecureKey          = "is_secure"
	defaultServerAddress = "chatsh-app-1083612487436.asia-northeast1.run.app:443"
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
		if viper.ConfigFileUsed() == "" {
			fmt.Fprintln(os.Stderr, "Configuration file not found. create config: ~/.chatsh.yaml")
			if err_create := createDefaultConfig(); err_create != nil {
				return fmt.Errorf("failed to create default config: %w", err_create)
			}
			if err_read := viper.ReadInConfig(); err_read != nil {
				return fmt.Errorf("failed to read newly created config: %w", err_read)
			}
		}

		// Ensure global variables are fresh, especially if config was just created.
		ownerToken = viper.GetString(ownerTokenKey)
		grpcServerAddress = viper.GetString(grpcServerAddressKey)
		isSecure = viper.GetBool(isSecureKey)
		ownerToken = viper.GetString(ownerTokenKey)

		credential := insecure.NewCredentials()
		if isSecure {
			credential = credentials.NewTLS(&tls.Config{})
		}
		conn, err := grpc.NewClient(grpcServerAddress, grpc.WithTransportCredentials(credential))
		if err != nil {
			return fmt.Errorf("did not connect to gRPC server: %w", err)
		}
		grpcConn = conn
		chatshClient = pb.NewChatshServiceClient(conn)

		// If config was just created, call SetConfig on the server
		if configCreated {
			setConfigReq := &pb.SetConfigRequest{
				OwnerToken:  viper.GetString(ownerTokenKey),  // Use the newly generated token
				DisplayName: viper.GetString("display_name"), // "test"
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := chatshClient.SetConfig(ctx, setConfigReq)
			if err != nil {
				// Log error but don't necessarily block CLI usage if server SetConfig fails
				fmt.Fprintf(os.Stderr, "Warning: failed to set initial config on server: %v\n", err)
			} else {
				fmt.Println("Initial configuration set on server.")
			}
			configCreated = false // Reset flag
		}

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

	dir, base := filepath.Split(toComplete)
	dirToList := filepath.Join(viper.GetString(currentDirectoryKey), dir)
	if filepath.IsAbs(toComplete) {
		dirToList = dir
	}

	if debugFile != nil {
		fmt.Fprintf(debugFile, "Calculated dirToList: '%s', prefix: '%s'\n", dirToList, base)
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
		if strings.HasPrefix(entry.Name, base) {
			suggestion := filepath.Join(dir, entry.Name)
			if entry.Type == pb.NodeType_DIRECTORY {
				suggestion += "/"
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

func completer(d prompt.Document) []prompt.Suggest {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	text := d.TextBeforeCursor()
	fields := strings.Fields(text)
	if len(fields) < 2 && !strings.HasSuffix(text, " ") {
		return nil
	}
	toComplete := fields[len(fields)-1]
	if strings.HasSuffix(text, " ") {
		toComplete = ""
	}
	dir, base := filepath.Split(toComplete)
	dirToList := filepath.Join(viper.GetString(currentDirectoryKey), dir)
	if filepath.IsAbs(toComplete) {
		dirToList = dir
	}

	req := &pb.ListNodesRequest{
		Path: dirToList,
	}
	res, err := chatshClient.ListNodes(ctx, req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error listing nodes:", err)
		return []prompt.Suggest{}
	}

	s := []prompt.Suggest{}
	for _, entry := range res.Entries {
		if strings.HasPrefix(entry.Name, base) {
			suggestion := entry.Name
			if entry.Type == pb.NodeType_DIRECTORY {
				suggestion += "/"
			}
			s = append(s, prompt.Suggest{
				Text:        filepath.Join(dir, suggestion),
				Description: "",
			})
		}
	}
	return s
}

func executor(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if line == "exit" || line == "quit" {
		fmt.Println("exiting interactive mode")
		os.Exit(0)
	}

	args, err := shellwords.Parse(line)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing command:", err)
		return
	}
	if len(args) == 0 {
		return
	}
	originalPostRunE := rootCmd.PersistentPostRunE
	rootCmd.PersistentPostRunE = nil
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error executing command:", err)
		return
	}
	rootCmd.PersistentPostRunE = originalPostRunE
}

func Execute() {
	if len(os.Args) > 1 {
		if err := rootCmd.Execute(); err != nil {
			os.Exit(1)
		}
		if os.Args[1] == "completion" {
			return
		}
		return
	}
	initConfig()
	if rootCmd.PersistentPreRunE != nil {
		if err := rootCmd.PersistentPreRunE(rootCmd, []string{}); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to initialize gRPC client for interactive mode:", err)
			os.Exit(1)
		}
	}

	defer func() {
		if rootCmd.PersistentPostRunE != nil {
			if err := rootCmd.PersistentPostRunE(rootCmd, []string{}); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to close gRPC client after interactive mode:", err)
			}
		}
	}()

	fmt.Println("entering interactive mode, type 'exit' to quit")
	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("❯ "),
		prompt.OptionTitle("chatsh interactive mode"),
	)
	p.Run()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cli.yaml)")
	rootCmd.PersistentFlags().String("home-directory", "", "Home directory for the CLI")
	rootCmd.PersistentFlags().String("owner-token", "", "Owner token for authentication with the chatsh server")
	rootCmd.PersistentFlags().String("grpc-server", defaultServerAddress, "Address of the gRPC chatsh server (e.g., "+defaultServerAddress+")")
	rootCmd.PersistentFlags().Bool("is-secure", true, "Use secure gRPC connection (default: true)")

	viper.BindPFlag(homeDirectoryKey, rootCmd.PersistentFlags().Lookup("home-directory"))
	viper.BindPFlag(ownerTokenKey, rootCmd.PersistentFlags().Lookup("owner-token"))
	viper.BindPFlag(grpcServerAddressKey, rootCmd.PersistentFlags().Lookup("grpc-server"))
	viper.BindPFlag(isSecureKey, rootCmd.PersistentFlags().Lookup("is-secure"))
	viper.SetDefault(homeDirectoryKey, "/home/chatsh")
	viper.SetDefault(currentDirectoryKey, "/")
	viper.SetDefault(ownerTokenKey, "")
	viper.SetDefault(grpcServerAddressKey, defaultServerAddress)
	viper.SetDefault(isSecureKey, true)
	viper.SetDefault("display_name", "test")

}

func createDefaultConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home directory: %w", err)
	}
	configPath := filepath.Join(home, ".chatsh.yaml")

	// Generate ULID for owner_token
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	newOwnerToken := ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()

	viper.Set(ownerTokenKey, newOwnerToken)
	viper.Set("display_name", "test")
	viper.SetConfigFile(configPath)

	if err := viper.SafeWriteConfig(); err != nil {
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return fmt.Errorf("could not create config directory: %w", err)
		}
		if err := viper.WriteConfigAs(configPath); err != nil {
			return fmt.Errorf("could not write config file: %w", err)
		}
	}
	fmt.Printf("Default configuration file created at %s with owner_token: %s\n", configPath, newOwnerToken)
	configCreated = true // Set flag
	return nil
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		configDir := home

		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".chatsh")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config file (%s): %v\n", viper.ConfigFileUsed(), err)
		}
	}

	ownerToken = viper.GetString(ownerTokenKey)
	grpcServerAddress = viper.GetString(grpcServerAddressKey)
	isSecure = viper.GetBool(isSecureKey)
}
