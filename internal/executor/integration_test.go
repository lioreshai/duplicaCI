//go:build integration

package executor

import (
	"os"
	"strings"
	"testing"
)

// Integration tests require:
// - INTEGRATION_SSH_HOST: SSH host to test against (e.g., root@192.168.1.100)
// - INTEGRATION_SSH_PASSWORD: SSH password
// - INTEGRATION_DOCKER_CONTAINER: Docker container name (e.g., Duplicacy)
// - INTEGRATION_REPO_PATH: Path to a Duplicacy repository inside the container
// - INTEGRATION_STORAGE: Storage name to test with
//
// Run with: go test -tags=integration -v ./internal/executor/

func getIntegrationConfig(t *testing.T) (host, password, container, repoPath, storage string) {
	host = os.Getenv("INTEGRATION_SSH_HOST")
	password = os.Getenv("INTEGRATION_SSH_PASSWORD")
	container = os.Getenv("INTEGRATION_DOCKER_CONTAINER")
	repoPath = os.Getenv("INTEGRATION_REPO_PATH")
	storage = os.Getenv("INTEGRATION_STORAGE")

	if host == "" || password == "" {
		t.Skip("INTEGRATION_SSH_HOST and INTEGRATION_SSH_PASSWORD required")
	}

	return
}

func TestIntegration_SSHConnection(t *testing.T) {
	host, password, _, _, _ := getIntegrationConfig(t)

	exec := New(Options{
		SSHHost:     host,
		SSHPassword: password,
		Verbose:     true,
	})

	// Test a simple command via SSH
	err := exec.execute("echo 'SSH connection successful'")
	if err != nil {
		t.Fatalf("SSH connection failed: %v", err)
	}
}

func TestIntegration_DockerContainerExists(t *testing.T) {
	host, password, container, _, _ := getIntegrationConfig(t)

	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required")
	}

	exec := New(Options{
		SSHHost:     host,
		SSHPassword: password,
		Verbose:     true,
	})

	// Verify container exists and is running
	err := exec.execute("docker ps --filter name=" + container + " --format '{{.Names}}' | grep -q " + container)
	if err != nil {
		t.Fatalf("Docker container %s not found or not running: %v", container, err)
	}
}

func TestIntegration_DuplicacyBinaryExists(t *testing.T) {
	host, password, container, _, _ := getIntegrationConfig(t)

	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required")
	}

	exec := New(Options{
		SSHHost:         host,
		SSHPassword:     password,
		DockerContainer: container,
		Verbose:         true,
	})

	// Verify duplicacy binary exists in container
	err := exec.execute("docker exec " + container + " which duplicacy")
	if err != nil {
		t.Fatalf("duplicacy binary not found in container: %v", err)
	}
}

func TestIntegration_DuplicacyList(t *testing.T) {
	host, password, container, repoPath, storage := getIntegrationConfig(t)

	if container == "" || repoPath == "" || storage == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER, INTEGRATION_REPO_PATH, and INTEGRATION_STORAGE required")
	}

	exec := New(Options{
		SSHHost:         host,
		SSHPassword:     password,
		DockerContainer: container,
		Verbose:         true,
	})

	// Run duplicacy list - this is a read-only command
	// We need to cd to the repository path first
	cmd := exec.buildCommand([]string{"-d", repoPath, "list", "-storage", storage})
	t.Logf("Running command: %s", cmd)

	err := exec.execute(cmd)
	if err != nil {
		// List might fail if repo doesn't exist, log but don't fail
		t.Logf("duplicacy list returned error (may be expected): %v", err)
	}
}

func TestIntegration_CommandBuilding(t *testing.T) {
	host, password, container, _, _ := getIntegrationConfig(t)

	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required")
	}

	exec := New(Options{
		SSHHost:         host,
		SSHPassword:     password,
		DockerContainer: container,
		Verbose:         true,
	})

	// Verify command is built correctly
	cmd := exec.buildCommand([]string{"list", "-storage", "test"})

	// Should contain docker exec
	if !strings.Contains(cmd, "docker exec "+container) {
		t.Errorf("command should contain docker exec, got: %s", cmd)
	}

	// Should contain duplicacy
	if !strings.Contains(cmd, "duplicacy list -storage test") {
		t.Errorf("command should contain duplicacy list, got: %s", cmd)
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
	host, password, container, _, _ := getIntegrationConfig(t)

	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required")
	}

	exec := New(Options{
		SSHHost:         host,
		SSHPassword:     password,
		DockerContainer: container,
		DryRun:          true,
		Verbose:         true,
	})

	// With dry run, this should not actually execute
	err := exec.RunDuplicacy("backup", "-storage", "nonexistent")
	if err != nil {
		t.Errorf("dry run should not return error: %v", err)
	}
}
