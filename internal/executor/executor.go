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
	DryRun          bool
	Verbose         bool
	DockerContainer string
	SSHHost         string
	SSHPassword     string
	DuplicacyPath   string // Path to duplicacy binary (default: auto-discover)
	RepoPath        string // Repository path to cd into before running duplicacy
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
	// Discover duplicacy path first (cached after first call)
	duplicacyBin, err := e.discoverDuplicacyPath()
	if err != nil {
		return fmt.Errorf("cannot find duplicacy: %w", err)
	}

	// Build the full command
	cmdStr := e.buildCommand(duplicacyBin, args)

	if e.opts.Verbose || e.opts.DryRun {
		fmt.Printf("    Command: %s\n", cmdStr)
	}

	if e.opts.DryRun {
		return nil
	}

	// Execute the command
	return e.execute(cmdStr)
}

// buildCommand constructs the full command string
func (e *Executor) buildCommand(duplicacyBin string, args []string) string {
	duplicacyCmd := duplicacyBin + " " + strings.Join(args, " ")

	// If repo path specified, cd to it first
	if e.opts.RepoPath != "" {
		duplicacyCmd = fmt.Sprintf("cd %s && %s", e.opts.RepoPath, duplicacyCmd)
	}

	// Wrap in docker exec if container specified
	if e.opts.DockerContainer != "" {
		if e.opts.RepoPath != "" {
			// Need sh -c to handle cd && command
			duplicacyCmd = fmt.Sprintf("docker exec %s sh -c '%s'", e.opts.DockerContainer, duplicacyCmd)
		} else {
			// Simple command, no shell needed
			duplicacyCmd = fmt.Sprintf("docker exec %s %s", e.opts.DockerContainer, duplicacyCmd)
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
