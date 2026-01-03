//go:build integration

package executor

import (
	"os"
	"strings"
	"testing"
)

// Integration tests can run in two modes:
//
// 1. Local mode (CI - no Docker):
//    - INTEGRATION_REPO_PATH: Path to initialized Duplicacy repository
//    - INTEGRATION_STORAGE: Storage name to test with
//
// 2. Docker mode (optional):
//    - INTEGRATION_DOCKER_CONTAINER: Docker container name
//    - Plus the repo/storage vars above
//
// 3. Remote SSH mode (optional):
//    - INTEGRATION_SSH_HOST: SSH host (e.g., root@192.168.1.100)
//    - INTEGRATION_SSH_PASSWORD: SSH password
//    - Plus Docker/repo/storage vars above
//
// Run with: go test -tags=integration -v ./internal/executor/

func getIntegrationConfig(t *testing.T) (host, password, container, repoPath, storage, duplicacyPath string) {
	host = os.Getenv("INTEGRATION_SSH_HOST")
	password = os.Getenv("INTEGRATION_SSH_PASSWORD")
	container = os.Getenv("INTEGRATION_DOCKER_CONTAINER")
	repoPath = os.Getenv("INTEGRATION_REPO_PATH")
	storage = os.Getenv("INTEGRATION_STORAGE")
	duplicacyPath = os.Getenv("INTEGRATION_DUPLICACY_PATH") // e.g., /usr/bin/duplicacy

	// At minimum we need a repo path to test with
	if repoPath == "" {
		t.Skip("INTEGRATION_REPO_PATH required")
	}

	return
}

func TestIntegration_DuplicacyVersion(t *testing.T) {
	container := os.Getenv("INTEGRATION_DOCKER_CONTAINER")
	if container != "" {
		t.Skip("Skipping local duplicacy test when using Docker")
	}

	// This test requires duplicacy to be installed locally
	// Skip if not in a proper integration test environment
	repoPath := os.Getenv("INTEGRATION_REPO_PATH")
	if repoPath == "" {
		t.Skip("INTEGRATION_REPO_PATH required for local duplicacy test")
	}

	exec := New(Options{
		Verbose: true,
	})

	// Test duplicacy is installed and accessible locally
	err := exec.execute("duplicacy -version")
	if err != nil {
		t.Fatalf("duplicacy not found or not working: %v", err)
	}
}

func TestIntegration_DuplicacyList(t *testing.T) {
	_, _, container, repoPath, storage, _ := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container, // empty if not using Docker
		RepoPath:        repoPath,
		Verbose:         true,
	})

	// Run duplicacy list - this is a read-only command
	err := exec.RunDuplicacy("list", "-storage", storage)
	if err != nil {
		t.Fatalf("duplicacy list failed: %v", err)
	}
}

func TestIntegration_DuplicacyBackupAndList(t *testing.T) {
	_, _, container, repoPath, storage, _ := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
		RepoPath:        repoPath,
		Verbose:         true,
	})

	// Run a backup - exit code 100 means "nothing to backup" which is OK
	// (test files may not be visible due to container permissions)
	err := exec.RunDuplicacy("backup", "-storage", storage)
	if err != nil {
		// Exit code 100 = nothing to backup, which is acceptable
		if !strings.Contains(err.Error(), "code 100") {
			t.Fatalf("duplicacy backup failed: %v", err)
		}
		t.Log("backup returned 'nothing to backup' (exit 100) - acceptable")
	}

	// Verify list works
	err = exec.RunDuplicacy("list", "-storage", storage)
	if err != nil {
		t.Fatalf("duplicacy list after backup failed: %v", err)
	}
}

func TestIntegration_DuplicacyBackupWithOptions(t *testing.T) {
	_, _, container, repoPath, storage, _ := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
		RepoPath:        repoPath,
		Verbose:         true,
	})

	// Run backup with -threads 4 (same as production)
	err := exec.RunDuplicacy("backup", "-storage", storage, "-threads", "4")
	if err != nil {
		// Exit code 100 = nothing to backup, which is acceptable
		if !strings.Contains(err.Error(), "code 100") {
			t.Fatalf("duplicacy backup with -threads failed: %v", err)
		}
		t.Log("backup with -threads returned 'nothing to backup' (exit 100) - acceptable")
	}
}

func TestIntegration_DuplicacyCheck(t *testing.T) {
	_, _, container, repoPath, storage, _ := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
		RepoPath:        repoPath,
		Verbose:         true,
	})

	// Run check - read-only verification
	err := exec.RunDuplicacy("check", "-storage", storage)
	if err != nil {
		t.Fatalf("duplicacy check failed: %v", err)
	}
}

func TestIntegration_DuplicacyPrune(t *testing.T) {
	_, _, container, repoPath, storage, _ := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
		RepoPath:        repoPath,
		Verbose:         true,
	})

	// Run prune with same options as production
	err := exec.RunDuplicacy("prune", "-storage", storage, "-keep", "0:180", "-keep", "7:14", "-keep", "1:1", "-a")
	if err != nil {
		t.Fatalf("duplicacy prune failed: %v", err)
	}
}

func TestIntegration_FullWorkflow(t *testing.T) {
	_, _, container, repoPath, storage, _ := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
		RepoPath:        repoPath,
		Verbose:         true,
	})

	// Full workflow: backup → check → prune (per duplicacy best practice)
	t.Log("Step 1: Running backup...")
	err := exec.RunDuplicacy("backup", "-storage", storage)
	if err != nil {
		// Exit code 100 = nothing to backup, acceptable
		if !strings.Contains(err.Error(), "code 100") {
			t.Fatalf("backup failed: %v", err)
		}
		t.Log("backup returned 'nothing to backup' (exit 100) - acceptable")
	}

	t.Log("Step 2: Running check...")
	err = exec.RunDuplicacy("check", "-storage", storage)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	t.Log("Step 3: Running prune...")
	err = exec.RunDuplicacy("prune", "-storage", storage, "-keep", "0:180", "-keep", "7:14", "-keep", "1:1", "-a")
	if err != nil {
		t.Fatalf("prune failed: %v", err)
	}

	t.Log("Full workflow completed successfully")
}

func TestIntegration_CommandBuilding_LocalDirect(t *testing.T) {
	exec := New(Options{
		Verbose: true,
	})

	// Verify command is built correctly for local direct execution
	cmd := exec.buildCommand("duplicacy", []string{"list", "-storage", "test"})

	// Should just be duplicacy command
	if !strings.HasPrefix(cmd, "duplicacy list -storage test") {
		t.Errorf("expected direct duplicacy command, got: %s", cmd)
	}

	// Should NOT contain docker or SSH
	if strings.Contains(cmd, "docker") {
		t.Errorf("local mode should not contain docker, got: %s", cmd)
	}
	if strings.Contains(cmd, "ssh") {
		t.Errorf("local mode should not contain ssh, got: %s", cmd)
	}
}

func TestIntegration_CommandBuilding_Docker(t *testing.T) {
	_, _, container, _, _, duplicacyPath := getIntegrationConfig(t)

	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required for Docker test")
	}

	// Use the discovered path or default
	binPath := duplicacyPath
	if binPath == "" {
		binPath = "duplicacy"
	}

	exec := New(Options{
		DockerContainer: container,
		DuplicacyPath:   duplicacyPath,
		Verbose:         true,
	})

	cmd := exec.buildCommand(binPath, []string{"list", "-storage", "test"})

	if !strings.Contains(cmd, "docker exec "+container) {
		t.Errorf("command should contain docker exec, got: %s", cmd)
	}
}

func TestIntegration_CommandBuilding_SSH(t *testing.T) {
	host, password, container, _, _, duplicacyPath := getIntegrationConfig(t)

	if host == "" || password == "" {
		t.Skip("SSH tests require INTEGRATION_SSH_HOST and INTEGRATION_SSH_PASSWORD")
	}

	// Use the discovered path or default
	binPath := duplicacyPath
	if binPath == "" {
		binPath = "duplicacy"
	}

	exec := New(Options{
		SSHHost:         host,
		SSHPassword:     password,
		DockerContainer: container,
		DuplicacyPath:   duplicacyPath,
		Verbose:         true,
	})

	cmd := exec.buildCommand(binPath, []string{"list", "-storage", "test"})

	if !strings.Contains(cmd, "sshpass") {
		t.Errorf("command should contain sshpass, got: %s", cmd)
	}
	if !strings.Contains(cmd, "ssh") {
		t.Errorf("command should contain ssh, got: %s", cmd)
	}
}

func TestIntegration_DryRunDoesNotExecute(t *testing.T) {
	_, _, container, _, _, duplicacyPath := getIntegrationConfig(t)

	exec := New(Options{
		DockerContainer: container,
		DuplicacyPath:   duplicacyPath,
		DryRun:          true,
		Verbose:         true,
	})

	// With dry run, this should not actually execute
	err := exec.RunDuplicacy("backup", "-storage", "nonexistent")
	if err != nil {
		t.Errorf("dry run should not return error: %v", err)
	}
}

func TestIntegration_DuplicacyCheckTabular(t *testing.T) {
	_, _, container, repoPath, storage, _ := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
		RepoPath:        repoPath,
		Verbose:         true,
	})

	// Run check with -tabular flag to get stats output
	output, err := exec.RunDuplicacyCaptureWithStorage(storage, "check", "-tabular", "-storage", storage)
	if err != nil {
		t.Fatalf("duplicacy check -tabular failed: %v", err)
	}

	// Verify we got some output
	if output == "" {
		t.Error("expected non-empty output from check -tabular")
	}

	// Log the output for debugging
	t.Logf("Check output length: %d bytes", len(output))

	// Verify output contains expected patterns
	if !strings.Contains(output, "SNAPSHOT_CHECK") && !strings.Contains(output, "Storage set to") {
		t.Error("output should contain SNAPSHOT_CHECK or Storage info")
	}
}

func TestIntegration_CaptureVsStream(t *testing.T) {
	_, _, container, repoPath, storage, _ := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
		RepoPath:        repoPath,
		Verbose:         true,
	})

	// Capture method should return the output
	output, err := exec.RunDuplicacyCaptureWithStorage(storage, "list", "-storage", storage)
	if err != nil {
		t.Fatalf("RunDuplicacyCaptureWithStorage failed: %v", err)
	}

	t.Logf("Captured output: %d bytes", len(output))

	// Stream method should not return output but not error
	err = exec.RunDuplicacyWithStorage(storage, "list", "-storage", storage)
	if err != nil {
		t.Fatalf("RunDuplicacyWithStorage failed: %v", err)
	}
}

func TestIntegration_FullCheckWithStatsWorkflow(t *testing.T) {
	_, _, container, repoPath, storage, _ := getIntegrationConfig(t)

	if storage == "" || container == "" {
		t.Skip("INTEGRATION_STORAGE and INTEGRATION_DOCKER_CONTAINER required")
	}

	exec := New(Options{
		DockerContainer: container,
		RepoPath:        repoPath,
		Verbose:         true,
	})

	// Run check with -tabular to get stats output (same as what run command does)
	output, err := exec.RunDuplicacyCaptureWithStorage(storage, "check", "-tabular", "-storage", storage)
	if err != nil {
		t.Fatalf("check -tabular failed: %v", err)
	}

	// Verify we can see the output (for CI visibility)
	t.Logf("Check output:\n%s", output)

	// Verify output contains the tabular format markers
	if !strings.Contains(output, "all") {
		t.Log("Note: output does not contain 'all' summary row - may have no revisions")
	}
}
