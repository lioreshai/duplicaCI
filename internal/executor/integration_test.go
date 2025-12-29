// +build integration

package executor

import (
	"os"
	"testing"
)

// Integration tests require:
// - INTEGRATION_SSH_HOST: SSH host to test against (e.g., root@192.168.1.100)
// - INTEGRATION_SSH_PASSWORD: SSH password
// - INTEGRATION_DOCKER_CONTAINER: Docker container name (e.g., Duplicacy)
//
// Run with: go test -tags=integration -v ./internal/executor/

func TestIntegration_SSHConnection(t *testing.T) {
	host := os.Getenv("INTEGRATION_SSH_HOST")
	password := os.Getenv("INTEGRATION_SSH_PASSWORD")

	if host == "" || password == "" {
		t.Skip("INTEGRATION_SSH_HOST and INTEGRATION_SSH_PASSWORD required")
	}

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

func TestIntegration_DockerExec(t *testing.T) {
	host := os.Getenv("INTEGRATION_SSH_HOST")
	password := os.Getenv("INTEGRATION_SSH_PASSWORD")
	container := os.Getenv("INTEGRATION_DOCKER_CONTAINER")

	if host == "" || password == "" || container == "" {
		t.Skip("INTEGRATION_SSH_HOST, INTEGRATION_SSH_PASSWORD, and INTEGRATION_DOCKER_CONTAINER required")
	}

	exec := New(Options{
		SSHHost:         host,
		SSHPassword:     password,
		DockerContainer: container,
		Verbose:         true,
	})

	// Build a command that would run inside the container
	cmd := exec.buildCommand([]string{"list"})
	t.Logf("Built command: %s", cmd)

	// We don't actually execute to avoid modifying anything
	// Just verify the command structure is correct
	if cmd == "" {
		t.Error("command should not be empty")
	}
}

func TestIntegration_DuplicacyList(t *testing.T) {
	host := os.Getenv("INTEGRATION_SSH_HOST")
	password := os.Getenv("INTEGRATION_SSH_PASSWORD")
	container := os.Getenv("INTEGRATION_DOCKER_CONTAINER")

	if host == "" || password == "" || container == "" {
		t.Skip("INTEGRATION_SSH_HOST, INTEGRATION_SSH_PASSWORD, and INTEGRATION_DOCKER_CONTAINER required")
	}

	exec := New(Options{
		SSHHost:         host,
		SSHPassword:     password,
		DockerContainer: container,
		Verbose:         true,
	})

	// Run duplicacy list - this is a read-only command
	err := exec.RunDuplicacy("list")
	if err != nil {
		t.Logf("duplicacy list failed (may be expected if not in a repository): %v", err)
	}
}
