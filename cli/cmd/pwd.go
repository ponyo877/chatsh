/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const currentDirectoryKey = "current_directory"

// pwdCmd represents the pwd command
var pwdCmd = &cobra.Command{
	Use:   "pwd",
	Short: "Prints the current working directory.",
	Long: `Prints the current working directory managed by this CLI.
This may be different from the OS's current working directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		currentDir := viper.GetString(currentDirectoryKey)
		if currentDir == "" {
			currentDir = viper.GetString(homeDirectoryKey)
			// Optionally, save the home directory as the default if it's the first run
			// viper.Set(currentDirectoryKey, currentDir)
			// if err := viper.SafeWriteConfig(); err != nil {
			//  if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			//    // Config file not found; ignore error if we want to auto-create it later with cd
			//  } else {
			//    fmt.Fprintln(os.Stderr, "Error saving current directory:", err)
			//  }
			// }
		}
		fmt.Println(currentDir)
	},
}

func init() {
	rootCmd.AddCommand(pwdCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pwdCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// pwdCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
