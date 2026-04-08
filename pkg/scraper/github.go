package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	requestDelay = 500 * time.Millisecond
	maxRetries   = 3
)

// Client wraps an HTTP client for GitHub API calls.
type Client struct {
	HTTPClient *http.Client
	Token      string
	BaseURL    string // defaults to https://api.github.com
}

const defaultBaseURL = "https://api.github.com"

// NewClient creates a new GitHub API client.
func NewClient(token string) *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Token:      token,
		BaseURL:    defaultBaseURL,
	}
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return defaultBaseURL
}

// ScanAll scans all organizations in the config and returns deduplicated issues.
func (c *Client) ScanAll(config *Config) ([]Issue, error) {
	var allIssues []Issue
	seen := make(map[string]bool)

	for _, org := range config.Organizations {
		fmt.Printf("\n=== Scanning organization: %s ===\n", org.Name)
		fmt.Printf("Note: %s\n", org.Note)

		for _, label := range org.Labels {
			fmt.Printf("\nSearching for label: %s\n", label)
			time.Sleep(requestDelay)

			ghIssues, err := c.SearchBountyIssues(org.Name, label)
			if err != nil {
				fmt.Printf("Warning: Error searching %s with label '%s': %v\n", org.Name, label, err)
				continue
			}

			for _, ghIssue := range ghIssues {
				if ghIssue.PullRequest != nil || seen[ghIssue.HTMLURL] {
					continue
				}
				seen[ghIssue.HTMLURL] = true

				issue := c.convertIssue(ghIssue)
				title := ghIssue.Title
				if len(title) > 50 {
					title = title[:50]
				}
				fmt.Printf("  Issue #%d: %d open PRs, %d comments - %s\n",
					issue.Number, issue.OpenPRCount, issue.CommentCount, title)

				allIssues = append(allIssues, issue)
			}
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total bounty issues found: %d\n", len(allIssues))
	return allIssues, nil
}

func (c *Client) convertIssue(gh GitHubIssue) Issue {
	labels := make([]string, len(gh.Labels))
	for i, l := range gh.Labels {
		labels[i] = l.Name
	}

	repoName := ExtractRepoName(gh.HTMLURL)

	time.Sleep(requestDelay)
	openPRCount := c.GetOpenPRCount(repoName, gh.Number)

	return Issue{
		Number:       gh.Number,
		Title:        gh.Title,
		URL:          gh.HTMLURL,
		State:        gh.State,
		Labels:       labels,
		CommentCount: gh.Comments,
		OpenPRCount:  openPRCount,
		Repository:   repoName,
		CreatedAt:    gh.CreatedAt,
		UpdatedAt:    gh.UpdatedAt,
		Author:       gh.User.Login,
		Body:         gh.Body,
	}
}

// SearchBountyIssues searches for open bounty issues in an org with a given label.
func (c *Client) SearchBountyIssues(org, label string) ([]GitHubIssue, error) {
	var allIssues []GitHubIssue
	page := 1

	for {
		query := fmt.Sprintf("is:open is:issue org:%s label:\"%s\"", org, label)
		apiURL := fmt.Sprintf("%s/search/issues?q=%s&per_page=100&page=%d",
			c.baseURL(), url.QueryEscape(query), page)

		data, err := c.DoRequest(apiURL)
		if err != nil {
			return nil, err
		}

		var result GitHubSearchResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("parsing search results: %w", err)
		}

		if len(result.Items) == 0 {
			break
		}

		allIssues = append(allIssues, result.Items...)

		if len(allIssues) >= result.TotalCount || page >= 10 {
			break
		}
		page++
		time.Sleep(requestDelay)
	}

	return allIssues, nil
}

// GetOpenPRCount returns the number of open PRs referencing an issue number.
func (c *Client) GetOpenPRCount(repoFullName string, issueNumber int) int {
	query := fmt.Sprintf("is:pr is:open repo:%s %d", repoFullName, issueNumber)
	apiURL := fmt.Sprintf("%s/search/issues?q=%s", c.baseURL(), url.QueryEscape(query))

	data, err := c.DoRequest(apiURL)
	if err != nil {
		fmt.Printf("  Warning: Could not get PR count for #%d: %v\n", issueNumber, err)
		return 0
	}

	var result GitHubSearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return 0
	}

	return result.TotalCount
}

// DoRequest performs an authenticated HTTP GET with retries.
func (c *Client) DoRequest(reqURL string) ([]byte, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(attempt*5) * time.Second
			fmt.Printf("  Retrying in %v (attempt %d/%d)...\n", waitTime, attempt+1, maxRetries)
			time.Sleep(waitTime)
		}

		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		if resp.StatusCode == 429 || resp.StatusCode == 403 {
			if attempt < maxRetries-1 {
				continue
			}
		}

		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// ExtractRepoName extracts "owner/repo" from a GitHub issue URL.
func ExtractRepoName(issueURL string) string {
	parts := strings.Split(issueURL, "/")
	if len(parts) >= 5 {
		return parts[3] + "/" + parts[4]
	}
	return ""
}
