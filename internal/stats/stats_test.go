package stats

import (
	"testing"
)

func TestParseCheckOutput(t *testing.T) {
	// Sample output from duplicacy check -tabular
	output := `Repository set to /mnt/moxy_unraid_appDataBackup
Storage set to /mnt/remotes/10.30.88.1_DuplicacyBackups
2025-12-29 01:00:18.421 INFO STORAGE_SET Storage set to gcd://My Drive@moxy-duplicacy-backup
2025-12-29 01:00:19.894 INFO SNAPSHOT_CHECK Listing all chunks
2025-12-29 01:02:45.064 INFO SNAPSHOT_CHECK 2 snapshots and 48 revisions
2025-12-29 01:02:45.064 INFO SNAPSHOT_CHECK Total chunk size is 4,617M in 975 chunks
2025-12-29 01:02:45.068 INFO SNAPSHOT_CHECK All chunks referenced by snapshot mikrotik_config_backup at revision 1 exist
2025-12-29 01:02:45.069 INFO SNAPSHOT_CHECK All chunks referenced by snapshot mikrotik_config_backup at revision 8 exist
2025-12-29 01:02:45.070 INFO SNAPSHOT_CHECK All chunks referenced by snapshot unraid_appdata_backup at revision 1 exist
2025-12-29 01:02:45.072 INFO SNAPSHOT_CHECK All chunks referenced by snapshot unraid_appdata_backup at revision 8 exist
2025-12-29 01:02:45.167 INFO SNAPSHOT_CHECK
                  snap | rev |                          | files |  bytes | chunks |    bytes | uniq |    bytes | new |    bytes |
 unraid_appdata_backup |   1 | @ 2025-10-13 20:34 -hash |    28 | 3,384M |    195 | 991,477K |   32 | 164,900K | 195 | 991,477K |
 unraid_appdata_backup |   8 | @ 2025-10-20 01:01       |    56 | 5,926M |    197 |   1,041M |   32 | 228,619K |  34 | 240,165K |
 unraid_appdata_backup | all |                          |       |        |    883 |   4,608M |  883 |   4,608M |     |          |

                   snap | rev |                          | files | bytes | chunks |  bytes | uniq |  bytes | new | bytes |
 mikrotik_config_backup |   1 | @ 2025-10-13 20:36 -hash |     9 |  826K |      4 |   672K |    4 |   672K |   4 |  672K |
 mikrotik_config_backup |   8 | @ 2025-10-20 01:01       |     8 |  532K |      4 |   377K |    4 |   377K |   4 |  377K |
 mikrotik_config_backup | all |                          |       |       |     92 | 8,853K |   92 | 8,853K |     |       |`

	stats, err := ParseCheckOutput(output)
	if err != nil {
		t.Fatalf("ParseCheckOutput failed: %v", err)
	}

	// Check total size and chunks
	expectedTotalSize := int64(4617 * 1024 * 1024) // 4,617M
	if stats.TotalSize != expectedTotalSize {
		t.Errorf("TotalSize = %d, want %d", stats.TotalSize, expectedTotalSize)
	}

	if stats.TotalChunks != 975 {
		t.Errorf("TotalChunks = %d, want 975", stats.TotalChunks)
	}

	if stats.Status != "Checked" {
		t.Errorf("Status = %q, want %q", stats.Status, "Checked")
	}

	// Check repositories
	if len(stats.Repositories) != 2 {
		t.Errorf("len(Repositories) = %d, want 2", len(stats.Repositories))
	}

	// Check unraid_appdata_backup
	unraid, ok := stats.Repositories["unraid_appdata_backup"]
	if !ok {
		t.Error("unraid_appdata_backup not found in repositories")
	} else {
		if unraid.Revisions != 2 {
			t.Errorf("unraid_appdata_backup.Revisions = %d, want 2", unraid.Revisions)
		}
		if unraid.TotalChunks != 883 {
			t.Errorf("unraid_appdata_backup.TotalChunks = %d, want 883", unraid.TotalChunks)
		}
	}

	// Check mikrotik_config_backup
	mikrotik, ok := stats.Repositories["mikrotik_config_backup"]
	if !ok {
		t.Error("mikrotik_config_backup not found in repositories")
	} else {
		if mikrotik.Revisions != 2 {
			t.Errorf("mikrotik_config_backup.Revisions = %d, want 2", mikrotik.Revisions)
		}
		if mikrotik.TotalChunks != 92 {
			t.Errorf("mikrotik_config_backup.TotalChunks = %d, want 92", mikrotik.TotalChunks)
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"100", 100},
		{"1,000", 1000},
		{"1K", 1024},
		{"1M", 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"1T", 1024 * 1024 * 1024 * 1024},
		{"4,617M", 4617 * 1024 * 1024},
		{"8,853K", 8853 * 1024},
		{"991,477K", 991477 * 1024},
		{"", 0},
	}

	for _, tt := range tests {
		got, err := parseSize(tt.input)
		if err != nil {
			t.Errorf("parseSize(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("parseSize(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestParseNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"100", 100},
		{"1,000", 1000},
		{"975", 975},
		{"10,000,000", 10000000},
		{"", 0},
	}

	for _, tt := range tests {
		got, err := parseNumber(tt.input)
		if err != nil {
			t.Errorf("parseNumber(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("parseNumber(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.expected {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseCheckOutput_NoRepos(t *testing.T) {
	output := `Repository set to /mnt/test
Storage set to /test
2025-12-29 01:00:18.421 INFO STORAGE_SET Storage set to /test`

	_, err := ParseCheckOutput(output)
	if err == nil {
		t.Error("ParseCheckOutput should return error when no repositories found")
	}
}

func TestTodayDate(t *testing.T) {
	date := TodayDate()
	// Should be in YYYY-MM-DD format
	if len(date) != 10 {
		t.Errorf("TodayDate() = %q, expected 10 chars (YYYY-MM-DD)", date)
	}
	if date[4] != '-' || date[7] != '-' {
		t.Errorf("TodayDate() = %q, expected YYYY-MM-DD format", date)
	}
}

func TestNewWriter(t *testing.T) {
	w := NewWriter("root@host", "password", "Duplicacy")

	if w.SSHHost != "root@host" {
		t.Errorf("SSHHost = %q, want %q", w.SSHHost, "root@host")
	}
	if w.SSHPassword != "password" {
		t.Errorf("SSHPassword = %q, want %q", w.SSHPassword, "password")
	}
	if w.DockerContainer != "Duplicacy" {
		t.Errorf("DockerContainer = %q, want %q", w.DockerContainer, "Duplicacy")
	}
	if w.StatsPath != "/config/stats/storages" {
		t.Errorf("StatsPath = %q, want %q", w.StatsPath, "/config/stats/storages")
	}
}

func TestBuildDockerCommand_NoSSH(t *testing.T) {
	w := &Writer{
		DockerContainer: "Duplicacy",
	}

	cmd := w.buildDockerCommand("cat /config/test.txt")
	expected := "docker exec Duplicacy sh -c 'cat /config/test.txt'"
	if cmd != expected {
		t.Errorf("buildDockerCommand() = %q, want %q", cmd, expected)
	}
}

func TestBuildDockerCommand_WithSSH(t *testing.T) {
	w := &Writer{
		DockerContainer: "Duplicacy",
		SSHHost:         "root@192.168.1.100",
	}

	cmd := w.buildDockerCommand("cat /config/test.txt")
	// Should wrap in ssh
	if !contains(cmd, "ssh -o StrictHostKeyChecking=no") {
		t.Errorf("buildDockerCommand() should contain ssh options: %s", cmd)
	}
	if !contains(cmd, "root@192.168.1.100") {
		t.Errorf("buildDockerCommand() should contain host: %s", cmd)
	}
	if !contains(cmd, "docker exec Duplicacy") {
		t.Errorf("buildDockerCommand() should contain docker exec: %s", cmd)
	}
}

func TestBuildDockerCommand_WithSSHAndPassword(t *testing.T) {
	w := &Writer{
		DockerContainer: "Duplicacy",
		SSHHost:         "root@192.168.1.100",
		SSHPassword:     "secret123",
	}

	cmd := w.buildDockerCommand("cat /config/test.txt")
	// Should wrap in sshpass
	if !contains(cmd, "sshpass -p") {
		t.Errorf("buildDockerCommand() should contain sshpass: %s", cmd)
	}
	if !contains(cmd, "secret123") {
		t.Errorf("buildDockerCommand() should contain password: %s", cmd)
	}
}

func TestWriteStatsFile_DryRun(t *testing.T) {
	w := &Writer{
		DockerContainer: "Duplicacy",
		DryRun:          true,
	}

	stats := StorageStats{
		"2025-01-01": &DayStats{
			TotalSize:   1000,
			TotalChunks: 10,
			Status:      "Checked",
		},
	}

	// Should not error in dry-run mode
	err := w.writeStatsFile("/config/stats/test.stats", stats)
	if err != nil {
		t.Errorf("writeStatsFile() in dry-run should not error: %v", err)
	}
}

func TestParseSize_InvalidNumber(t *testing.T) {
	// Test with non-numeric input (X is not a recognized suffix, so it tries to parse "100X" as a number)
	_, err := parseSize("abc")
	if err == nil {
		t.Error("parseSize should error on non-numeric input")
	}

	// Also test with a string that contains an unrecognized suffix
	_, err = parseSize("100X")
	if err == nil {
		t.Error("parseSize should error when parsing fails")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
