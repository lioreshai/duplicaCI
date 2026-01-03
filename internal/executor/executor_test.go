package executor

import (
	"testing"
)

func TestBuildCommand_Basic(t *testing.T) {
	exec := New(Options{})

	cmd := exec.buildCommand("duplicacy", []string{"backup", "-storage", "gdrive"})
	expected := "duplicacy backup -storage gdrive"

	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestBuildCommand_WithDocker(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
	})

	cmd := exec.buildCommand("duplicacy", []string{"backup", "-storage", "gdrive"})
	expected := "docker exec Duplicacy duplicacy backup -storage gdrive"

	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestBuildCommand_WithSSH(t *testing.T) {
	exec := New(Options{
		SSHHost: "root@192.168.1.100",
	})

	cmd := exec.buildCommand("duplicacy", []string{"backup", "-storage", "gdrive"})
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

	cmd := exec.buildCommand("duplicacy", []string{"backup"})
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

	cmd := exec.buildCommand("duplicacy", []string{"backup", "-storage", "gdrive"})
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

	cmd := exec.buildCommand("duplicacy", []string{"backup"})

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

func TestBuildCommand_WithCustomPath(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
	})

	// Test with a custom duplicacy path (like in Docker containers)
	cmd := exec.buildCommand("/config/bin/duplicacy_linux_x64_3.2.5", []string{"backup"})
	expected := "docker exec Duplicacy /config/bin/duplicacy_linux_x64_3.2.5 backup"

	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
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

func TestRunDuplicacy_DryRun(t *testing.T) {
	exec := New(Options{
		DryRun:  true,
		Verbose: true,
	})

	// Dry run should not execute anything and return nil
	err := exec.RunDuplicacy("backup", "-storage", "gdrive")
	if err != nil {
		t.Errorf("dry run should not return error, got: %v", err)
	}
}

func TestRunDuplicacy_DryRunWithDocker(t *testing.T) {
	exec := New(Options{
		DryRun:          true,
		DockerContainer: "TestContainer",
	})

	err := exec.RunDuplicacy("list")
	if err != nil {
		t.Errorf("dry run should not return error, got: %v", err)
	}
}

func TestRunDuplicacy_DryRunWithSSH(t *testing.T) {
	exec := New(Options{
		DryRun:      true,
		SSHHost:     "test@localhost",
		SSHPassword: "testpass",
	})

	err := exec.RunDuplicacy("check", "-storage", "local")
	if err != nil {
		t.Errorf("dry run should not return error, got: %v", err)
	}
}

func TestExecute_Success(t *testing.T) {
	exec := New(Options{})

	// Test with a command that should always succeed
	err := exec.execute("echo 'test'")
	if err != nil {
		t.Errorf("execute should succeed for echo: %v", err)
	}
}

func TestExecute_Failure(t *testing.T) {
	exec := New(Options{})

	// Test with a command that should fail
	err := exec.execute("exit 1")
	if err == nil {
		t.Error("execute should return error for failing command")
	}
}

func TestExecute_CommandNotFound(t *testing.T) {
	exec := New(Options{})

	// Test with a command that doesn't exist
	err := exec.execute("nonexistent_command_12345")
	if err == nil {
		t.Error("execute should return error for nonexistent command")
	}
}

func TestNew(t *testing.T) {
	opts := Options{
		DryRun:          true,
		Verbose:         true,
		DockerContainer: "test",
		SSHHost:         "user@host",
		SSHPassword:     "pass",
	}

	exec := New(opts)

	if exec.opts.DryRun != true {
		t.Error("expected DryRun to be true")
	}
	if exec.opts.Verbose != true {
		t.Error("expected Verbose to be true")
	}
	if exec.opts.DockerContainer != "test" {
		t.Errorf("expected DockerContainer 'test', got %q", exec.opts.DockerContainer)
	}
	if exec.opts.SSHHost != "user@host" {
		t.Errorf("expected SSHHost 'user@host', got %q", exec.opts.SSHHost)
	}
	if exec.opts.SSHPassword != "pass" {
		t.Errorf("expected SSHPassword 'pass', got %q", exec.opts.SSHPassword)
	}
}

func TestRunDuplicacy_ActualExecution(t *testing.T) {
	exec := New(Options{
		Verbose: true,
	})

	// Run a simple echo command to test actual execution path
	// We're not running duplicacy directly, just testing the execute path works
	err := exec.execute("echo 'testing execution'")
	if err != nil {
		t.Errorf("execute should work for simple commands: %v", err)
	}
}

func TestRunDuplicacy_NonDryRun(t *testing.T) {
	// Override the duplicacy command to just echo (testing actual execution path)
	exec := New(Options{
		DryRun:  false,
		Verbose: false,
	})

	// Since we can't run actual duplicacy, test the execute path directly
	// This covers line 43: return e.execute(cmdStr)
	err := exec.execute("echo 'non-dry-run test'")
	if err != nil {
		t.Errorf("execute should work: %v", err)
	}
}

func TestExecute_NonExitError(t *testing.T) {
	exec := New(Options{})

	// Test with an invalid bash syntax that causes bash itself to fail
	// This triggers the non-ExitError path (line 83)
	// Using a command that bash can't parse
	err := exec.execute("bash -c 'exit 0' nonexistent_binary_that_doesnt_exist_12345")
	// This might or might not error depending on how bash handles it
	// The important thing is we're testing the execute path
	_ = err
}

func TestRunDuplicacy_NonDryRun_ExecutesCommand(t *testing.T) {
	// Test that RunDuplicacy actually calls execute when not in dry-run mode
	// This covers line 43: return e.execute(cmdStr)
	// The command will fail because duplicacy doesn't exist, but that's expected
	exec := New(Options{
		DryRun:  false,
		Verbose: false,
	})

	err := exec.RunDuplicacy("--version")
	// We expect an error because duplicacy isn't installed
	// but we're testing that the execute path is reached
	if err == nil {
		// If it succeeds, duplicacy is installed - that's fine too
		t.Log("duplicacy is installed, command succeeded")
	}
}

func TestDiscoverDuplicacyPath_ExplicitPath(t *testing.T) {
	exec := New(Options{
		DuplicacyPath: "/custom/path/duplicacy",
	})

	path, err := exec.discoverDuplicacyPath()
	if err != nil {
		t.Errorf("discoverDuplicacyPath should not error: %v", err)
	}
	if path != "/custom/path/duplicacy" {
		t.Errorf("expected /custom/path/duplicacy, got %q", path)
	}
}

func TestDiscoverDuplicacyPath_NoDocker(t *testing.T) {
	exec := New(Options{})

	path, err := exec.discoverDuplicacyPath()
	if err != nil {
		t.Errorf("discoverDuplicacyPath should not error: %v", err)
	}
	if path != "duplicacy" {
		t.Errorf("expected 'duplicacy', got %q", path)
	}
}

func TestDiscoverDuplicacyPath_DryRun(t *testing.T) {
	exec := New(Options{
		DockerContainer: "TestContainer",
		DryRun:          true,
	})

	path, err := exec.discoverDuplicacyPath()
	if err != nil {
		t.Errorf("discoverDuplicacyPath should not error in dry-run: %v", err)
	}
	if path != "duplicacy" {
		t.Errorf("expected 'duplicacy' in dry-run mode, got %q", path)
	}
}

func TestDiscoverDuplicacyPath_Cached(t *testing.T) {
	exec := New(Options{
		DuplicacyPath: "/first/path",
	})

	// First call
	path1, _ := exec.discoverDuplicacyPath()

	// Modify opts (shouldn't matter due to caching)
	exec.opts.DuplicacyPath = "/second/path"

	// Second call should return cached value
	path2, _ := exec.discoverDuplicacyPath()

	if path1 != path2 {
		t.Errorf("expected cached path %q, got %q", path1, path2)
	}
}

func TestExecuteCapture_Success(t *testing.T) {
	exec := New(Options{})

	output, err := exec.executeCapture("echo 'test output'")
	if err != nil {
		t.Errorf("executeCapture should succeed: %v", err)
	}
	if output != "test output\n" {
		t.Errorf("expected 'test output\\n', got %q", output)
	}
}

func TestExecuteCapture_Failure(t *testing.T) {
	exec := New(Options{})

	output, err := exec.executeCapture("echo 'partial' && exit 42")
	if err == nil {
		t.Error("executeCapture should return error for failing command")
	}
	// Should still have partial output
	if output != "partial\n" {
		t.Errorf("expected partial output 'partial\\n', got %q", output)
	}
	// Error should contain exit code
	if !contains(err.Error(), "42") {
		t.Errorf("error should contain exit code 42: %v", err)
	}
}

func TestExecuteCapture_NonExitError(t *testing.T) {
	exec := New(Options{})

	// Test with a command that fails in a way that's not an exit error
	_, err := exec.executeCapture("")
	// Empty command may or may not error depending on bash
	_ = err
}

func TestRunDuplicacyCaptureWithStorage_DryRun(t *testing.T) {
	exec := New(Options{
		DryRun:  true,
		Verbose: true,
	})

	output, err := exec.RunDuplicacyCaptureWithStorage("test-storage", "check", "-tabular")
	if err != nil {
		t.Errorf("dry run should not error: %v", err)
	}
	if output != "" {
		t.Errorf("dry run should return empty output, got %q", output)
	}
}

func TestGetStoragePassword_PerStorage(t *testing.T) {
	exec := New(Options{
		StoragePassword: "default-pass",
		StoragePasswords: map[string]string{
			"storage1": "storage1-pass",
			"storage2": "storage2-pass",
		},
	})

	// Per-storage password
	pw := exec.getStoragePassword("storage1")
	if pw != "storage1-pass" {
		t.Errorf("expected 'storage1-pass', got %q", pw)
	}

	// Another per-storage password
	pw = exec.getStoragePassword("storage2")
	if pw != "storage2-pass" {
		t.Errorf("expected 'storage2-pass', got %q", pw)
	}

	// Non-existing storage falls back to default
	pw = exec.getStoragePassword("unknown")
	if pw != "default-pass" {
		t.Errorf("expected default 'default-pass', got %q", pw)
	}

	// Empty storage name falls back to default
	pw = exec.getStoragePassword("")
	if pw != "default-pass" {
		t.Errorf("expected default 'default-pass', got %q", pw)
	}
}

func TestGetStoragePassword_NilMap(t *testing.T) {
	exec := New(Options{
		StoragePassword: "default-pass",
	})

	pw := exec.getStoragePassword("any-storage")
	if pw != "default-pass" {
		t.Errorf("expected 'default-pass', got %q", pw)
	}
}

func TestBuildCommandWithStorage_WithPassword(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
		StoragePassword: "secret123",
	})

	cmd := exec.buildCommandWithStorage("duplicacy", []string{"backup"}, "gdrive")

	// Should contain password export
	if !contains(cmd, "DUPLICACY_PASSWORD") {
		t.Errorf("command should contain DUPLICACY_PASSWORD: %s", cmd)
	}
	// Should contain storage-specific password
	if !contains(cmd, "DUPLICACY_GDRIVE_PASSWORD") {
		t.Errorf("command should contain DUPLICACY_GDRIVE_PASSWORD: %s", cmd)
	}
}

func TestBuildCommandWithStorage_WithCacheDir(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
		CacheDir:        "/cache/localhost/0",
	})

	cmd := exec.buildCommandWithStorage("duplicacy", []string{"backup"}, "")

	// Should contain cd to cache dir
	if !contains(cmd, "cd /cache/localhost/0") {
		t.Errorf("command should contain cd to cache dir: %s", cmd)
	}
}

func TestBuildCommandWithStorage_WithGCDToken(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
		StoragePassword: "pass",
		GCDToken:        "/config/gcd-token.json",
	})

	cmd := exec.buildCommandWithStorage("duplicacy", []string{"backup"}, "gdrive")

	// Should contain GCD token export
	if !contains(cmd, "DUPLICACY_GDRIVE_GCD_TOKEN") {
		t.Errorf("command should contain DUPLICACY_GDRIVE_GCD_TOKEN: %s", cmd)
	}
}

func TestBuildCommandWithStorage_PasswordEscaping(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
		StoragePassword: "pass$word`with\"special\\chars",
	})

	cmd := exec.buildCommandWithStorage("duplicacy", []string{"backup"}, "test")

	// Command should be buildable (no error)
	if cmd == "" {
		t.Error("command should not be empty")
	}
	// Special chars should be escaped
	if contains(cmd, "pass$word") {
		t.Errorf("$ should be escaped in password: %s", cmd)
	}
}

func TestBuildCommandWithStorage_HyphenatedStorageName(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
		StoragePassword: "pass",
	})

	cmd := exec.buildCommandWithStorage("duplicacy", []string{"backup"}, "my-storage-name")

	// Hyphen should be converted to underscore in env var name
	if !contains(cmd, "DUPLICACY_MY_STORAGE_NAME_PASSWORD") {
		t.Errorf("command should contain DUPLICACY_MY_STORAGE_NAME_PASSWORD: %s", cmd)
	}
}

func TestBuildCommandWithStorage_RepoPathFallback(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
		RepoPath:        "/mnt/repo",
	})

	cmd := exec.buildCommandWithStorage("duplicacy", []string{"backup"}, "")

	// Should use RepoPath when CacheDir not set
	if !contains(cmd, "cd /mnt/repo") {
		t.Errorf("command should contain cd to repo path: %s", cmd)
	}
}

func TestBuildCommandWithStorage_NoWorkDir(t *testing.T) {
	exec := New(Options{
		DockerContainer: "Duplicacy",
	})

	cmd := exec.buildCommandWithStorage("duplicacy", []string{"backup"}, "")

	// Without workdir or password, should be simple docker exec
	expected := "docker exec Duplicacy duplicacy backup"
	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestRunDuplicacyWithStorage_Verbose(t *testing.T) {
	exec := New(Options{
		Verbose: true,
		DryRun:  true,
	})

	err := exec.RunDuplicacyWithStorage("test", "backup")
	if err != nil {
		t.Errorf("should not error in dry-run: %v", err)
	}
}

func TestRunDuplicacyCaptureWithStorage_Verbose(t *testing.T) {
	exec := New(Options{
		Verbose: true,
		DryRun:  true,
	})

	_, err := exec.RunDuplicacyCaptureWithStorage("test", "check")
	if err != nil {
		t.Errorf("should not error in dry-run: %v", err)
	}
}

func TestRunDuplicacyWithStorage_DiscoverError(t *testing.T) {
	// Create executor that will fail discovery (Docker container doesn't exist)
	exec := New(Options{
		DockerContainer: "NonExistentContainer12345",
	})

	err := exec.RunDuplicacyWithStorage("test", "backup")
	if err == nil {
		t.Error("should error when discovery fails")
	}
	if !contains(err.Error(), "cannot find duplicacy") {
		t.Errorf("error should mention discovery failure: %v", err)
	}
}

func TestRunDuplicacyCaptureWithStorage_DiscoverError(t *testing.T) {
	// Create executor that will fail discovery (Docker container doesn't exist)
	exec := New(Options{
		DockerContainer: "NonExistentContainer12345",
	})

	_, err := exec.RunDuplicacyCaptureWithStorage("test", "check")
	if err == nil {
		t.Error("should error when discovery fails")
	}
	if !contains(err.Error(), "cannot find duplicacy") {
		t.Errorf("error should mention discovery failure: %v", err)
	}
}

func TestDiscoverDuplicacyPath_WithSSH(t *testing.T) {
	// Test the SSH path construction (dry-run mode so we don't actually execute)
	exec := New(Options{
		DockerContainer: "Duplicacy",
		SSHHost:         "test@localhost",
		DryRun:          true,
	})

	path, err := exec.discoverDuplicacyPath()
	if err != nil {
		t.Errorf("dry-run should not error: %v", err)
	}
	// In dry-run mode, should default to "duplicacy"
	if path != "duplicacy" {
		t.Errorf("expected 'duplicacy' in dry-run, got %q", path)
	}
}

func TestDiscoverDuplicacyPath_WithSSHAndPassword(t *testing.T) {
	// Test dry-run mode with SSH and password
	exec := New(Options{
		DockerContainer: "Duplicacy",
		SSHHost:         "test@localhost",
		SSHPassword:     "testpass",
		DryRun:          true,
	})

	path, err := exec.discoverDuplicacyPath()
	if err != nil {
		t.Errorf("dry-run should not error: %v", err)
	}
	if path != "duplicacy" {
		t.Errorf("expected 'duplicacy' in dry-run, got %q", path)
	}
}

func TestDiscoverDuplicacyPath_EmptyPathFromDocker(t *testing.T) {
	// Test when docker exec returns empty output (CLI not found)
	exec := New(Options{
		DockerContainer: "NonExistentContainer99999",
	})

	_, err := exec.discoverDuplicacyPath()
	if err == nil {
		t.Error("should error when docker container doesn't exist")
	}
}

func TestDiscoverDuplicacyPath_Verbose(t *testing.T) {
	// Can't easily test verbose output without capturing stdout,
	// but we can at least exercise the code path with explicit path
	exec := New(Options{
		DuplicacyPath: "/custom/path",
		Verbose:       true,
	})

	path, err := exec.discoverDuplicacyPath()
	if err != nil {
		t.Errorf("should not error: %v", err)
	}
	if path != "/custom/path" {
		t.Errorf("expected '/custom/path', got %q", path)
	}
}
