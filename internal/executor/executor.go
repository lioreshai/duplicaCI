package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Options configures the executor
type Options struct {
	DryRun           bool
	Verbose          bool
	DockerContainer  string
	SSHHost          string
	SSHPassword      string
	DuplicacyPath    string            // Path to duplicacy binary (default: auto-discover)
	RepoPath         string            // Repository path to cd into before running duplicacy
	CacheDir         string            // Duplicacy Web GUI cache directory (e.g., /cache/localhost/0)
	StoragePassword  string            // Default storage encryption password
	StoragePasswords map[string]string // Per-storage passwords (storage name -> password)
}

// Executor runs duplicacy commands
type Executor struct {
	opts           Options
	discoveredPath string
	discoverOnce   sync.Once
	discoverErr    error
}

// New creates a new Executor
func New(opts Options) *Executor {
	return &Executor{opts: opts}
}

// discoverDuplicacyPath finds the duplicacy CLI binary in a Docker container
// The web UI downloads it to /config/bin/duplicacy_linux_x64_<version>
func (e *Executor) discoverDuplicacyPath() (string, error) {
	e.discoverOnce.Do(func() {
		// If explicit path provided, use it
		if e.opts.DuplicacyPath != "" {
			e.discoveredPath = e.opts.DuplicacyPath
			return
		}

		// If not using Docker, default to "duplicacy" in PATH
		if e.opts.DockerContainer == "" {
			e.discoveredPath = "duplicacy"
			return
		}

		// In dry-run mode, don't try to discover - use default
		if e.opts.DryRun {
			e.discoveredPath = "duplicacy"
			return
		}

		// Search for CLI in Docker container
		searchCmd := fmt.Sprintf("docker exec %s sh -c 'ls /config/bin/duplicacy_linux_x64_* 2>/dev/null | head -1'",
			e.opts.DockerContainer)

		// Wrap in SSH if needed
		if e.opts.SSHHost != "" {
			escapedCmd := strings.ReplaceAll(searchCmd, "'", "'\"'\"'")
			searchCmd = fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o LogLevel=ERROR %s '%s'", e.opts.SSHHost, escapedCmd)
			if e.opts.SSHPassword != "" {
				searchCmd = fmt.Sprintf("sshpass -p '%s' %s",
					strings.ReplaceAll(e.opts.SSHPassword, "'", "'\"'\"'"),
					searchCmd)
			}
		}

		cmd := exec.Command("bash", "-c", searchCmd)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			e.discoverErr = fmt.Errorf("failed to discover duplicacy path: %w", err)
			return
		}

		path := strings.TrimSpace(out.String())
		if path == "" {
			e.discoverErr = fmt.Errorf("duplicacy CLI not found in /config/bin/")
			return
		}

		e.discoveredPath = path
		if e.opts.Verbose {
			fmt.Printf("    Discovered duplicacy at: %s\n", path)
		}
	})

	return e.discoveredPath, e.discoverErr
}

// RunDuplicacy executes a duplicacy command with the given arguments
func (e *Executor) RunDuplicacy(args ...string) error {
	return e.RunDuplicacyWithStorage("", args...)
}

// RunDuplicacyWithStorage executes a duplicacy command with storage-specific password
func (e *Executor) RunDuplicacyWithStorage(storageName string, args ...string) error {
	// Discover duplicacy path first (cached after first call)
	duplicacyBin, err := e.discoverDuplicacyPath()
	if err != nil {
		return fmt.Errorf("cannot find duplicacy: %w", err)
	}

	// Build the full command with storage-specific password
	cmdStr := e.buildCommandWithStorage(duplicacyBin, args, storageName)

	if e.opts.Verbose || e.opts.DryRun {
		fmt.Printf("    Command: %s\n", cmdStr)
	}

	if e.opts.DryRun {
		return nil
	}

	// Execute the command
	return e.execute(cmdStr)
}

// buildCommand constructs the full command string (for backward compatibility)
func (e *Executor) buildCommand(duplicacyBin string, args []string) string {
	return e.buildCommandWithStorage(duplicacyBin, args, "")
}

// buildCommandWithStorage constructs the full command string with storage-specific password
func (e *Executor) buildCommandWithStorage(duplicacyBin string, args []string, storageName string) string {
	duplicacyCmd := duplicacyBin + " " + strings.Join(args, " ")

	// Determine working directory: CacheDir takes precedence over RepoPath
	workDir := e.opts.CacheDir
	if workDir == "" {
		workDir = e.opts.RepoPath
	}

	// If working directory specified, cd to it first
	if workDir != "" {
		duplicacyCmd = fmt.Sprintf("cd %s && %s", workDir, duplicacyCmd)
	}

	// Build docker exec with optional environment variables
	if e.opts.DockerContainer != "" {
		dockerEnv := ""

		// Get the password for this storage (check per-storage first, then default)
		password := e.getStoragePassword(storageName)
		if password != "" {
			// Pass the generic DUPLICACY_PASSWORD which works for all storages
			dockerEnv = fmt.Sprintf("-e DUPLICACY_PASSWORD='%s' ", password)
		}

		if workDir != "" {
			// Need sh -c to handle cd && command
			duplicacyCmd = fmt.Sprintf("docker exec %s%s sh -c '%s'", dockerEnv, e.opts.DockerContainer, duplicacyCmd)
		} else {
			// Simple command, no shell needed
			duplicacyCmd = fmt.Sprintf("docker exec %s%s %s", dockerEnv, e.opts.DockerContainer, duplicacyCmd)
		}
	}

	// Wrap in SSH if host specified
	if e.opts.SSHHost != "" {
		// Escape single quotes in the command
		escapedCmd := strings.ReplaceAll(duplicacyCmd, "'", "'\"'\"'")
		duplicacyCmd = fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o LogLevel=ERROR %s '%s'", e.opts.SSHHost, escapedCmd)

		// Add sshpass if password provided
		if e.opts.SSHPassword != "" {
			duplicacyCmd = fmt.Sprintf("sshpass -p '%s' %s",
				strings.ReplaceAll(e.opts.SSHPassword, "'", "'\"'\"'"),
				duplicacyCmd)
		}
	}

	return duplicacyCmd
}

// getStoragePassword returns the password for a storage, checking per-storage first then default
func (e *Executor) getStoragePassword(storageName string) string {
	// Check per-storage passwords first
	if storageName != "" && e.opts.StoragePasswords != nil {
		if pw, ok := e.opts.StoragePasswords[storageName]; ok {
			return pw
		}
	}
	// Fall back to default password
	return e.opts.StoragePassword
}

// execute runs the command and streams output
func (e *Executor) execute(cmdStr string) error {
	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("command exited with code %d", exitErr.ExitCode())
		}
		return err
	}

	return nil
}
