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
	exec := New(Options{
		Verbose: true,
	})

	// Test duplicacy is installed and accessible
	err := exec.execute("duplicacy -version")
	if err != nil {
		t.Fatalf("duplicacy not found or not working: %v", err)
	}
}

func TestIntegration_DuplicacyList(t *testing.T) {
	_, _, container, repoPath, storage, duplicacyPath := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container, // empty if not using Docker
		DuplicacyPath:   duplicacyPath,
		Verbose:         true,
	})

	// Run duplicacy list - this is a read-only command
	err := exec.RunDuplicacy("-d", repoPath, "list", "-storage", storage)
	if err != nil {
		t.Fatalf("duplicacy list failed: %v", err)
	}
}

func TestIntegration_DuplicacyBackupAndList(t *testing.T) {
	_, _, container, repoPath, storage, duplicacyPath := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
		DuplicacyPath:   duplicacyPath,
		Verbose:         true,
	})

	// Run a backup
	err := exec.RunDuplicacy("-d", repoPath, "backup", "-storage", storage)
	if err != nil {
		t.Fatalf("duplicacy backup failed: %v", err)
	}

	// Verify backup shows in list
	err = exec.RunDuplicacy("-d", repoPath, "list", "-storage", storage)
	if err != nil {
		t.Fatalf("duplicacy list after backup failed: %v", err)
	}
}

func TestIntegration_DuplicacyCheck(t *testing.T) {
	_, _, container, repoPath, storage, duplicacyPath := getIntegrationConfig(t)

	if storage == "" {
		t.Skip("INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
		DuplicacyPath:   duplicacyPath,
		Verbose:         true,
	})

	// Run check - read-only verification
	err := exec.RunDuplicacy("-d", repoPath, "check", "-storage", storage)
	if err != nil {
		t.Fatalf("duplicacy check failed: %v", err)
	}
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
