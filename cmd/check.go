package cmd

import (
	"fmt"
	"os"

	"github.com/lioreshai/duplicaci/internal/executor"
	"github.com/spf13/cobra"
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
	})

	var hasErrors bool

	for _, storage := range storages {
		fmt.Printf("==> Checking storage '%s'\n", storage)

		err := exec.RunDuplicacyWithStorage(storage, "check", "-storage", storage)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: check on %s failed: %v\n", storage, err)
			hasErrors = true
			continue
		}
		fmt.Printf("    Check on '%s' completed successfully\n", storage)
	}

	if hasErrors {
		return fmt.Errorf("check completed with errors")
	}

	fmt.Println("==> All checks completed successfully")
	return nil
}
