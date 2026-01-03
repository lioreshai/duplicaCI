package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	versionStr string
	commitStr  string
	dateStr    string

	// Global flags
	configFile string
	dryRun     bool
	verbose    bool
)

// SetVersionInfo sets version information from main
func SetVersionInfo(version, commit, date string) {
	versionStr = version
	commitStr = commit
	dateStr = date
}

var rootCmd = &cobra.Command{
	Use:   "duplicaci",
	Short: "Duplicacy + CI - Run Duplicacy backups from CI/CD pipelines",
	Long: `duplicaci is a Go wrapper for running Duplicacy backup operations
from CI/CD systems like GitHub Actions, Forgejo Actions, or cron.

It supports running Duplicacy commands locally, via SSH, or inside
Docker containers, with optional failure notifications via issue creation.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("duplicaci %s\n", versionStr)
		fmt.Printf("  commit: %s\n", commitStr)
		fmt.Printf("  built:  %s\n", dateStr)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "Print commands without executing")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(pruneCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
