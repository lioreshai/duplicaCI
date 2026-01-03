package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewForgejo(t *testing.T) {
	n := NewForgejo("https://git.example.com", "user/repo", "token123")

	if n.baseURL != "https://git.example.com" {
		t.Errorf("expected baseURL 'https://git.example.com', got %q", n.baseURL)
	}
	if n.repo != "user/repo" {
		t.Errorf("expected repo 'user/repo', got %q", n.repo)
	}
	if n.token != "token123" {
		t.Errorf("expected token 'token123', got %q", n.token)
	}
}

func TestNewForgejo_TrimsTrailingSlash(t *testing.T) {
	n := NewForgejo("https://git.example.com/", "user/repo", "token123")

	if n.baseURL != "https://git.example.com" {
		t.Errorf("expected baseURL without trailing slash, got %q", n.baseURL)
	}
}

func TestSetAssignee(t *testing.T) {
	n := NewForgejo("https://git.example.com", "user/repo", "token123")
	n.SetAssignee("testuser")

	if n.assignee != "testuser" {
		t.Errorf("expected assignee 'testuser', got %q", n.assignee)
	}
}

func TestCreateIssue_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/api/v1/repos/user/repo/issues" {
			// Check for open issues first
			if r.URL.Path == "/api/v1/repos/user/repo/issues" && r.URL.Query().Get("state") == "open" {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]interface{}{})
				return
			}
		}

		if r.Method == "GET" {
			// Return empty list for issue search
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{})
			return
		}

		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "token testtoken" {
			t.Errorf("expected auth header 'token testtoken', got %q", auth)
		}

		// Return success
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"html_url": "https://git.example.com/user/repo/issues/1",
		})
	}))
	defer server.Close()

	n := NewForgejo(server.URL, "user/repo", "testtoken")
	err := n.CreateOrUpdateIssue("Test Issue", "Test body")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFindExistingIssue_ReturnsID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"number": 42, "title": "Test Issue"},
			{"number": 43, "title": "Other Issue"},
		})
	}))
	defer server.Close()

	n := NewForgejo(server.URL, "user/repo", "testtoken")
	id, err := n.findExistingIssue("Test Issue")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("expected issue ID 42, got %d", id)
	}
}

func TestFindExistingIssue_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"number": 43, "title": "Other Issue"},
		})
	}))
	defer server.Close()

	n := NewForgejo(server.URL, "user/repo", "testtoken")
	id, err := n.findExistingIssue("Test Issue")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if id != 0 {
		t.Errorf("expected issue ID 0 (not found), got %d", id)
	}
}

func TestFindExistingIssue_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	n := NewForgejo(server.URL, "user/repo", "testtoken")
	_, err := n.findExistingIssue("Test Issue")

	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestCreateOrUpdateIssue_AddsComment(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if r.Method == "GET" {
			// Return existing issue
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"number": 42, "title": "Test Issue"},
			})
			return
		}

		if r.Method == "POST" {
			// This should be a comment, not a new issue
			if r.URL.Path != "/api/v1/repos/user/repo/issues/42/comments" {
				t.Errorf("expected comment URL, got %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{})
			return
		}
	}))
	defer server.Close()

	n := NewForgejo(server.URL, "user/repo", "testtoken")
	err := n.CreateOrUpdateIssue("Test Issue", "New body")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateIssue_WithAssignee(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// Return empty list
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{})
			return
		}

		if r.Method == "POST" {
			// Verify assignee is in request body
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)

			assignees, ok := payload["assignees"].([]interface{})
			if !ok || len(assignees) == 0 {
				t.Error("expected assignees in payload")
			} else if assignees[0] != "testuser" {
				t.Errorf("expected assignee 'testuser', got %v", assignees[0])
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"html_url": "https://git.example.com/user/repo/issues/1",
			})
			return
		}
	}))
	defer server.Close()

	n := NewForgejo(server.URL, "user/repo", "testtoken")
	n.SetAssignee("testuser")
	err := n.CreateOrUpdateIssue("New Issue", "Body")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateIssue_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{})
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
	}))
	defer server.Close()

	n := NewForgejo(server.URL, "user/repo", "testtoken")
	err := n.CreateOrUpdateIssue("Test Issue", "Body")

	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestAddComment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/user/repo/issues/42/comments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	n := NewForgejo(server.URL, "user/repo", "testtoken")
	err := n.addComment(42, "Test comment")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAddComment_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	n := NewForgejo(server.URL, "user/repo", "testtoken")
	err := n.addComment(42, "Test comment")

	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestFindExistingIssue_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	n := NewForgejo(server.URL, "user/repo", "testtoken")
	_, err := n.findExistingIssue("Test Issue")

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFindExistingIssue_InvalidURL(t *testing.T) {
	// Test with an invalid URL that causes http.NewRequest to fail
	n := NewForgejo("://invalid-url", "user/repo", "testtoken")
	_, err := n.findExistingIssue("Test Issue")

	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestFindExistingIssue_ConnectionError(t *testing.T) {
	// Create a server and close it immediately to simulate connection error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	n := NewForgejo(serverURL, "user/repo", "testtoken")
	_, err := n.findExistingIssue("Test Issue")

	if err == nil {
		t.Error("expected error for connection failure")
	}
}

func TestCreateIssue_InvalidURL(t *testing.T) {
	// Test with an invalid URL that causes http.NewRequest to fail
	n := NewForgejo("://invalid-url", "user/repo", "testtoken")
	err := n.createIssue("Test Issue", "Body")

	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestCreateIssue_ConnectionError(t *testing.T) {
	// Create a server and close it immediately to simulate connection error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	n := NewForgejo(serverURL, "user/repo", "testtoken")
	err := n.createIssue("Test Issue", "Body")

	if err == nil {
		t.Error("expected error for connection failure")
	}
}

func TestAddComment_InvalidURL(t *testing.T) {
	// Test with an invalid URL that causes http.NewRequest to fail
	n := NewForgejo("://invalid-url", "user/repo", "testtoken")
	err := n.addComment(42, "Test comment")

	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestAddComment_ConnectionError(t *testing.T) {
	// Create a server and close it immediately to simulate connection error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	n := NewForgejo(serverURL, "user/repo", "testtoken")
	err := n.addComment(42, "Test comment")

	if err == nil {
		t.Error("expected error for connection failure")
	}
}

func TestCreateOrUpdateIssue_FindExistingIssueError(t *testing.T) {
	// Test CreateOrUpdateIssue when findExistingIssue returns an error
	n := NewForgejo("://invalid-url", "user/repo", "testtoken")
	err := n.CreateOrUpdateIssue("Test Issue", "Body")

	if err == nil {
		t.Error("expected error when findExistingIssue fails")
	}
}
