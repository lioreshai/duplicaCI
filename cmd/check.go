package cmd

import (
	"fmt"
	"os"

	"github.com/lioreshai/duplicaci/internal/executor"
	"github.com/lioreshai/duplicaci/internal/stats"
	"github.com/spf13/cobra"
)

var (
	updateStats bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check backup integrity",
	Long:  `Run Duplicacy check command to verify backup integrity.`,
	RunE:  runCheckCmd,
}

func init() {
	checkCmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository ID")
	checkCmd.Flags().StringVarP(&repoPath, "repo-path", "p", "", "Path to repository (cd here before running duplicacy)")
	checkCmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Duplicacy Web GUI cache directory (e.g., /cache/localhost/0)")
	checkCmd.Flags().StringSliceVarP(&storages, "storage", "s", []string{}, "Storage backend(s) to check")
	checkCmd.Flags().StringVar(&dockerContainer, "docker-container", "", "Run inside Docker container")
	checkCmd.Flags().StringVar(&sshHost, "ssh-host", "", "SSH to host before running (user@host)")
	checkCmd.Flags().StringVar(&sshPassword, "ssh-password", "", "SSH password (or SSH_PASSWORD env)")
	checkCmd.Flags().StringVar(&storagePassword, "storage-password", "", "Duplicacy storage encryption password (or DUPLICACY_PASSWORD env)")
	checkCmd.Flags().StringVar(&gcdToken, "gcd-token", "", "Google Drive token file path (for gcd:// storages)")
	checkCmd.Flags().BoolVar(&updateStats, "update-stats", false, "Update Duplicacy Web UI stats after check")
}

func runCheckCmd(cmd *cobra.Command, args []string) error {
	if len(storages) == 0 {
		return fmt.Errorf("at least one --storage is required")
	}

	if sshPassword == "" {
		sshPassword = os.Getenv("SSH_PASSWORD")
	}

	if storagePassword == "" {
		storagePassword = os.Getenv("DUPLICACY_PASSWORD")
	}

	exec := executor.New(executor.Options{
		DryRun:          dryRun,
		Verbose:         verbose,
		DockerContainer: dockerContainer,
		SSHHost:         sshHost,
		SSHPassword:     sshPassword,
		RepoPath:        repoPath,
		CacheDir:        cacheDir,
		StoragePassword: storagePassword,
		GCDToken:        gcdToken,
	})

	// Create stats writer if updating stats
	var statsWriter *stats.Writer
	if updateStats && dockerContainer != "" {
		statsWriter = stats.NewWriter(sshHost, sshPassword, dockerContainer)
		statsWriter.DryRun = dryRun
		statsWriter.Verbose = verbose
	}

	var hasErrors bool

	for _, storage := range storages {
		fmt.Printf("==> Checking storage '%s'\n", storage)

		// Run check with -tabular to get stats output
		output, err := exec.RunDuplicacyCaptureWithStorage(storage, "check", "-tabular", "-storage", storage)

		// Print the output (since we captured it)
		if output != "" {
			fmt.Print(output)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: check on %s failed: %v\n", storage, err)
			hasErrors = true
			continue
		}

		fmt.Printf("    Check on '%s' completed successfully\n", storage)

		// Update stats if enabled
		if statsWriter != nil && output != "" {
			dayStats, parseErr := stats.ParseCheckOutput(output)
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "    WARNING: failed to parse check output for stats: %v\n", parseErr)
			} else {
				// Print parsed stats summary
				fmt.Printf("\n    Storage Stats Summary:\n")
				fmt.Printf("      Total size: %s\n", stats.FormatBytes(dayStats.TotalSize))
				fmt.Printf("      Total chunks: %d\n", dayStats.TotalChunks)
				fmt.Printf("      Repositories: %d\n", len(dayStats.Repositories))
				for repoName, repoStats := range dayStats.Repositories {
					fmt.Printf("        - %s: %d revisions, %s\n", repoName, repoStats.Revisions, stats.FormatBytes(repoStats.TotalSize))
				}

				if writeErr := statsWriter.UpdateStorageStats(storage, dayStats); writeErr != nil {
					fmt.Fprintf(os.Stderr, "    WARNING: failed to update stats: %v\n", writeErr)
				} else {
					fmt.Printf("    Updated Duplicacy Web UI stats for '%s'\n", storage)
				}
			}
		}
	}

	if hasErrors {
		return fmt.Errorf("check completed with errors")
	}

	fmt.Println("==> All checks completed successfully")
	return nil
}
