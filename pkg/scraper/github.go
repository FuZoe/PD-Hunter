package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	requestDelay = 500 * time.Millisecond
	maxRetries   = 3
	// GitHub Search has a much smaller quota than the Core API. Keep a
	// conservative interval between Search requests so a full mapping scan
	// does not exhaust the rolling one-minute limit.
	searchIntervalAuthenticated   = 2100 * time.Millisecond
	searchIntervalUnauthenticated = 6100 * time.Millisecond
	maxRateLimitWait              = 2 * time.Minute
)

// Client wraps an HTTP client for GitHub API calls.
type Client struct {
	HTTPClient *http.Client
	Token      string
	BaseURL    string // defaults to https://api.github.com

	searchMu     sync.Mutex
	lastSearchAt time.Time
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
		return strings.TrimRight(c.BaseURL, "/")
	}
	return defaultBaseURL
}

// RateLimitError reports a GitHub response that explicitly indicates rate
// limiting. RetryAt is populated when GitHub supplied a reset/retry time.
type RateLimitError struct {
	StatusCode int
	RetryAt    time.Time
	Body       string
}

func (e *RateLimitError) Error() string {
	if !e.RetryAt.IsZero() {
		return fmt.Sprintf("GitHub rate limit exceeded (HTTP %d), retry after %s", e.StatusCode, e.RetryAt.UTC().Format(time.RFC3339))
	}
	return fmt.Sprintf("GitHub rate limit exceeded (HTTP %d)", e.StatusCode)
}

// ScanAll scans all organizations in the config and returns deduplicated issues.
func (c *Client) ScanAll(config *Config) ([]Issue, error) {
	allIssues := make([]Issue, 0)
	seen := make(map[string]bool)

	for _, org := range config.Organizations {
		fmt.Printf("\n=== Scanning organization: %s ===\n", org.Name)
		fmt.Printf("Note: %s\n", org.Note)

		for _, label := range org.Labels {
			fmt.Printf("\nSearching for label: %s\n", label)

			ghIssues, err := c.SearchBountyIssues(org.Name, label)
			if err != nil {
				return nil, fmt.Errorf("searching %s with label %q: %w", org.Name, label, err)
			}

			for _, ghIssue := range ghIssues {
				if ghIssue.PullRequest != nil || seen[ghIssue.HTMLURL] {
					continue
				}
				if strings.ToLower(ghIssue.State) != "open" {
					continue
				}
				seen[ghIssue.HTMLURL] = true

				issue, err := c.convertIssue(ghIssue)
				if err != nil {
					return nil, fmt.Errorf("enriching %s#%d: %w", ExtractRepoName(ghIssue.HTMLURL), ghIssue.Number, err)
				}
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

	// Build org→labels lookup for filtering project items
	orgLabels := make(map[string]map[string]bool)
	for _, org := range config.Organizations {
		lower := strings.ToLower(org.Name)
		orgLabels[lower] = make(map[string]bool)
		for _, l := range org.Labels {
			orgLabels[lower][strings.ToLower(l)] = true
		}
	}

	// Scan GitHub Projects V2 boards
	for _, proj := range config.Projects {
		fmt.Printf("\n=== Scanning project: %s/%d ===\n", proj.OrgLogin, proj.ProjectNumber)
		fmt.Printf("Note: %s\n", proj.Note)

		ghIssues, err := c.FetchProjectItems(proj.OrgLogin, proj.ProjectNumber)
		if err != nil {
			return nil, fmt.Errorf("fetching project %s/%d: %w", proj.OrgLogin, proj.ProjectNumber, err)
		}

		// Get bounty labels for this org; if org not in config, accept any "bounty" label
		acceptLabels := orgLabels[strings.ToLower(proj.OrgLogin)]

		newCount := 0
		skippedNoLabel := 0
		for _, ghIssue := range ghIssues {
			if ghIssue.PullRequest != nil || seen[ghIssue.HTMLURL] {
				continue
			}

			// Filter: issue must have at least one bounty-related label
			if !hasBountyLabel(ghIssue.Labels, acceptLabels) {
				skippedNoLabel++
				continue
			}

			seen[ghIssue.HTMLURL] = true
			newCount++

			issue, err := c.convertIssue(ghIssue)
			if err != nil {
				return nil, fmt.Errorf("enriching %s#%d: %w", ExtractRepoName(ghIssue.HTMLURL), ghIssue.Number, err)
			}
			title := ghIssue.Title
			if len(title) > 50 {
				title = title[:50]
			}
			fmt.Printf("  [project] Issue #%d: %d open PRs, %d comments - %s\n",
				issue.Number, issue.OpenPRCount, issue.CommentCount, title)

			allIssues = append(allIssues, issue)
		}
		fmt.Printf("  Project %s/%d: %d total items, %d new, %d skipped (no bounty label)\n",
			proj.OrgLogin, proj.ProjectNumber, len(ghIssues), newCount, skippedNoLabel)
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total bounty issues found: %d\n", len(allIssues))
	return allIssues, nil
}

// hasBountyLabel returns true if the issue has at least one label matching the
// accepted bounty labels (case-insensitive). If acceptLabels is nil or empty,
// it falls back to checking if any label contains the substring "bounty".
func hasBountyLabel(issueLabels []GitHubLabel, acceptLabels map[string]bool) bool {
	for _, l := range issueLabels {
		lower := strings.ToLower(l.Name)
		if len(acceptLabels) > 0 {
			if acceptLabels[lower] {
				return true
			}
		} else {
			if strings.Contains(lower, "bounty") {
				return true
			}
		}
	}
	return false
}

func (c *Client) convertIssue(gh GitHubIssue) (Issue, error) {
	labels := make([]string, len(gh.Labels))
	for i, l := range gh.Labels {
		labels[i] = l.Name
	}

	repoName := ExtractRepoName(gh.HTMLURL)
	if repoName == "" {
		return Issue{}, fmt.Errorf("cannot determine repository from issue URL %q", gh.HTMLURL)
	}

	time.Sleep(requestDelay)
	openPRCount, err := c.getOpenPRCount(repoName, gh.Number)
	if err != nil {
		return Issue{}, err
	}

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
	}, nil
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
		if result.IncompleteResults {
			return nil, fmt.Errorf("GitHub returned incomplete search results for org %q label %q", org, label)
		}

		if len(result.Items) == 0 {
			break
		}

		allIssues = append(allIssues, result.Items...)

		if len(allIssues) >= result.TotalCount {
			break
		}
		if page >= 10 {
			return nil, fmt.Errorf("search result for org %q label %q exceeds GitHub's 1000-item limit (%d)", org, label, result.TotalCount)
		}
		page++
	}

	return allIssues, nil
}

// timelineEvent is the subset of an issue timeline event needed to identify
// a referenced pull request. A map is used to deduplicate repeated references
// to the same PR in the timeline.
type timelineEvent struct {
	Event  string `json:"event"`
	Source *struct {
		Issue *struct {
			Number      int       `json:"number"`
			HTMLURL     string    `json:"html_url"`
			State       string    `json:"state"`
			PullRequest *struct{} `json:"pull_request"`
			Repository  *struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
		} `json:"issue"`
	} `json:"source"`
}

// getOpenPRCount uses GitHub's Core API cross-reference events rather than the
// Search API. Search is limited to a small rolling quota and the old
// implementation made one Search request per issue, which exhausted that
// quota on large mappings. This intentionally counts GitHub-linked references;
// it does not treat arbitrary text in a PR comment as a link.
func (c *Client) getOpenPRCount(repoFullName string, issueNumber int) (int, error) {
	parts := strings.Split(repoFullName, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, fmt.Errorf("invalid repository name %q", repoFullName)
	}

	seen := make(map[string]bool)
	count := 0
	for page := 1; page <= 100; page++ {
		apiURL := fmt.Sprintf("%s/repos/%s/%s/issues/%d/timeline?per_page=100&page=%d",
			c.baseURL(), url.PathEscape(parts[0]), url.PathEscape(parts[1]), issueNumber, page)

		data, err := c.DoRequest(apiURL)
		if err != nil {
			return 0, fmt.Errorf("fetching issue timeline for %s#%d: %w", repoFullName, issueNumber, err)
		}

		var events []timelineEvent
		if err := json.Unmarshal(data, &events); err != nil {
			return 0, fmt.Errorf("parsing issue timeline for %s#%d: %w", repoFullName, issueNumber, err)
		}

		for _, event := range events {
			if event.Event != "cross-referenced" {
				continue
			}
			if event.Source == nil || event.Source.Issue == nil || event.Source.Issue.PullRequest == nil {
				continue
			}
			if strings.ToLower(event.Source.Issue.State) != "open" {
				continue
			}
			sourceRepo := ""
			if event.Source.Issue.Repository != nil {
				sourceRepo = event.Source.Issue.Repository.FullName
			}
			if sourceRepo == "" {
				sourceRepo = ExtractRepoName(event.Source.Issue.HTMLURL)
			}
			if !strings.EqualFold(sourceRepo, repoFullName) {
				continue
			}
			key := event.Source.Issue.HTMLURL
			if key == "" {
				key = fmt.Sprintf("%d", event.Source.Issue.Number)
			}
			if !seen[key] {
				seen[key] = true
				count++
			}
		}

		if len(events) < 100 {
			break
		}
		if page == 100 {
			return 0, fmt.Errorf("issue timeline for %s#%d exceeds the 10000-event safety limit", repoFullName, issueNumber)
		}
	}

	return count, nil
}

// GetOpenPRCount returns the number of open PRs referencing an issue number.
// It remains as a compatibility wrapper for callers that cannot handle an
// error. ScanAll uses getOpenPRCount so failed enrichment aborts the scan.
func (c *Client) GetOpenPRCount(repoFullName string, issueNumber int) int {
	count, err := c.getOpenPRCount(repoFullName, issueNumber)
	if err != nil {
		fmt.Printf("  Warning: Could not get PR count for #%d: %v\n", issueNumber, err)
		return 0
	}
	return count
}

// DoRequest performs an authenticated HTTP GET with retries.
func (c *Client) DoRequest(reqURL string) ([]byte, error) {
	var retryDelay time.Duration
	for attempt := 0; attempt < maxRetries; attempt++ {
		if retryDelay > 0 {
			fmt.Printf("  Retrying in %v (attempt %d/%d)...\n", retryDelay, attempt+1, maxRetries)
			time.Sleep(retryDelay)
		}
		retryDelay = 0

		if isSearchURL(reqURL) {
			c.waitForSearchSlot(reqURL)
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

		httpClient := c.HTTPClient
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		if isRateLimitResponse(resp.StatusCode, resp.Header, body) {
			if attempt < maxRetries-1 {
				retryDelay = c.rateLimitRetryDelay(reqURL, resp.Header, attempt)
				if retryDelay < 0 {
					return nil, newRateLimitError(resp.StatusCode, resp.Header, body)
				}
				continue
			}
			return nil, newRateLimitError(resp.StatusCode, resp.Header, body)
		}

		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("max retries exceeded")
}

func isSearchURL(reqURL string) bool {
	parsed, err := url.Parse(reqURL)
	return err == nil && strings.Contains(parsed.Path, "/search/")
}

func isLocalHost(reqURL string) bool {
	parsed, err := url.Parse(reqURL)
	if err != nil {
		return true
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (c *Client) waitForSearchSlot(reqURL string) {
	if isLocalHost(reqURL) {
		return
	}

	interval := searchIntervalAuthenticated
	if c.Token == "" {
		interval = searchIntervalUnauthenticated
	}

	c.searchMu.Lock()
	defer c.searchMu.Unlock()
	if !c.lastSearchAt.IsZero() {
		if wait := interval - time.Since(c.lastSearchAt); wait > 0 {
			time.Sleep(wait)
		}
	}
	c.lastSearchAt = time.Now()
}

func isRateLimitResponse(status int, headers http.Header, body []byte) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	if status != http.StatusForbidden {
		return false
	}
	if headers.Get("Retry-After") != "" || headers.Get("X-RateLimit-Remaining") == "0" {
		return true
	}
	bodyText := strings.ToLower(string(body))
	return strings.Contains(bodyText, "rate limit") || strings.Contains(bodyText, "secondary rate limit")
}

func parseRetryAfter(headers http.Header, now time.Time) time.Duration {
	if value := strings.TrimSpace(headers.Get("Retry-After")); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			if seconds <= 0 {
				return time.Second
			}
			return time.Duration(seconds) * time.Second
		}
		if when, err := http.ParseTime(value); err == nil {
			if wait := when.Sub(now); wait > 0 {
				return wait
			}
			return time.Second
		}
	}
	if reset := strings.TrimSpace(headers.Get("X-RateLimit-Reset")); reset != "" {
		if epoch, err := strconv.ParseInt(reset, 10, 64); err == nil {
			if wait := time.Unix(epoch, 0).Sub(now); wait > 0 {
				return wait
			}
		}
	}
	return 0
}

func (c *Client) rateLimitRetryDelay(reqURL string, headers http.Header, attempt int) time.Duration {
	wait := parseRetryAfter(headers, time.Now())
	if wait == 0 {
		// A local test server should not make the test suite sleep for 15s.
		if isLocalHost(reqURL) {
			return 0
		}
		wait = time.Duration(attempt+1) * 5 * time.Second
	}
	if wait > maxRateLimitWait {
		return -1
	}
	return wait
}

func newRateLimitError(status int, headers http.Header, body []byte) error {
	retryAt := time.Time{}
	if wait := parseRetryAfter(headers, time.Now()); wait > 0 {
		retryAt = time.Now().Add(wait)
	}
	return &RateLimitError{StatusCode: status, RetryAt: retryAt, Body: string(body)}
}

// ExtractRepoName extracts "owner/repo" from a GitHub issue URL.
func ExtractRepoName(issueURL string) string {
	parts := strings.Split(issueURL, "/")
	if len(parts) >= 5 {
		return parts[3] + "/" + parts[4]
	}
	return ""
}
