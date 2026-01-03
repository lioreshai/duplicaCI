package stats

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// StorageStats represents the stats file structure (date -> stats)
type StorageStats map[string]*DayStats

// DayStats represents statistics for a single day
type DayStats struct {
	TotalSize       int64                `json:"total-size"`
	TotalChunks     int                  `json:"total-chunks"`
	PrunedChunks    int                  `json:"pruned-chunks"`
	PrunedRevisions int                  `json:"pruned-revisions"`
	Status          string               `json:"status"`
	Repositories    map[string]RepoStats `json:"repositories"`
}

// RepoStats represents statistics for a single repository
type RepoStats struct {
	Revisions   int   `json:"revisions"`
	TotalSize   int64 `json:"total-size"`
	UniqueSize  int64 `json:"unique-size"`
	TotalChunks int   `json:"total-chunks"`
}

// ParseCheckOutput parses duplicacy check -tabular output and returns DayStats
func ParseCheckOutput(output string) (*DayStats, error) {
	stats := &DayStats{
		Status:       "Checked",
		Repositories: make(map[string]RepoStats),
	}

	lines := strings.Split(output, "\n")

	// Parse total chunks line: "INFO SNAPSHOT_CHECK Total chunk size is 4,617M in 975 chunks"
	totalChunksRe := regexp.MustCompile(`Total chunk size is ([\d,]+[KMGT]?) in ([\d,]+) chunks`)

	// Parse tabular "all" rows for each repository
	// Format: " repo_name | all |    |     |      | chunks |    bytes | uniq |    bytes | new | bytes |"
	// Columns: snap | rev | date | files | bytes | chunks | bytes | uniq | bytes | new | bytes
	// The "all" row has empty files/bytes columns, we need to capture chunks and uniq columns
	allRowRe := regexp.MustCompile(`^\s*(\S+)\s*\|\s*all\s*\|[^|]*\|[^|]*\|[^|]*\|\s*([\d,]+)\s*\|\s*([\d,]+[KMGT]?)\s*\|\s*([\d,]+)\s*\|\s*([\d,]+[KMGT]?)\s*\|`)

	// Count revisions per repository from individual revision lines
	// Format: " repo_name | rev_num | @ date ... |"
	revisionRe := regexp.MustCompile(`^\s*(\S+)\s*\|\s*(\d+)\s*\|\s*@`)

	revisionCounts := make(map[string]int)

	for _, line := range lines {
		// Check for total chunks summary
		if matches := totalChunksRe.FindStringSubmatch(line); matches != nil {
			size, err := parseSize(matches[1])
			if err == nil {
				stats.TotalSize = size
			}
			chunks, err := parseNumber(matches[2])
			if err == nil {
				stats.TotalChunks = int(chunks)
			}
			continue
		}

		// Check for revision lines (to count revisions per repo)
		if matches := revisionRe.FindStringSubmatch(line); matches != nil {
			repoName := matches[1]
			revisionCounts[repoName]++
			continue
		}

		// Check for "all" summary rows
		if matches := allRowRe.FindStringSubmatch(line); matches != nil {
			repoName := matches[1]
			chunks, _ := parseNumber(matches[2])
			totalSize, _ := parseSize(matches[3])
			uniqueChunks, _ := parseNumber(matches[4])
			uniqueSize, _ := parseSize(matches[5])

			stats.Repositories[repoName] = RepoStats{
				TotalChunks: int(chunks),
				TotalSize:   totalSize,
				UniqueSize:  uniqueSize,
				Revisions:   revisionCounts[repoName],
			}
			// Use unique chunks count if different (though typically same as total for "all" row)
			_ = uniqueChunks
		}
	}

	// If no repositories found, return error
	if len(stats.Repositories) == 0 {
		return nil, fmt.Errorf("no repository statistics found in check output")
	}

	return stats, nil
}

// TodayDate returns today's date in YYYY-MM-DD format
func TodayDate() string {
	return time.Now().Format("2006-01-02")
}

// parseSize converts size strings like "4,617M", "8,853K", "123G", "456" to bytes
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")

	if s == "" {
		return 0, nil
	}

	var multiplier int64 = 1
	lastChar := s[len(s)-1]

	switch lastChar {
	case 'K':
		multiplier = 1024
		s = s[:len(s)-1]
	case 'M':
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	case 'G':
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case 'T':
		multiplier = 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}

	// Parse the numeric part
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size %q: %w", s, err)
	}

	return int64(val * float64(multiplier)), nil
}

// parseNumber removes commas and parses an integer
func parseNumber(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")

	if s == "" {
		return 0, nil
	}

	return strconv.ParseInt(s, 10, 64)
}

// FormatBytes formats bytes into human-readable format (e.g., "1.5 GB")
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
