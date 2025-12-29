package executor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Options configures the executor
type Options struct {
	DryRun          bool
	Verbose         bool
	DockerContainer string
	SSHHost         string
	SSHPassword     string
	DuplicacyPath   string // Path to duplicacy binary (default: "duplicacy")
}

// Executor runs duplicacy commands
type Executor struct {
	opts Options
}

// New creates a new Executor
func New(opts Options) *Executor {
	return &Executor{opts: opts}
}

// RunDuplicacy executes a duplicacy command with the given arguments
func (e *Executor) RunDuplicacy(args ...string) error {
	// Build the full command
	cmdStr := e.buildCommand(args)

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
func (e *Executor) buildCommand(args []string) string {
	// Base duplicacy command (use custom path if specified)
	duplicacyBin := e.opts.DuplicacyPath
	if duplicacyBin == "" {
		duplicacyBin = "duplicacy"
	}
	duplicacyCmd := duplicacyBin + " " + strings.Join(args, " ")

	// Wrap in docker exec if container specified
	if e.opts.DockerContainer != "" {
		duplicacyCmd = fmt.Sprintf("docker exec %s %s", e.opts.DockerContainer, duplicacyCmd)
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
