package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/lioreshai/duplicaci/internal/executor"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune old backup revisions",
	Long:  `Run Duplicacy prune command to remove old backup revisions according to retention policy.`,
	RunE:  runPruneCmd,
}

func init() {
	pruneCmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository ID")
	pruneCmd.Flags().StringVarP(&repoPath, "repo-path", "p", "", "Path to repository (cd here before running duplicacy)")
	pruneCmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Duplicacy Web GUI cache directory (e.g., /cache/localhost/0)")
	pruneCmd.Flags().StringSliceVarP(&storages, "storage", "s", []string{}, "Storage backend(s) to prune")
	pruneCmd.Flags().StringVar(&pruneOptions, "prune-options", "-keep 0:180 -keep 7:14 -keep 1:1 -a", "Prune retention options")
	pruneCmd.Flags().StringVar(&dockerContainer, "docker-container", "", "Run inside Docker container")
	pruneCmd.Flags().StringVar(&sshHost, "ssh-host", "", "SSH to host before running (user@host)")
	pruneCmd.Flags().StringVar(&sshPassword, "ssh-password", "", "SSH password (or SSH_PASSWORD env)")
	pruneCmd.Flags().StringVar(&storagePassword, "storage-password", "", "Duplicacy storage encryption password (or DUPLICACY_PASSWORD env)")
	pruneCmd.Flags().StringVar(&gcdToken, "gcd-token", "", "Google Drive token file path (for gcd:// storages)")
}

func runPruneCmd(cmd *cobra.Command, args []string) error {
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

	var hasErrors bool

	for _, storage := range storages {
		fmt.Printf("==> Pruning storage '%s'\n", storage)

		pruneArgs := []string{"prune", "-storage", storage}
		pruneArgs = append(pruneArgs, strings.Fields(pruneOptions)...)

		err := exec.RunDuplicacyWithStorage(storage, pruneArgs...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: prune on %s failed: %v\n", storage, err)
			hasErrors = true
			continue
		}
		fmt.Printf("    Prune on '%s' completed successfully\n", storage)
	}

	if hasErrors {
		return fmt.Errorf("prune completed with errors")
	}

	fmt.Println("==> All prune operations completed successfully")
	return nil
}
