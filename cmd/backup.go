package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/lioreshai/duplicaci/internal/config"
	"github.com/lioreshai/duplicaci/internal/executor"
	"github.com/lioreshai/duplicaci/internal/notifier"
	"github.com/spf13/cobra"
)

var (
	// Backup flags
	repository      string
	repoPath        string
	cacheDir        string
	storages        []string
	backupOptions   string
	runPrune        bool
	pruneOptions    string
	runCheck        bool
	dockerContainer string
	sshHost         string
	sshPassword     string
	storagePassword string
	gcdToken        string

	// Notification flags
	createIssues bool
	forgejoURL   string
	forgejoRepo  string
	forgejoToken string
	assignee     string
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Run a Duplicacy backup",
	Long: `Run a Duplicacy backup for the specified repository to one or more storage backends.

Optionally run prune and/or check operations after the backup completes.`,
	RunE: runBackup,
}

func init() {
	backupCmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository ID to backup")
	backupCmd.Flags().StringVarP(&repoPath, "repo-path", "p", "", "Path to repository (cd here before running duplicacy)")
	backupCmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Duplicacy Web GUI cache directory (e.g., /cache/localhost/0)")
	backupCmd.Flags().StringSliceVarP(&storages, "storage", "s", []string{}, "Storage backend(s) to backup to")
	backupCmd.Flags().StringVar(&backupOptions, "backup-options", "", "Additional backup options (e.g., '-threads 4')")
	backupCmd.Flags().BoolVar(&runPrune, "prune", false, "Run prune after backup")
	backupCmd.Flags().StringVar(&pruneOptions, "prune-options", "-keep 0:180 -keep 7:14 -keep 1:1 -a", "Prune retention options")
	backupCmd.Flags().BoolVar(&runCheck, "check", false, "Run check after backup")

	backupCmd.Flags().StringVar(&dockerContainer, "docker-container", "", "Run inside Docker container")
	backupCmd.Flags().StringVar(&sshHost, "ssh-host", "", "SSH to host before running (user@host)")
	backupCmd.Flags().StringVar(&sshPassword, "ssh-password", "", "SSH password (or SSH_PASSWORD env)")
	backupCmd.Flags().StringVar(&storagePassword, "storage-password", "", "Duplicacy storage encryption password (or DUPLICACY_PASSWORD env)")
	backupCmd.Flags().StringVar(&gcdToken, "gcd-token", "", "Google Drive token file path (for gcd:// storages)")

	backupCmd.Flags().BoolVar(&createIssues, "create-issues", false, "Create Forgejo/GitHub issue on failure")
	backupCmd.Flags().StringVar(&forgejoURL, "forgejo-url", "", "Forgejo server URL")
	backupCmd.Flags().StringVar(&forgejoRepo, "forgejo-repo", "", "Repository for issues (owner/repo)")
	backupCmd.Flags().StringVar(&forgejoToken, "forgejo-token", "", "Forgejo API token (or FORGEJO_TOKEN env)")
	backupCmd.Flags().StringVar(&assignee, "assignee", "", "Assign issues to this user")
}

func runBackup(cmd *cobra.Command, args []string) error {
	// Load config file if specified
	var cfg *config.Config
	var err error

	if configFile != "" {
		cfg, err = config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		// Apply config values if flags not set
		applyConfig(cfg)
	}

	// Validate required fields
	if repository == "" {
		return fmt.Errorf("--repository is required")
	}
	if len(storages) == 0 {
		return fmt.Errorf("at least one --storage is required")
	}

	// Get SSH password from env if not set
	if sshPassword == "" {
		sshPassword = os.Getenv("SSH_PASSWORD")
	}

	// Get storage password from env if not set
	if storagePassword == "" {
		storagePassword = os.Getenv("DUPLICACY_PASSWORD")
	}

	// Get Forgejo token from env if not set
	if forgejoToken == "" {
		forgejoToken = os.Getenv("FORGEJO_TOKEN")
	}

	// Create executor
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

	var allErrors []string

	// Run backup for each storage
	for _, storage := range storages {
		fmt.Printf("==> Backing up repository '%s' to storage '%s'\n", repository, storage)

		backupArgs := []string{"backup", "-storage", storage}
		if backupOptions != "" {
			backupArgs = append(backupArgs, strings.Fields(backupOptions)...)
		}

		err := exec.RunDuplicacyWithStorage(storage, backupArgs...)
		if err != nil {
			errMsg := fmt.Sprintf("backup to %s failed: %v", storage, err)
			allErrors = append(allErrors, errMsg)
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", errMsg)
			continue
		}
		fmt.Printf("    Backup to '%s' completed successfully\n", storage)
	}

	// Run check if requested (after backup, before prune)
	// Per Duplicacy best practice: backup → check → prune
	if runCheck && len(allErrors) == 0 {
		for _, storage := range storages {
			fmt.Printf("==> Checking storage '%s'\n", storage)

			err := exec.RunDuplicacyWithStorage(storage, "check", "-storage", storage)
			if err != nil {
				errMsg := fmt.Sprintf("check on %s failed: %v", storage, err)
				allErrors = append(allErrors, errMsg)
				fmt.Fprintf(os.Stderr, "ERROR: %s\n", errMsg)
			}
		}
	}

	// Run prune if requested and backup/check succeeded
	if runPrune && len(allErrors) == 0 {
		for _, storage := range storages {
			fmt.Printf("==> Pruning storage '%s'\n", storage)

			pruneArgs := []string{"prune", "-storage", storage}
			pruneArgs = append(pruneArgs, strings.Fields(pruneOptions)...)

			err := exec.RunDuplicacyWithStorage(storage, pruneArgs...)
			if err != nil {
				errMsg := fmt.Sprintf("prune on %s failed: %v", storage, err)
				allErrors = append(allErrors, errMsg)
				fmt.Fprintf(os.Stderr, "ERROR: %s\n", errMsg)
			}
		}
	}

	// Handle notifications
	if len(allErrors) > 0 && createIssues {
		if err := sendFailureNotification(allErrors); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Failed to create issue: %v\n", err)
		}
		return fmt.Errorf("backup completed with %d error(s)", len(allErrors))
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("backup completed with %d error(s)", len(allErrors))
	}

	fmt.Println("==> All operations completed successfully")
	return nil
}

func applyConfig(cfg *config.Config) {
	if sshHost == "" && cfg.SSH.Host != "" {
		sshHost = cfg.SSH.Host
	}
	if sshPassword == "" && cfg.SSH.PasswordEnv != "" {
		sshPassword = os.Getenv(cfg.SSH.PasswordEnv)
	}
	if dockerContainer == "" && cfg.Docker.Container != "" {
		dockerContainer = cfg.Docker.Container
	}
	if forgejoURL == "" && cfg.Notifications.Forgejo.URL != "" {
		forgejoURL = cfg.Notifications.Forgejo.URL
	}
	if forgejoRepo == "" && cfg.Notifications.Forgejo.Repo != "" {
		forgejoRepo = cfg.Notifications.Forgejo.Repo
	}
	if forgejoToken == "" && cfg.Notifications.Forgejo.TokenEnv != "" {
		forgejoToken = os.Getenv(cfg.Notifications.Forgejo.TokenEnv)
	}
	if assignee == "" && cfg.Notifications.Forgejo.Assignee != "" {
		assignee = cfg.Notifications.Forgejo.Assignee
	}
}

func sendFailureNotification(errors []string) error {
	if forgejoURL == "" || forgejoRepo == "" || forgejoToken == "" {
		return fmt.Errorf("forgejo notification requires --forgejo-url, --forgejo-repo, and --forgejo-token")
	}

	n := notifier.NewForgejo(forgejoURL, forgejoRepo, forgejoToken)
	if assignee != "" {
		n.SetAssignee(assignee)
	}

	title := fmt.Sprintf("[duplicaci] %s: backup failed", repository)
	body := fmt.Sprintf("## Backup Failure\n\n**Repository:** %s\n**Storages:** %s\n\n### Errors\n\n",
		repository, strings.Join(storages, ", "))

	for _, err := range errors {
		body += fmt.Sprintf("- %s\n", err)
	}

	return n.CreateOrUpdateIssue(title, body)
}
