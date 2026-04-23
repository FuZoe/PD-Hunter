package scraper

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// validProjectResponse returns a realistic GraphQL response with 2 issues and 1 draft item.
func validProjectResponse() string {
	return `{
		"data": {
			"organization": {
				"projectV2": {
					"items": {
						"pageInfo": {"hasNextPage": false, "endCursor": "abc123"},
						"nodes": [
							{
								"content": {
									"number": 32386,
									"title": "Ship Ubuntu 24.04 + mainline kernel to master",
									"url": "https://github.com/commaai/openpilot/issues/32386",
									"state": "OPEN",
									"createdAt": "2024-03-01T00:00:00Z",
									"updatedAt": "2024-06-15T00:00:00Z",
									"author": {"login": "geohot"},
									"body": "$3000 bounty",
									"labels": {"nodes": [{"name": "bounty"}, {"name": "linux"}]},
									"repository": {"nameWithOwner": "commaai/openpilot"},
									"comments": {"totalCount": 42}
								}
							},
							{
								"content": {
									"number": 2017,
									"title": "[$3000 Bounty] BYD Port",
									"url": "https://github.com/commaai/opendbc/issues/2017",
									"state": "OPEN",
									"createdAt": "2024-02-01T00:00:00Z",
									"updatedAt": "2024-05-10T00:00:00Z",
									"author": {"login": "adeebshihadeh"},
									"body": "BYD port bounty $3000",
									"labels": {"nodes": [{"name": "bounty"}]},
									"repository": {"nameWithOwner": "commaai/opendbc"},
									"comments": {"totalCount": 15}
								}
							},
							{
								"content": null
							}
						]
					}
				}
			}
		}
	}`
}

func TestDoGraphQLRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type, got %s", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var req graphQLRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}
		if req.Query == "" {
			t.Error("expected non-empty query")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {}}`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	data, err := client.DoGraphQLRequest("query { viewer { login } }", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty response")
	}
}

func TestDoGraphQLRequest_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`server error`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	_, err := client.DoGraphQLRequest("query { viewer { login } }", nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "GraphQL HTTP 500") {
		t.Errorf("expected GraphQL HTTP 500 error, got: %v", err)
	}
}

func TestDoGraphQLRequest_NoToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("expected no Authorization header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {}}`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "",
		BaseURL:    server.URL,
	}

	_, err := client.DoGraphQLRequest("query { viewer { login } }", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchProjectItems_WithIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validProjectResponse()))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	issues, err := client.FetchProjectItems("commaai", 26)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should get 2 issues (draft item skipped)
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	// Check first issue
	if issues[0].Number != 32386 {
		t.Errorf("expected issue #32386, got #%d", issues[0].Number)
	}
	if issues[0].HTMLURL != "https://github.com/commaai/openpilot/issues/32386" {
		t.Errorf("unexpected URL: %s", issues[0].HTMLURL)
	}
	if issues[0].User.Login != "geohot" {
		t.Errorf("expected author 'geohot', got '%s'", issues[0].User.Login)
	}
	if issues[0].Comments != 42 {
		t.Errorf("expected 42 comments, got %d", issues[0].Comments)
	}
	if len(issues[0].Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issues[0].Labels))
	}

	// Check second issue
	if issues[1].Number != 2017 {
		t.Errorf("expected issue #2017, got #%d", issues[1].Number)
	}
	if issues[1].User.Login != "adeebshihadeh" {
		t.Errorf("expected author 'adeebshihadeh', got '%s'", issues[1].User.Login)
	}
}

func TestFetchProjectItems_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"organization": {
					"projectV2": {
						"items": {
							"pageInfo": {"hasNextPage": false, "endCursor": ""},
							"nodes": []
						}
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	issues, err := client.FetchProjectItems("commaai", 26)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestFetchProjectItems_Pagination(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)

		if callCount == 1 {
			// First page: has next page
			w.Write([]byte(`{
				"data": {
					"organization": {
						"projectV2": {
							"items": {
								"pageInfo": {"hasNextPage": true, "endCursor": "cursor_page2"},
								"nodes": [
									{
										"content": {
											"number": 100,
											"title": "First page issue",
											"url": "https://github.com/org/repo/issues/100",
											"state": "OPEN",
											"createdAt": "2024-01-01T00:00:00Z",
											"updatedAt": "2024-01-15T00:00:00Z",
											"author": {"login": "dev1"},
											"body": "page 1",
											"labels": {"nodes": []},
											"repository": {"nameWithOwner": "org/repo"},
											"comments": {"totalCount": 5}
										}
									}
								]
							}
						}
					}
				}
			}`))
		} else {
			// Second page: no more pages
			w.Write([]byte(`{
				"data": {
					"organization": {
						"projectV2": {
							"items": {
								"pageInfo": {"hasNextPage": false, "endCursor": "cursor_end"},
								"nodes": [
									{
										"content": {
											"number": 200,
											"title": "Second page issue",
											"url": "https://github.com/org/repo/issues/200",
											"state": "OPEN",
											"createdAt": "2024-02-01T00:00:00Z",
											"updatedAt": "2024-02-15T00:00:00Z",
											"author": {"login": "dev2"},
											"body": "page 2",
											"labels": {"nodes": [{"name": "bounty"}]},
											"repository": {"nameWithOwner": "org/repo"},
											"comments": {"totalCount": 3}
										}
									}
								]
							}
						}
					}
				}
			}`))
		}
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	issues, err := client.FetchProjectItems("org", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 GraphQL requests (pagination), got %d", callCount)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues across pages, got %d", len(issues))
	}
	if issues[0].Number != 100 {
		t.Errorf("expected first issue #100, got #%d", issues[0].Number)
	}
	if issues[1].Number != 200 {
		t.Errorf("expected second issue #200, got #%d", issues[1].Number)
	}
}

func TestFetchProjectItems_ProjectNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"organization": {
					"projectV2": null
				}
			}
		}`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	_, err := client.FetchProjectItems("commaai", 999)
	if err == nil {
		t.Fatal("expected error for missing project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestFetchProjectItems_GraphQLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {"organization": {"projectV2": null}},
			"errors": [{"message": "Could not resolve to a ProjectV2"}]
		}`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	_, err := client.FetchProjectItems("commaai", 26)
	if err == nil {
		t.Fatal("expected error for GraphQL error response")
	}
	if !strings.Contains(err.Error(), "GraphQL error") {
		t.Errorf("expected 'GraphQL error', got: %v", err)
	}
}

func TestFetchProjectItems_SkipsDrafts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// All items are drafts (null content)
		w.Write([]byte(`{
			"data": {
				"organization": {
					"projectV2": {
						"items": {
							"pageInfo": {"hasNextPage": false, "endCursor": ""},
							"nodes": [
								{"content": null},
								{"content": null},
								{"content": null}
							]
						}
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	issues, err := client.FetchProjectItems("commaai", 26)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues (all drafts), got %d", len(issues))
	}
}

func TestFetchProjectItems_NilAuthor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"organization": {
					"projectV2": {
						"items": {
							"pageInfo": {"hasNextPage": false, "endCursor": ""},
							"nodes": [
								{
									"content": {
										"number": 1,
										"title": "Issue with no author",
										"url": "https://github.com/org/repo/issues/1",
										"state": "OPEN",
										"createdAt": "2024-01-01T00:00:00Z",
										"updatedAt": "2024-01-01T00:00:00Z",
										"author": null,
										"body": "test",
										"labels": {"nodes": []},
										"repository": {"nameWithOwner": "org/repo"},
										"comments": {"totalCount": 0}
									}
								}
							]
						}
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	issues, err := client.FetchProjectItems("org", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].User.Login != "" {
		t.Errorf("expected empty login for nil author, got '%s'", issues[0].User.Login)
	}
}

func TestScanAll_WithProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		// GraphQL request (POST)
		if r.Method == "POST" {
			w.Write([]byte(`{
				"data": {
					"organization": {
						"projectV2": {
							"items": {
								"pageInfo": {"hasNextPage": false, "endCursor": ""},
								"nodes": [
									{
										"content": {
											"number": 999,
											"title": "Project-only issue",
											"url": "https://github.com/testorg/testrepo/issues/999",
											"state": "OPEN",
											"createdAt": "2024-03-01T00:00:00Z",
											"updatedAt": "2024-03-15T00:00:00Z",
											"author": {"login": "projdev"},
											"body": "$500 bounty",
											"labels": {"nodes": [{"name": "bounty"}]},
											"repository": {"nameWithOwner": "testorg/testrepo"},
											"comments": {"totalCount": 7}
										}
									},
									{
										"content": {
											"number": 100,
											"title": "Already seen in label scan",
											"url": "https://github.com/testorg/testrepo/issues/100",
											"state": "OPEN",
											"createdAt": "2024-01-01T00:00:00Z",
											"updatedAt": "2024-01-15T00:00:00Z",
											"author": {"login": "hunter"},
											"body": "Bounty $500",
											"labels": {"nodes": [{"name": "bounty"}]},
											"repository": {"nameWithOwner": "testorg/testrepo"},
											"comments": {"totalCount": 5}
										}
									}
								]
							}
						}
					}
				}
			}`))
			return
		}

		// REST API requests (GET)
		query := r.URL.Query().Get("q")

		if strings.Contains(query, "is:pr") {
			w.Write([]byte(`{"total_count": 1, "items": []}`))
			return
		}

		// Label search returns issue #100
		w.Write([]byte(`{
			"total_count": 1,
			"items": [{
				"number": 100,
				"title": "Already seen in label scan",
				"html_url": "https://github.com/testorg/testrepo/issues/100",
				"state": "open",
				"labels": [{"name": "bounty"}],
				"comments": 5,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-15T00:00:00Z",
				"user": {"login": "hunter"},
				"body": "Bounty $500"
			}]
		}`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	config := &Config{
		Organizations: []Organization{
			{Name: "testorg", Labels: []string{"bounty"}, Note: "Test org"},
		},
		Projects: []ProjectBoard{
			{OrgLogin: "testorg", ProjectNumber: 1, Note: "Test project"},
		},
	}

	issues, err := client.ScanAll(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Issue #100 from label scan + Issue #999 from project (issue #100 from project is deduped)
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues (1 label + 1 new from project), got %d", len(issues))
	}

	// Verify we got both #100 and #999
	numbers := map[int]bool{}
	for _, issue := range issues {
		numbers[issue.Number] = true
	}
	if !numbers[100] {
		t.Error("expected issue #100 from label scan")
	}
	if !numbers[999] {
		t.Error("expected issue #999 from project scan")
	}
}

func TestScanAll_ProjectError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			// GraphQL returns error
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`server error`))
			return
		}

		// REST works fine
		query := r.URL.Query().Get("q")
		w.WriteHeader(http.StatusOK)
		if strings.Contains(query, "is:pr") {
			w.Write([]byte(`{"total_count": 0, "items": []}`))
			return
		}
		w.Write([]byte(`{
			"total_count": 1,
			"items": [{
				"number": 1,
				"title": "Label issue",
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

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Token:      "test-token",
		BaseURL:    server.URL,
	}

	config := &Config{
		Organizations: []Organization{
			{Name: "org", Labels: []string{"bounty"}, Note: "Test"},
		},
		Projects: []ProjectBoard{
			{OrgLogin: "org", ProjectNumber: 1, Note: "Broken project"},
		},
	}

	// ScanAll should continue even if project fetch fails
	issues, err := client.ScanAll(config)
	if err != nil {
		t.Fatalf("ScanAll should not return error on project failure: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue from label scan (project failed), got %d", len(issues))
	}
}
