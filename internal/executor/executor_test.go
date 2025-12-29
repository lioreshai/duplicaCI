package executor

import (
	"testing"
)

func TestBuildCommand_Basic(t *testing.T) {
	exec := New(Options{})

	cmd := exec.buildCommand([]string{"backup", "-storage", "gdrive"})
	expected := "duplicacy backup -storage gdrive"

	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestBuildCommand_WithDocker(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
	})

	cmd := exec.buildCommand([]string{"backup", "-storage", "gdrive"})
	expected := "docker exec Duplicacy duplicacy backup -storage gdrive"

	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestBuildCommand_WithSSH(t *testing.T) {
	exec := New(Options{
		SSHHost: "root@192.168.1.100",
	})

	cmd := exec.buildCommand([]string{"backup", "-storage", "gdrive"})
	expected := "ssh -o StrictHostKeyChecking=no -o LogLevel=ERROR root@192.168.1.100 'duplicacy backup -storage gdrive'"

	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestBuildCommand_WithSSHAndPassword(t *testing.T) {
	exec := New(Options{
		SSHHost:     "root@192.168.1.100",
		SSHPassword: "secret123",
	})

	cmd := exec.buildCommand([]string{"backup"})
	expected := "sshpass -p 'secret123' ssh -o StrictHostKeyChecking=no -o LogLevel=ERROR root@192.168.1.100 'duplicacy backup'"

	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestBuildCommand_WithDockerAndSSH(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
		SSHHost:         "root@192.168.1.100",
		SSHPassword:     "secret123",
	})

	cmd := exec.buildCommand([]string{"backup", "-storage", "gdrive"})
	expected := "sshpass -p 'secret123' ssh -o StrictHostKeyChecking=no -o LogLevel=ERROR root@192.168.1.100 'docker exec Duplicacy duplicacy backup -storage gdrive'"

	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestBuildCommand_EscapesSingleQuotes(t *testing.T) {
	exec := New(Options{
		SSHHost:     "root@192.168.1.100",
		SSHPassword: "pass'word",
	})

	cmd := exec.buildCommand([]string{"backup"})

	// Password with single quote should be escaped
	if cmd == "" {
		t.Error("command should not be empty")
	}

	// Verify the password is escaped
	expectedPasswordPart := "'pass'\"'\"'word'"
	if !contains(cmd, expectedPasswordPart) {
		t.Errorf("expected password to be escaped, got %q", cmd)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
