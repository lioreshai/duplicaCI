package stats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Writer handles updating stats files via SSH/Docker
type Writer struct {
	SSHHost         string
	SSHPassword     string
	DockerContainer string
	StatsPath       string // default: /config/stats/storages
	DryRun          bool
	Verbose         bool
}

// NewWriter creates a new stats writer
func NewWriter(sshHost, sshPassword, dockerContainer string) *Writer {
	return &Writer{
		SSHHost:         sshHost,
		SSHPassword:     sshPassword,
		DockerContainer: dockerContainer,
		StatsPath:       "/config/stats/storages",
	}
}

// UpdateStorageStats reads existing stats, adds today's entry, writes back
func (w *Writer) UpdateStorageStats(storage string, dayStats *DayStats) error {
	statsFile := fmt.Sprintf("%s/%s.stats", w.StatsPath, storage)

	// Read existing stats
	existingStats, err := w.readStatsFile(statsFile)
	if err != nil {
		// If file doesn't exist, start fresh
		existingStats = make(StorageStats)
	}

	// Add/update today's entry
	today := TodayDate()
	existingStats[today] = dayStats

	// Write back
	return w.writeStatsFile(statsFile, existingStats)
}

// readStatsFile reads and parses a stats file from the Docker container
func (w *Writer) readStatsFile(path string) (StorageStats, error) {
	cmd := w.buildDockerCommand(fmt.Sprintf("cat %s 2>/dev/null || echo '{}'", path))

	if w.Verbose {
		fmt.Printf("    Reading stats: %s\n", path)
	}

	output, err := w.executeCapture(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to read stats file: %w", err)
	}

	var stats StorageStats
	if err := json.Unmarshal([]byte(output), &stats); err != nil {
		// If parsing fails, return empty stats
		return make(StorageStats), nil
	}

	return stats, nil
}

// writeStatsFile writes stats to a file in the Docker container
func (w *Writer) writeStatsFile(path string, stats StorageStats) error {
	// Marshal with indentation to match Duplicacy Web format
	data, err := json.MarshalIndent(stats, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	if w.DryRun {
		fmt.Printf("    [DRY-RUN] Would write to %s:\n%s\n", path, string(data))
		return nil
	}

	// Escape the JSON for shell
	escapedJSON := strings.ReplaceAll(string(data), "'", "'\"'\"'")

	// Write via cat with heredoc-style input
	cmd := w.buildDockerCommand(fmt.Sprintf("cat > %s << 'STATSEOF'\n%s\nSTATSEOF", path, escapedJSON))

	if w.Verbose {
		fmt.Printf("    Writing stats: %s\n", path)
	}

	if err := w.execute(cmd); err != nil {
		return fmt.Errorf("failed to write stats file: %w", err)
	}

	return nil
}

// buildDockerCommand constructs a command to run inside the Docker container
func (w *Writer) buildDockerCommand(shellCmd string) string {
	// Escape the shell command for docker exec
	dockerCmd := fmt.Sprintf("docker exec %s sh -c '%s'", w.DockerContainer, shellCmd)

	// Wrap in SSH if host specified
	if w.SSHHost != "" {
		// Escape single quotes in the command
		escapedCmd := strings.ReplaceAll(dockerCmd, "'", "'\"'\"'")
		dockerCmd = fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o LogLevel=ERROR %s '%s'", w.SSHHost, escapedCmd)

		// Add sshpass if password provided
		if w.SSHPassword != "" {
			dockerCmd = fmt.Sprintf("sshpass -p '%s' %s",
				strings.ReplaceAll(w.SSHPassword, "'", "'\"'\"'"),
				dockerCmd)
		}
	}

	return dockerCmd
}

// executeCapture runs a command and returns stdout
func (w *Writer) executeCapture(cmdStr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdStr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command failed: %w (stderr: %s)", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// execute runs a command and streams output
func (w *Writer) execute(cmdStr string) error {
	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
