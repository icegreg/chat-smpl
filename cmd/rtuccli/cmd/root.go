package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var serverURL string

var rootCmd = &cobra.Command{
	Use:   "rtuccli",
	Short: "RTUC CLI - chat-smpl management tool",
	Long: `rtuccli is a command-line interface for managing and monitoring
chat-smpl voice conferences, participants, and microservices.`,
	Version: "1.0.0",
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", getEnv("ADMIN_SERVICE_URL", "http://localhost:8086"), "Admin service URL")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
