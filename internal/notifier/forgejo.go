package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ForgejoNotifier sends notifications via Forgejo issues
type ForgejoNotifier struct {
	baseURL  string
	repo     string
	token    string
	assignee string
	client   *http.Client
}

// NewForgejo creates a new Forgejo notifier
func NewForgejo(baseURL, repo, token string) *ForgejoNotifier {
	return &ForgejoNotifier{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		repo:    repo,
		token:   token,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// SetAssignee sets the user to assign issues to
func (f *ForgejoNotifier) SetAssignee(username string) {
	f.assignee = username
}

// CreateOrUpdateIssue creates a new issue or adds a comment to an existing one
func (f *ForgejoNotifier) CreateOrUpdateIssue(title, body string) error {
	// Check for existing open issue with same title
	existingID, err := f.findExistingIssue(title)
	if err != nil {
		return fmt.Errorf("failed to search for existing issues: %w", err)
	}

	if existingID > 0 {
		// Add comment to existing issue
		return f.addComment(existingID, body)
	}

	// Create new issue
	return f.createIssue(title, body)
}

func (f *ForgejoNotifier) findExistingIssue(title string) (int, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/issues?state=open&type=issues", f.baseURL, f.repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "token "+f.token)

	resp, err := f.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var issues []struct {
		ID    int    `json:"number"`
		Title string `json:"title"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return 0, err
	}

	for _, issue := range issues {
		if issue.Title == title {
			return issue.ID, nil
		}
	}

	return 0, nil
}

func (f *ForgejoNotifier) createIssue(title, body string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/issues", f.baseURL, f.repo)

	payload := map[string]interface{}{
		"title": title,
		"body":  body,
	}

	if f.assignee != "" {
		payload["assignees"] = []string{f.assignee}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+f.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.HTMLURL != "" {
		fmt.Printf("    Created issue: %s\n", result.HTMLURL)
	}

	return nil
}

func (f *ForgejoNotifier) addComment(issueID int, body string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/issues/%d/comments", f.baseURL, f.repo, issueID)

	timestamp := time.Now().Format("2006-01-02 15:04:05 MST")
	commentBody := fmt.Sprintf("**Update %s**\n\n%s", timestamp, body)

	payload := map[string]string{
		"body": commentBody,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+f.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("    Added comment to issue #%d\n", issueID)
	return nil
}
