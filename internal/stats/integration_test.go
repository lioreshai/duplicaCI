//go:build integration

package stats

import (
	"os"
	"strings"
	"testing"
)

// Integration tests for stats parsing from real duplicacy check output
//
// Run with: go test -tags=integration -v ./internal/stats/
//
// These tests use actual duplicacy check output to verify parsing works
// in real-world scenarios.

func TestIntegration_ParseRealCheckOutput(t *testing.T) {
	// Skip if no test output file provided
	outputFile := os.Getenv("INTEGRATION_CHECK_OUTPUT_FILE")
	if outputFile == "" {
		t.Skip("INTEGRATION_CHECK_OUTPUT_FILE required")
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read check output file: %v", err)
	}

	output := string(data)
	stats, err := ParseCheckOutput(output)
	if err != nil {
		t.Fatalf("ParseCheckOutput failed: %v", err)
	}

	// Verify we got some data
	if stats.Status != "Checked" {
		t.Errorf("Status = %q, want %q", stats.Status, "Checked")
	}

	if len(stats.Repositories) == 0 {
		t.Error("expected at least one repository")
	}

	// Log what we found
	t.Logf("Parsed stats:")
	t.Logf("  Total size: %s", FormatBytes(stats.TotalSize))
	t.Logf("  Total chunks: %d", stats.TotalChunks)
	t.Logf("  Repositories: %d", len(stats.Repositories))
	for name, repo := range stats.Repositories {
		t.Logf("    - %s: %d revisions, %s", name, repo.Revisions, FormatBytes(repo.TotalSize))
	}
}

func TestIntegration_ParseCheckOutputWithTabular(t *testing.T) {
	// This test uses hardcoded sample output from a real duplicacy check -tabular run
	// to ensure the parser handles actual output format correctly

	// Sample from GenizaNAS check
	output := `Repository set to /mnt/moxy_unraid_appDataBackup
Storage set to /mnt/remotes/10.30.88.1_DuplicacyBackups
2025-12-29 01:00:18.421 INFO STORAGE_SET Storage set to /mnt/remotes/10.30.88.1_DuplicacyBackups
2025-12-29 01:02:45.064 INFO SNAPSHOT_CHECK 2 snapshots and 48 revisions
2025-12-29 01:02:45.064 INFO SNAPSHOT_CHECK Total chunk size is 1,211M in 223 chunks
2025-12-29 01:02:45.068 INFO SNAPSHOT_CHECK All chunks referenced by snapshot mikrotik_config_backup at revision 76 exist
2025-12-29 01:02:45.069 INFO SNAPSHOT_CHECK All chunks referenced by snapshot mikrotik_config_backup at revision 77 exist
2025-12-29 01:02:45.070 INFO SNAPSHOT_CHECK All chunks referenced by snapshot unraid_appdata_backup at revision 76 exist
2025-12-29 01:02:45.072 INFO SNAPSHOT_CHECK All chunks referenced by snapshot unraid_appdata_backup at revision 77 exist
2025-12-29 01:02:45.167 INFO SNAPSHOT_CHECK
                  snap | rev |                          | files |  bytes | chunks |    bytes | uniq |    bytes | new |    bytes |
 unraid_appdata_backup |  76 | @ 2025-12-27 01:03       |    84 | 6,519M |    223 |   1,194M |    3 |      20K |  10 |  41,857K |
 unraid_appdata_backup |  77 | @ 2025-12-28 01:03       |    84 | 6,544M |    225 |   1,211M |    0 |        0 |  12 |  74,000K |
 unraid_appdata_backup | all |                          |       |        |    219 |   1,211M |  219 |   1,211M |     |          |

                   snap | rev |                          | files | bytes | chunks |  bytes | uniq |  bytes | new | bytes |
 mikrotik_config_backup |  76 | @ 2025-12-27 01:03       |     8 |  530K |      4 |   375K |    4 |   375K |   4 |  375K |
 mikrotik_config_backup |  77 | @ 2025-12-28 01:03       |     8 |  531K |      4 |   377K |    0 |      0 |   4 |  377K |
 mikrotik_config_backup | all |                          |       |       |      4 |   377K |    4 |   377K |     |       |`

	stats, err := ParseCheckOutput(output)
	if err != nil {
		t.Fatalf("ParseCheckOutput failed: %v", err)
	}

	// Check total chunks from summary line
	expectedChunks := 223
	if stats.TotalChunks != expectedChunks {
		t.Errorf("TotalChunks = %d, want %d", stats.TotalChunks, expectedChunks)
	}

	// Check repository count
	if len(stats.Repositories) != 2 {
		t.Errorf("len(Repositories) = %d, want 2", len(stats.Repositories))
	}

	// Check unraid_appdata_backup
	unraid, ok := stats.Repositories["unraid_appdata_backup"]
	if !ok {
		t.Error("unraid_appdata_backup not found")
	} else {
		if unraid.Revisions != 2 {
			t.Errorf("unraid_appdata_backup.Revisions = %d, want 2", unraid.Revisions)
		}
		if unraid.TotalChunks != 219 {
			t.Errorf("unraid_appdata_backup.TotalChunks = %d, want 219", unraid.TotalChunks)
		}
	}

	// Check mikrotik_config_backup
	mikrotik, ok := stats.Repositories["mikrotik_config_backup"]
	if !ok {
		t.Error("mikrotik_config_backup not found")
	} else {
		if mikrotik.Revisions != 2 {
			t.Errorf("mikrotik_config_backup.Revisions = %d, want 2", mikrotik.Revisions)
		}
		if mikrotik.TotalChunks != 4 {
			t.Errorf("mikrotik_config_backup.TotalChunks = %d, want 4", mikrotik.TotalChunks)
		}
	}

	t.Logf("Successfully parsed: %d repos, %d total chunks, %s total size",
		len(stats.Repositories), stats.TotalChunks, FormatBytes(stats.TotalSize))
}

func TestIntegration_StatsWriterDryRun(t *testing.T) {
	// Test the stats writer in dry-run mode (no actual file writes)
	writer := NewWriter("user@host", "password", "container")
	writer.DryRun = true
	writer.Verbose = true

	dayStats := &DayStats{
		TotalSize:   1211 * 1024 * 1024,
		TotalChunks: 223,
		Status:      "Checked",
		Repositories: map[string]RepoStats{
			"test_backup": {
				Revisions:   10,
				TotalSize:   1000 * 1024 * 1024,
				UniqueSize:  1000 * 1024 * 1024,
				TotalChunks: 200,
			},
		},
	}

	// This should not error in dry-run mode
	err := writer.UpdateStorageStats("TestStorage", dayStats)
	if err != nil {
		t.Errorf("dry-run UpdateStorageStats failed: %v", err)
	}
}

func TestIntegration_StatsWriterWithDocker(t *testing.T) {
	// Integration test that requires Docker container
	container := os.Getenv("INTEGRATION_DOCKER_CONTAINER")
	sshHost := os.Getenv("INTEGRATION_SSH_HOST")
	sshPassword := os.Getenv("INTEGRATION_SSH_PASSWORD")

	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required")
	}

	writer := NewWriter(sshHost, sshPassword, container)
	writer.Verbose = true

	// Read existing stats for a known storage
	stats, err := writer.readStatsFile("/config/stats/storages/GenizaNAS.stats")
	if err != nil {
		t.Logf("Note: could not read existing stats (may not exist): %v", err)
	} else {
		t.Logf("Read existing stats: %d days of data", len(stats))
		for date := range stats {
			t.Logf("  - %s", date)
		}
	}
}

func TestIntegration_StatsWriterFullWorkflow(t *testing.T) {
	// Full integration test: read, update, write stats
	container := os.Getenv("INTEGRATION_DOCKER_CONTAINER")
	sshHost := os.Getenv("INTEGRATION_SSH_HOST")
	sshPassword := os.Getenv("INTEGRATION_SSH_PASSWORD")

	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required")
	}

	writer := NewWriter(sshHost, sshPassword, container)
	writer.Verbose = true

	// Ensure stats directory exists
	mkdirCmd := writer.buildDockerCommand("mkdir -p /config/stats/storages")
	if err := writer.execute(mkdirCmd); err != nil {
		t.Fatalf("failed to create stats directory: %v", err)
	}

	// Create test stats
	testStats := &DayStats{
		TotalSize:       100 * 1024 * 1024,
		TotalChunks:     50,
		PrunedChunks:    5,
		PrunedRevisions: 2,
		Status:          "Checked",
		Repositories: map[string]RepoStats{
			"test_backup": {
				Revisions:   10,
				TotalSize:   80 * 1024 * 1024,
				UniqueSize:  60 * 1024 * 1024,
				TotalChunks: 40,
			},
		},
	}

	// Write stats
	err := writer.UpdateStorageStats("TestCI", testStats)
	if err != nil {
		t.Fatalf("UpdateStorageStats failed: %v", err)
	}

	// Read back and verify
	readStats, err := writer.readStatsFile("/config/stats/storages/TestCI.stats")
	if err != nil {
		t.Fatalf("readStatsFile failed: %v", err)
	}

	today := TodayDate()
	if _, ok := readStats[today]; !ok {
		t.Errorf("expected stats for today (%s), got keys: %v", today, func() []string {
			keys := make([]string, 0, len(readStats))
			for k := range readStats {
				keys = append(keys, k)
			}
			return keys
		}())
	}

	t.Logf("Successfully wrote and read back stats for %s", today)
}

func TestIntegration_ReadStatsFileNonExistent(t *testing.T) {
	container := os.Getenv("INTEGRATION_DOCKER_CONTAINER")
	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required")
	}

	writer := NewWriter("", "", container)
	writer.Verbose = true

	// Reading non-existent file should return empty stats (not error)
	stats, err := writer.readStatsFile("/config/stats/storages/NonExistent12345.stats")
	if err != nil {
		t.Fatalf("readStatsFile should not error for non-existent file: %v", err)
	}

	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %d entries", len(stats))
	}
}

func TestIntegration_ExecuteCapture(t *testing.T) {
	container := os.Getenv("INTEGRATION_DOCKER_CONTAINER")
	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required")
	}

	writer := NewWriter("", "", container)

	// Test executeCapture with a simple command (use double quotes for shell compatibility)
	cmd := writer.buildDockerCommand("echo hello")
	output, err := writer.executeCapture(cmd)
	if err != nil {
		t.Fatalf("executeCapture failed: %v", err)
	}

	if output != "hello" {
		t.Errorf("expected 'hello', got %q", output)
	}
}

func TestIntegration_Execute(t *testing.T) {
	container := os.Getenv("INTEGRATION_DOCKER_CONTAINER")
	if container == "" {
		t.Skip("INTEGRATION_DOCKER_CONTAINER required")
	}

	writer := NewWriter("", "", container)

	// Test execute with a simple command that succeeds
	cmd := writer.buildDockerCommand("echo 'test'")
	err := writer.execute(cmd)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
}

func TestIntegration_ParseOutputWithManyRevisions(t *testing.T) {
	// Test with output containing many revisions to ensure counting works
	output := `2025-12-29 01:02:45.064 INFO SNAPSHOT_CHECK Total chunk size is 100M in 50 chunks
 test_repo |   1 | @ 2025-01-01 |    10 | 10M |  5 |  10M |  5 |  10M |  5 |  10M |
 test_repo |   2 | @ 2025-01-02 |    10 | 10M |  5 |  10M |  5 |  10M |  5 |  10M |
 test_repo |   3 | @ 2025-01-03 |    10 | 10M |  5 |  10M |  5 |  10M |  5 |  10M |
 test_repo |   4 | @ 2025-01-04 |    10 | 10M |  5 |  10M |  5 |  10M |  5 |  10M |
 test_repo |   5 | @ 2025-01-05 |    10 | 10M |  5 |  10M |  5 |  10M |  5 |  10M |
 test_repo | all |              |       |     | 25 | 100M | 25 | 100M |    |      |`

	stats, err := ParseCheckOutput(output)
	if err != nil {
		t.Fatalf("ParseCheckOutput failed: %v", err)
	}

	repo, ok := stats.Repositories["test_repo"]
	if !ok {
		t.Fatal("test_repo not found in repositories")
	}

	// Should count 5 revisions
	if repo.Revisions != 5 {
		t.Errorf("Revisions = %d, want 5", repo.Revisions)
	}

	// Should have 25 chunks from "all" row
	if repo.TotalChunks != 25 {
		t.Errorf("TotalChunks = %d, want 25", repo.TotalChunks)
	}
}

func TestIntegration_ParseOutputWithSpecialChars(t *testing.T) {
	// Test with repository names containing underscores and dashes
	output := `2025-12-29 01:02:45.064 INFO SNAPSHOT_CHECK Total chunk size is 100M in 50 chunks
 my-repo_name |   1 | @ 2025-01-01 |    10 | 10M |  5 |  10M |  5 |  10M |  5 |  10M |
 my-repo_name | all |              |       |     | 25 | 100M | 25 | 100M |    |      |
 another_one  |   1 | @ 2025-01-01 |    10 | 10M |  5 |  10M |  5 |  10M |  5 |  10M |
 another_one  | all |              |       |     | 25 | 100M | 25 | 100M |    |      |`

	stats, err := ParseCheckOutput(output)
	if err != nil {
		t.Fatalf("ParseCheckOutput failed: %v", err)
	}

	if len(stats.Repositories) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(stats.Repositories))
	}

	if _, ok := stats.Repositories["my-repo_name"]; !ok {
		t.Error("my-repo_name not found")
	}

	if _, ok := stats.Repositories["another_one"]; !ok {
		t.Error("another_one not found")
	}
}

func TestIntegration_FormatBytesEdgeCases(t *testing.T) {
	tests := []struct {
		input    int64
		contains string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "B"},
		{1024, "KB"},
		{1024 * 1024, "MB"},
		{1024 * 1024 * 1024, "GB"},
		{1024 * 1024 * 1024 * 1024, "TB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.input)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("FormatBytes(%d) = %q, should contain %q", tt.input, result, tt.contains)
		}
	}
}
