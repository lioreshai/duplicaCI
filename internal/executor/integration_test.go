//go:build integration

package executor

import (
	"os"
	"strings"
	"testing"
)

// Integration tests can run in two modes:
//
// 1. Local Docker mode (CI):
//    - INTEGRATION_DOCKER_CONTAINER: Docker container name (e.g., Duplicacy)
//    - INTEGRATION_REPO_PATH: Path to repository inside container
//    - INTEGRATION_STORAGE: Storage name to test with
//
// 2. Remote SSH mode (optional):
//    - INTEGRATION_SSH_HOST: SSH host (e.g., root@192.168.1.100)
//    - INTEGRATION_SSH_PASSWORD: SSH password
//    - Plus the Docker/repo/storage vars above
//
// Run with: go test -tags=integration -v ./internal/executor/

func getIntegrationConfig(t *testing.T) (host, password, container, repoPath, storage string) {
	host = os.Getenv("INTEGRATION_SSH_HOST")
	password = os.Getenv("INTEGRATION_SSH_PASSWORD")
	container = os.Getenv("INTEGRATION_DOCKER_CONTAINER")
	repoPath = os.Getenv("INTEGRATION_REPO_PATH")
	storage = os.Getenv("INTEGRATION_STORAGE")

	// At minimum we need a container to test with
	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required")
	}

	return
}

func TestIntegration_DockerContainerExists(t *testing.T) {
	_, _, container, _, _ := getIntegrationConfig(t)

	exec := New(Options{
		Verbose: true,
	})

	// Verify container exists and is running
	err := exec.execute("docker ps --filter name=" + container + " --format '{{.Names}}' | grep -q " + container)
	if err != nil {
		t.Fatalf("Docker container %s not found or not running: %v", container, err)
	}
}

func TestIntegration_DuplicacyBinaryExists(t *testing.T) {
	_, _, container, _, _ := getIntegrationConfig(t)

	exec := New(Options{
		Verbose: true,
	})

	// Verify duplicacy binary exists in container
	err := exec.execute("docker exec " + container + " which duplicacy")
	if err != nil {
		t.Fatalf("duplicacy binary not found in container: %v", err)
	}
}

func TestIntegration_DuplicacyList(t *testing.T) {
	_, _, container, repoPath, storage := getIntegrationConfig(t)

	if repoPath == "" || storage == "" {
		t.Skip("INTEGRATION_REPO_PATH and INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
		Verbose:         true,
	})

	// Run duplicacy list - this is a read-only command
	err := exec.RunDuplicacy("-d", repoPath, "list", "-storage", storage)
	if err != nil {
		t.Fatalf("duplicacy list failed: %v", err)
	}
}

func TestIntegration_DuplicacyBackupAndList(t *testing.T) {
	_, _, container, repoPath, storage := getIntegrationConfig(t)

	if repoPath == "" || storage == "" {
		t.Skip("INTEGRATION_REPO_PATH and INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		DockerContainer: container,
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

func TestIntegration_CommandBuilding_LocalDocker(t *testing.T) {
	_, _, container, _, _ := getIntegrationConfig(t)

	exec := New(Options{
		DockerContainer: container,
		Verbose:         true,
	})

	// Verify command is built correctly for local Docker
	cmd := exec.buildCommand([]string{"list", "-storage", "test"})

	// Should contain docker exec
	if !strings.Contains(cmd, "docker exec "+container) {
		t.Errorf("command should contain docker exec, got: %s", cmd)
	}

	// Should contain duplicacy
	if !strings.Contains(cmd, "duplicacy list -storage test") {
		t.Errorf("command should contain duplicacy list, got: %s", cmd)
	}

	// Should NOT contain SSH (local mode)
	if strings.Contains(cmd, "sshpass") || strings.Contains(cmd, "ssh ") {
		t.Errorf("local mode should not contain SSH, got: %s", cmd)
	}
}

func TestIntegration_CommandBuilding_RemoteSSH(t *testing.T) {
	host, password, container, _, _ := getIntegrationConfig(t)

	if host == "" || password == "" {
		t.Skip("SSH tests require INTEGRATION_SSH_HOST and INTEGRATION_SSH_PASSWORD")
	}

	exec := New(Options{
		SSHHost:         host,
		SSHPassword:     password,
		DockerContainer: container,
		Verbose:         true,
	})

	// Verify command is built correctly for remote SSH
	cmd := exec.buildCommand([]string{"list", "-storage", "test"})

	// Should contain docker exec
	if !strings.Contains(cmd, "docker exec "+container) {
		t.Errorf("command should contain docker exec, got: %s", cmd)
	}

	// Should contain SSH wrapper
	if !strings.Contains(cmd, "sshpass") {
		t.Errorf("command should contain sshpass, got: %s", cmd)
	}

	if !strings.Contains(cmd, "ssh") {
		t.Errorf("command should contain ssh, got: %s", cmd)
	}
}

func TestIntegration_DryRunDoesNotExecute(t *testing.T) {
	_, _, container, _, _ := getIntegrationConfig(t)

	exec := New(Options{
		DockerContainer: container,
		DryRun:          true,
		Verbose:         true,
	})

	// With dry run, this should not actually execute (nonexistent storage)
	err := exec.RunDuplicacy("backup", "-storage", "nonexistent")
	if err != nil {
		t.Errorf("dry run should not return error: %v", err)
	}
}
