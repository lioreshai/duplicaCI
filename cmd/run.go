package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/lioreshai/duplicaci/internal/config"
	"github.com/lioreshai/duplicaci/internal/executor"
	"github.com/lioreshai/duplicaci/internal/notifier"
	"github.com/lioreshai/duplicaci/internal/stats"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run all backups defined in config file",
	Long: `Run all backup, prune, and check operations defined in the configuration file.

This is the recommended way to use duplicaci. Define your backups in a YAML config
file and let duplicaci handle the orchestration.

Example config (duplicaci.yaml):

  connection:
    host: root@192.168.1.100
    container: Duplicacy

  backups:
    - name: server_appdata
      path: /mnt/appdata
      destinations:
        - NASBackup
        - GoogleDrive
      retention:
        days: 14
        weeks: 180

  notifications:
    forgejo:
      url: https://git.example.com
      repo: user/repo
      assignee: user

Then run: duplicaci run --config duplicaci.yaml`,
	RunE: runAllBackups,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runAllBackups(cmd *cobra.Command, args []string) error {
	// Config file is required for run command
	if configFile == "" {
		return fmt.Errorf("--config is required for the run command")
	}

	// Load config
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Get credentials from environment
	sshPassword := os.Getenv("SSH_PASSWORD")
	storagePassword := os.Getenv("DUPLICACY_PASSWORD")

	// Track all errors
	var allErrors []string
	var failedBackups []string

	// Phase 1: Run backups
	fmt.Println("==========================================")
	fmt.Println("Phase 1: Backups")
	fmt.Println("==========================================")

	for _, backup := range cfg.Backups {
		fmt.Printf("\n==> Backing up '%s'\n", backup.Name)

		// Determine cache directory
		cacheDir := backup.CacheDir
		if cacheDir == "" {
			// Auto-discover would go here, for now require it or use path
			cacheDir = backup.Path
		}

		// Update executor with this backup's cache dir
		backupExec := executor.New(executor.Options{
			DryRun:          dryRun,
			Verbose:         verbose,
			DockerContainer: cfg.Connection.Container,
			SSHHost:         cfg.Connection.Host,
			SSHPassword:     sshPassword,
			StoragePassword: storagePassword,
			GCDToken:        cfg.Connection.GCDToken,
			CacheDir:        cacheDir,
		})

		backupFailed := false

		// Backup to each destination
		for _, dest := range backup.Destinations {
			fmt.Printf("    -> %s\n", dest)

			backupArgs := []string{"backup", "-storage", dest}
			if backup.Threads > 1 {
				backupArgs = append(backupArgs, "-threads", fmt.Sprintf("%d", backup.Threads))
			}

			err := backupExec.RunDuplicacyWithStorage(dest, backupArgs...)
			if err != nil {
				errMsg := fmt.Sprintf("%s -> %s: %v", backup.Name, dest, err)
				allErrors = append(allErrors, errMsg)
				fmt.Fprintf(os.Stderr, "       ERROR: %v\n", err)
				backupFailed = true
				continue
			}
			fmt.Printf("       OK\n")
		}

		if backupFailed {
			failedBackups = append(failedBackups, backup.Name)
		}
	}

	// Phase 2: Prune all storages
	fmt.Println("\n==========================================")
	fmt.Println("Phase 2: Prune")
	fmt.Println("==========================================")

	allStorages := cfg.AllStorages()

	// Use first backup's cache dir for prune/check, or empty if no backups
	var maintenanceCacheDir string
	if len(cfg.Backups) > 0 {
		maintenanceCacheDir = cfg.Backups[0].CacheDir
		if maintenanceCacheDir == "" {
			maintenanceCacheDir = cfg.Backups[0].Path
		}
	}

	maintenanceExec := executor.New(executor.Options{
		DryRun:          dryRun,
		Verbose:         verbose,
		DockerContainer: cfg.Connection.Container,
		SSHHost:         cfg.Connection.Host,
		SSHPassword:     sshPassword,
		StoragePassword: storagePassword,
		GCDToken:        cfg.Connection.GCDToken,
		CacheDir:        maintenanceCacheDir,
	})

	for _, storage := range allStorages {
		// Check if storage has retention defined
		if retention, ok := cfg.GetStorageRetention(storage); ok {
			// Storage-level retention: prune all repositories with -a
			fmt.Printf("\n==> Pruning '%s' (all repositories)\n", storage)

			pruneArgs := []string{"prune", "-storage", storage}
			pruneArgs = append(pruneArgs, strings.Fields(retention.ToPruneOptions())...)

			err := maintenanceExec.RunDuplicacyWithStorage(storage, pruneArgs...)
			if err != nil {
				errMsg := fmt.Sprintf("prune %s: %v", storage, err)
				allErrors = append(allErrors, errMsg)
				fmt.Fprintf(os.Stderr, "    ERROR: %v\n", err)
			} else {
				fmt.Printf("    OK\n")
			}
		} else {
			// Per-backup retention: prune each repository separately with -id
			backups := cfg.BackupsForStorage(storage)
			if len(backups) == 0 {
				// Maintenance-only storage with no backups targeting it
				// Use default retention with -a
				fmt.Printf("\n==> Pruning '%s' (maintenance, default retention)\n", storage)

				defaultRetention := config.RetentionConfig{Daily: 7, Weekly: 4}
				pruneArgs := []string{"prune", "-storage", storage}
				pruneArgs = append(pruneArgs, strings.Fields(defaultRetention.ToPruneOptions())...)

				err := maintenanceExec.RunDuplicacyWithStorage(storage, pruneArgs...)
				if err != nil {
					errMsg := fmt.Sprintf("prune %s: %v", storage, err)
					allErrors = append(allErrors, errMsg)
					fmt.Fprintf(os.Stderr, "    ERROR: %v\n", err)
				} else {
					fmt.Printf("    OK\n")
				}
			} else {
				// Prune each backup's repository separately
				for _, backupName := range backups {
					fmt.Printf("\n==> Pruning '%s' (repository: %s)\n", storage, backupName)

					retention := cfg.GetBackupRetention(backupName)
					pruneArgs := []string{"prune", "-storage", storage, "-id", backupName}
					// Remove -a from options since we're targeting specific repository
					opts := retention.ToPruneOptionsWithoutAll()
					pruneArgs = append(pruneArgs, strings.Fields(opts)...)

					err := maintenanceExec.RunDuplicacyWithStorage(storage, pruneArgs...)
					if err != nil {
						errMsg := fmt.Sprintf("prune %s/%s: %v", storage, backupName, err)
						allErrors = append(allErrors, errMsg)
						fmt.Fprintf(os.Stderr, "    ERROR: %v\n", err)
						continue
					}
					fmt.Printf("    OK\n")
				}
			}
		}
	}

	// Phase 3: Check all storages
	fmt.Println("\n==========================================")
	fmt.Println("Phase 3: Check")
	fmt.Println("==========================================")

	// Create stats writer for updating Duplicacy Web UI stats
	var statsWriter *stats.Writer
	if cfg.Connection.Container != "" {
		statsWriter = stats.NewWriter(cfg.Connection.Host, sshPassword, cfg.Connection.Container)
		statsWriter.DryRun = dryRun
		statsWriter.Verbose = verbose
	}

	for _, storage := range allStorages {
		fmt.Printf("\n==> Checking '%s'\n", storage)

		// Run check with -tabular to get stats output
		output, err := maintenanceExec.RunDuplicacyCaptureWithStorage(storage, "check", "-tabular", "-storage", storage)

		// Print the output (since we captured it)
		if output != "" {
			fmt.Print(output)
		}

		if err != nil {
			errMsg := fmt.Sprintf("check %s: %v", storage, err)
			allErrors = append(allErrors, errMsg)
			fmt.Fprintf(os.Stderr, "    ERROR: %v\n", err)
			continue
		}
		fmt.Printf("    OK\n")

		// Update stats for Duplicacy Web UI
		if statsWriter != nil && output != "" {
			dayStats, parseErr := stats.ParseCheckOutput(output)
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "    WARNING: failed to parse check output for stats: %v\n", parseErr)
			} else {
				// Print parsed stats summary for CI visibility
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

	// Summary
	fmt.Println("\n==========================================")
	fmt.Println("Summary")
	fmt.Println("==========================================")

	if len(allErrors) == 0 {
		fmt.Println("All operations completed successfully")
		return nil
	}

	// Report errors
	fmt.Printf("\n%d error(s) occurred:\n", len(allErrors))
	for _, e := range allErrors {
		fmt.Printf("  - %s\n", e)
	}

	// Send notification if configured
	if cfg.Notifications.Forgejo.URL != "" && cfg.Notifications.Forgejo.Repo != "" {
		token := cfg.Notifications.Forgejo.GetToken()
		if token != "" {
			if err := sendRunFailureNotification(cfg, allErrors, failedBackups); err != nil {
				fmt.Fprintf(os.Stderr, "\nWARNING: Failed to create issue: %v\n", err)
			}
		}
	}

	return fmt.Errorf("completed with %d error(s)", len(allErrors))
}

func sendRunFailureNotification(cfg *config.Config, errors []string, failedBackups []string) error {
	n := notifier.NewForgejo(
		cfg.Notifications.Forgejo.URL,
		cfg.Notifications.Forgejo.Repo,
		cfg.Notifications.Forgejo.GetToken(),
	)

	if cfg.Notifications.Forgejo.Assignee != "" {
		n.SetAssignee(cfg.Notifications.Forgejo.Assignee)
	}

	// Build title
	var title string
	if len(failedBackups) > 0 {
		title = fmt.Sprintf("[duplicaci] %s: backup failed", strings.Join(failedBackups, ", "))
	} else {
		title = "[duplicaci] maintenance failed"
	}

	// Build body
	body := "## Backup Run Failed\n\n"

	if len(failedBackups) > 0 {
		body += fmt.Sprintf("**Failed backups:** %s\n\n", strings.Join(failedBackups, ", "))
	}

	body += "### Errors\n\n"
	for _, e := range errors {
		body += fmt.Sprintf("- %s\n", e)
	}

	return n.CreateOrUpdateIssue(title, body)
}
