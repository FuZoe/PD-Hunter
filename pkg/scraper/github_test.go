package scraper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard issue URL",
			url:      "https://github.com/projectdiscovery/nuclei/issues/123",
			expected: "projectdiscovery/nuclei",
		},
		{
			name:     "deep path URL",
			url:      "https://github.com/onyx-dot-app/onyx/issues/456",
			expected: "onyx-dot-app/onyx",
		},
		{
			name:     "PR URL also works",
			url:      "https://github.com/commaai/openpilot/pull/789",
			expected: "commaai/openpilot",
		},
		{
			name:     "short URL returns empty",
			url:      "https://github.com",
			expected: "",
		},
		{
			name:     "empty string",
			url:      "",
			expected: "",
		},
		{
			name:     "no scheme returns wrong segments",
			url:      "github.com/owner/repo/issues/1",
			expected: "issues/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractRepoName(tt.url)
			if got != tt.expected {
				t.Errorf("ExtractRepoName(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestDoRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("unexpected Accept header: %s", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"total_count": 0, "items": []}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	data, err := client.DoRequest(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result GitHubSearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.TotalCount != 0 {
		t.Errorf("expected total_count 0, got %d", result.TotalCount)
	}
}

func TestDoRequest_NoToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("expected no Authorization header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient("")
	_, err := client.DoRequest(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoRequest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal server error`))
	}))
	defer server.Close()

	client := NewClient("")
	_, err := client.DoRequest(server.URL)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func newMockClient(serverURL, token string) *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      token,
		BaseURL:    serverURL,
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("my-token")
	if client.Token != "my-token" {
		t.Errorf("expected token 'my-token', got '%s'", client.Token)
	}
	if client.HTTPClient == nil {
		t.Error("expected non-nil HTTPClient")
	}
	if client.BaseURL != defaultBaseURL {
		t.Errorf("expected BaseURL '%s', got '%s'", defaultBaseURL, client.BaseURL)
	}
}

func TestSearchBountyIssues_WithResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"total_count": 2,
			"items": [
				{"number": 1, "title": "Bug fix bounty", "html_url": "https://github.com/org/repo/issues/1", "state": "open", "labels": [{"name": "bounty"}], "comments": 3, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-15T00:00:00Z", "user": {"login": "alice"}, "body": "fix this"},
				{"number": 2, "title": "Feature bounty", "html_url": "https://github.com/org/repo/issues/2", "state": "open", "labels": [{"name": "bounty"}], "comments": 7, "created_at": "2024-02-01T00:00:00Z", "updated_at": "2024-02-15T00:00:00Z", "user": {"login": "bob"}, "body": "add this"}
			]
		}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test-token")
	issues, err := client.SearchBountyIssues("org", "bounty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Number != 1 {
		t.Errorf("expected issue #1, got #%d", issues[0].Number)
	}
	if issues[1].User.Login != "bob" {
		t.Errorf("expected user 'bob', got '%s'", issues[1].User.Login)
	}
}

func TestSearchBountyIssues_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"total_count": 0, "items": []}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	issues, err := client.SearchBountyIssues("org", "bounty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestSearchBountyIssues_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`server error`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	_, err := client.SearchBountyIssues("org", "bounty")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGetOpenPRCount_WithResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"total_count": 3, "items": []}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	count := client.GetOpenPRCount("org/repo", 42)
	if count != 3 {
		t.Errorf("expected 3 PRs, got %d", count)
	}
}

func TestGetOpenPRCount_Zero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"total_count": 0, "items": []}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	count := client.GetOpenPRCount("org/repo", 99)
	if count != 0 {
		t.Errorf("expected 0 PRs, got %d", count)
	}
}

func TestGetOpenPRCount_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`error`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	count := client.GetOpenPRCount("org/repo", 1)
	if count != 0 {
		t.Errorf("expected 0 on error, got %d", count)
	}
}

func TestGetOpenPRCount_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	count := client.GetOpenPRCount("org/repo", 1)
	if count != 0 {
		t.Errorf("expected 0 on invalid JSON, got %d", count)
	}
}

func TestScanAll_FullFlow(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		query := r.URL.Query().Get("q")
		w.WriteHeader(http.StatusOK)

		// PR count queries
		if strings.Contains(query, "is:pr") {
			w.Write([]byte(`{"total_count": 2, "items": []}`))
			return
		}

		// Issue search queries
		w.Write([]byte(`{
			"total_count": 1,
			"items": [{
				"number": 100,
				"title": "A very long bounty title that exceeds fifty characters limit for testing",
				"html_url": "https://github.com/testorg/testrepo/issues/100",
				"state": "open",
				"labels": [{"name": "bounty"}, {"name": "good first issue"}],
				"comments": 5,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-15T00:00:00Z",
				"user": {"login": "hunter"},
				"body": "Bounty $500"
			}]
		}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test-token")

	config := &Config{
		Organizations: []Organization{
			{Name: "testorg", Labels: []string{"bounty"}, Note: "Test org"},
		},
	}

	issues, err := client.ScanAll(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Number != 100 {
		t.Errorf("expected issue #100, got #%d", issue.Number)
	}
	if issue.Repository != "testorg/testrepo" {
		t.Errorf("expected repo 'testorg/testrepo', got '%s'", issue.Repository)
	}
	if issue.OpenPRCount != 2 {
		t.Errorf("expected 2 open PRs, got %d", issue.OpenPRCount)
	}
	if issue.Author != "hunter" {
		t.Errorf("expected author 'hunter', got '%s'", issue.Author)
	}
	if len(issue.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issue.Labels))
	}
}

func TestScanAll_DeduplicatesIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(query, "is:pr") {
			w.Write([]byte(`{"total_count": 0, "items": []}`))
			return
		}

		// Return the same issue for both labels
		w.Write([]byte(`{
			"total_count": 1,
			"items": [{
				"number": 1,
				"title": "Duplicate issue",
				"html_url": "https://github.com/org/repo/issues/1",
				"state": "open",
				"labels": [{"name": "bounty"}],
				"comments": 1,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z",
				"user": {"login": "dev"},
				"body": "test"
			}]
		}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	config := &Config{
		Organizations: []Organization{
			{Name: "org", Labels: []string{"bounty", "reward"}, Note: "Test"},
		},
	}

	issues, err := client.ScanAll(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Same issue returned for both labels, should be deduplicated to 1
	if len(issues) != 1 {
		t.Errorf("expected 1 deduplicated issue, got %d", len(issues))
	}
}

func TestScanAll_SkipsPRs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(query, "is:pr") {
			w.Write([]byte(`{"total_count": 0, "items": []}`))
			return
		}

		// Return a PR (has pull_request field) and a real issue
		w.Write([]byte(`{
			"total_count": 2,
			"items": [
				{"number": 1, "title": "This is a PR", "html_url": "https://github.com/org/repo/pull/1", "state": "open", "labels": [], "comments": 0, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z", "user": {"login": "dev"}, "body": "", "pull_request": {}},
				{"number": 2, "title": "Real issue", "html_url": "https://github.com/org/repo/issues/2", "state": "open", "labels": [], "comments": 0, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z", "user": {"login": "dev"}, "body": ""}
			]
		}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	config := &Config{
		Organizations: []Organization{
			{Name: "org", Labels: []string{"bounty"}, Note: "Test"},
		},
	}

	issues, err := client.ScanAll(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue (PR skipped), got %d", len(issues))
	}
	if issues[0].Number != 2 {
		t.Errorf("expected issue #2, got #%d", issues[0].Number)
	}
}

func TestScanAll_SearchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`error`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	config := &Config{
		Organizations: []Organization{
			{Name: "org", Labels: []string{"bounty"}, Note: "Test"},
		},
	}

	// ScanAll continues on error (prints warning), returns empty
	issues, err := client.ScanAll(config)
	if err != nil {
		t.Fatalf("ScanAll should not return error on search failure: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues on search error, got %d", len(issues))
	}
}

func TestExtractRepoName_VariousFormats(t *testing.T) {
	// Test with exactly 5 parts (minimum for valid extraction)
	got := ExtractRepoName("a/b/c/owner/repo")
	if got != "owner/repo" {
		t.Errorf("expected 'owner/repo', got '%s'", got)
	}

	// Test with trailing slash
	got = ExtractRepoName("https://github.com/org/repo/")
	if got != "org/repo" {
		t.Errorf("expected 'org/repo', got '%s'", got)
	}

	// Test with only 4 parts (less than 5, returns empty)
	got = ExtractRepoName("a/b/c/d")
	if got != "" {
		t.Errorf("expected '', got '%s'", got)
	}
}
