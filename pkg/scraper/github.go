package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// requestDelay spaces out non-search requests (GraphQL and REST core).
	requestDelay = 500 * time.Millisecond
	// The Search API allows 30 requests per minute with a token and 10
	// without one. Pacing slightly slower than that keeps a scan under budget.
	searchDelayAuthenticated   = 2100 * time.Millisecond
	searchDelayUnauthenticated = 6100 * time.Millisecond

	maxAttempts = 4
	// defaultMaxRateLimitWait caps a single rate-limit wait. A longer reset,
	// such as an exhausted hourly core budget, fails the scan instead of
	// stalling the workflow.
	defaultMaxRateLimitWait = 15 * time.Minute
	// maxSearchResults is the most results the Search API returns per query.
	maxSearchResults = 1000
)

type rateBucket string

const (
	bucketCore    rateBucket = "core"
	bucketSearch  rateBucket = "search"
	bucketGraphQL rateBucket = "graphql"
)

// Client wraps an HTTP client for GitHub API calls. It is not safe for
// concurrent use.
//
// NewClient configures request pacing. A Client built as a struct literal
// does not pace requests, which keeps tests fast, but still honours
// rate-limit responses.
type Client struct {
	HTTPClient *http.Client
	Token      string
	BaseURL    string // defaults to https://api.github.com

	// SearchInterval and RequestInterval are the minimum gaps between
	// consecutive Search API requests and other requests.
	SearchInterval  time.Duration
	RequestInterval time.Duration
	// MaxRateLimitWait caps one rate-limit wait (0 means the default).
	MaxRateLimitWait time.Duration

	sleep       func(time.Duration)
	now         func() time.Time
	nextAllowed map[rateBucket]time.Time
}

const defaultBaseURL = "https://api.github.com"

// NewClient creates a new GitHub API client with request pacing.
func NewClient(token string) *Client {
	searchInterval := searchDelayAuthenticated
	if token == "" {
		searchInterval = searchDelayUnauthenticated
	}
	return &Client{
		HTTPClient:      &http.Client{Timeout: 30 * time.Second},
		Token:           token,
		BaseURL:         defaultBaseURL,
		SearchInterval:  searchInterval,
		RequestInterval: requestDelay,
	}
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return defaultBaseURL
}

func (c *Client) sleepFor(d time.Duration) {
	if d <= 0 {
		return
	}
	if c.sleep != nil {
		c.sleep(d)
		return
	}
	time.Sleep(d)
}

func (c *Client) currentTime() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *Client) maxRateLimitWait() time.Duration {
	if c.MaxRateLimitWait > 0 {
		return c.MaxRateLimitWait
	}
	return defaultMaxRateLimitWait
}

func (c *Client) interval(b rateBucket) time.Duration {
	if b == bucketSearch {
		return c.SearchInterval
	}
	return c.RequestInterval
}

// holdUntil blocks the bucket until t (it never shortens an existing hold).
func (c *Client) holdUntil(b rateBucket, t time.Time) {
	if c.nextAllowed == nil {
		c.nextAllowed = make(map[rateBucket]time.Time)
	}
	if t.After(c.nextAllowed[b]) {
		c.nextAllowed[b] = t
	}
}

// waitTurn sleeps until the bucket may send its next request, then reserves
// the following slot. It fails instead of sleeping longer than the cap.
func (c *Client) waitTurn(b rateBucket) error {
	wait := c.nextAllowed[b].Sub(c.currentTime())
	if wait > c.maxRateLimitWait() {
		return fmt.Errorf("%s rate limit resets in %v, longer than the %v limit", b, wait.Round(time.Second), c.maxRateLimitWait())
	}
	if wait >= 10*time.Second {
		fmt.Printf("  Waiting %v for the %s rate limit to reset...\n", wait.Round(time.Second), b)
	}
	c.sleepFor(wait)
	c.holdUntil(b, c.currentTime().Add(c.interval(b)))
	return nil
}

// recordRateLimit holds the bucket until its reset time once GitHub reports
// the budget is spent, so the next request waits instead of being rejected.
func (c *Client) recordRateLimit(b rateBucket, h http.Header) {
	if h.Get("X-RateLimit-Remaining") != "0" {
		return
	}
	if reset, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64); err == nil {
		c.holdUntil(b, time.Unix(reset, 0).Add(time.Second))
	}
}

// rateLimitRetryAt reports whether a 403/429 response is a rate limit and,
// if so, when to retry. Other 403s (for example missing permissions) are
// not retried.
func (c *Client) rateLimitRetryAt(resp *http.Response, body []byte, attempt int) (time.Time, bool) {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return time.Time{}, false
	}
	return c.retryAt(resp.Header, body, resp.StatusCode == http.StatusTooManyRequests, attempt)
}

// retryAt works out when to retry a rate-limited request from its headers.
// limited forces a backoff when the headers say nothing; otherwise the body
// must mention a rate limit.
func (c *Client) retryAt(h http.Header, body []byte, limited bool, attempt int) (time.Time, bool) {
	now := c.currentTime()
	if s := h.Get("Retry-After"); s != "" {
		if secs, err := strconv.Atoi(s); err == nil && secs >= 0 {
			return now.Add(time.Duration(secs) * time.Second), true
		}
	}
	if h.Get("X-RateLimit-Remaining") == "0" {
		if reset, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			retryAt := time.Unix(reset, 0).Add(time.Second)
			if retryAt.Before(now.Add(time.Second)) {
				retryAt = now.Add(time.Second)
			}
			return retryAt, true
		}
	}
	// Secondary rate limits without headers: GitHub asks clients to wait at
	// least a minute and back off exponentially.
	if limited || strings.Contains(strings.ToLower(string(body)), "rate limit") {
		return now.Add(time.Minute << uint(attempt)), true
	}
	return time.Time{}, false
}

// send performs a request with pacing and retries. newReq must build a fresh
// request for every attempt. errPrefix labels HTTP errors ("HTTP",
// "GraphQL HTTP"). limitedBody, if set, detects rate limits reported in the
// body of a 200 response (GraphQL does this).
func (c *Client) send(b rateBucket, errPrefix string, newReq func() (*http.Request, error), limitedBody func([]byte) bool) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := c.waitTurn(b); err != nil {
			if lastErr != nil {
				return nil, fmt.Errorf("%w (last error: %v)", err, lastErr)
			}
			return nil, err
		}

		req, err := newReq()
		if err != nil {
			return nil, err
		}

		retryAt := c.currentTime().Add(time.Duration(attempt+1) * 5 * time.Second)
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			c.recordRateLimit(b, resp.Header)

			switch {
			case readErr != nil:
				lastErr = fmt.Errorf("reading response: %w", readErr)
			case resp.StatusCode == http.StatusOK && (limitedBody == nil || !limitedBody(body)):
				return body, nil
			case resp.StatusCode == http.StatusOK:
				lastErr = fmt.Errorf("%s 200: rate limited: %s", errPrefix, string(body))
				retryAt, _ = c.retryAt(resp.Header, body, true, attempt)
			default:
				lastErr = fmt.Errorf("%s %d: %s", errPrefix, resp.StatusCode, string(body))
				if at, ok := c.rateLimitRetryAt(resp, body, attempt); ok {
					retryAt = at
				} else if resp.StatusCode < 500 {
					// Client errors other than rate limits will not succeed on retry.
					return nil, lastErr
				}
			}
		}

		if attempt == maxAttempts-1 {
			break
		}
		fmt.Printf("  %s; retrying (attempt %d/%d)...\n", shorten(lastErr.Error(), 160), attempt+2, maxAttempts)
		c.holdUntil(b, retryAt)
	}
	return nil, fmt.Errorf("giving up after %d attempts: %w", maxAttempts, lastErr)
}

// DoRequest performs an authenticated, paced HTTP GET with retries.
func (c *Client) DoRequest(reqURL string) ([]byte, error) {
	return c.send(bucketFor(reqURL), "HTTP", func() (*http.Request, error) {
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		return req, nil
	}, nil)
}

// bucketFor picks the rate-limit bucket for a REST URL.
func bucketFor(reqURL string) rateBucket {
	if u, err := url.Parse(reqURL); err == nil && strings.Contains(u.Path, "/search/") {
		return bucketSearch
	}
	return bucketCore
}

// searchIssues runs one Search API page. A response flagged
// incomplete_results (GitHub timed out and may have dropped matches) is
// retried and then reported as an error.
func (c *Client) searchIssues(query string, perPage, page int) (*GitHubSearchResult, error) {
	apiURL := fmt.Sprintf("%s/search/issues?q=%s&per_page=%d&page=%d",
		c.baseURL(), url.QueryEscape(query), perPage, page)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		data, err := c.DoRequest(apiURL)
		if err != nil {
			return nil, err
		}
		var result GitHubSearchResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("parsing search results: %w", err)
		}
		if !result.IncompleteResults {
			return &result, nil
		}
		fmt.Printf("  Search returned incomplete results; retrying (attempt %d/%d)...\n", attempt+2, maxAttempts)
	}
	return nil, fmt.Errorf("search %q kept returning incomplete results", query)
}

// searchAllIssues returns every result of a Search API query. Queries with
// more matches than the Search API can return fail instead of truncating.
func (c *Client) searchAllIssues(query string) ([]GitHubIssue, error) {
	var all []GitHubIssue
	for page := 1; ; page++ {
		result, err := c.searchIssues(query, 100, page)
		if err != nil {
			return nil, err
		}
		if result.TotalCount > maxSearchResults {
			return nil, fmt.Errorf("search %q matched %d issues, more than the %d the Search API returns", query, result.TotalCount, maxSearchResults)
		}
		all = append(all, result.Items...)
		if len(result.Items) == 0 || len(all) >= result.TotalCount || page*100 >= maxSearchResults {
			return all, nil
		}
	}
}

// SearchBountyIssues searches for open bounty issues in an org with a given label.
func (c *Client) SearchBountyIssues(org, label string) ([]GitHubIssue, error) {
	return c.searchAllIssues(fmt.Sprintf("is:open is:issue org:%s label:\"%s\"", org, label))
}

// SearchTextIssues searches for open issues in an org whose title or body
// contains a phrase.
func (c *Client) SearchTextIssues(org string, ts TextSearch) ([]GitHubIssue, error) {
	if ts.In != "title" && ts.In != "body" {
		return nil, fmt.Errorf("unsupported text search field %q", ts.In)
	}
	return c.searchAllIssues(fmt.Sprintf("is:open is:issue org:%s in:%s \"%s\"", org, ts.In, ts.Phrase))
}

// GetOpenPRCount returns the number of open PRs referencing an issue number.
func (c *Client) GetOpenPRCount(repoFullName string, issueNumber int) (int, error) {
	query := fmt.Sprintf("is:pr is:open repo:%s \"#%d\"", repoFullName, issueNumber)
	result, err := c.searchIssues(query, 1, 1)
	if err != nil {
		return 0, err
	}
	return result.TotalCount, nil
}

// ScanAll scans every configured organization and project board and returns
// deduplicated, available bounty issues. Any failed query fails the whole
// scan so that partial data is never published.
func (c *Client) ScanAll(config *Config) ([]Issue, error) {
	allIssues := make([]Issue, 0)
	seen := make(map[string]bool)
	excluded := labelSet(config.ExcludeLabels)
	excludedCount := 0

	// add records an open, unseen, available issue and fetches its PR count.
	add := func(gh GitHubIssue, prefix string) (bool, error) {
		if gh.PullRequest != nil || seen[gh.HTMLURL] {
			return false, nil
		}
		if strings.ToLower(gh.State) != "open" {
			return false, nil
		}
		seen[gh.HTMLURL] = true
		if name, ok := firstMatchingLabel(gh.Labels, excluded); ok {
			excludedCount++
			fmt.Printf("  %sSkipping #%d (%s): labeled %q\n", prefix, gh.Number, ExtractRepoName(gh.HTMLURL), name)
			return false, nil
		}
		issue, err := c.convertIssue(gh)
		if err != nil {
			return false, err
		}
		fmt.Printf("  %sIssue #%d: %d open PRs, %d comments - %s\n",
			prefix, issue.Number, issue.OpenPRCount, issue.CommentCount, truncate(gh.Title, 50))
		allIssues = append(allIssues, issue)
		return true, nil
	}

	for _, org := range config.Organizations {
		fmt.Printf("\n=== Scanning organization: %s ===\n", org.Name)
		fmt.Printf("Note: %s\n", org.Note)

		for _, label := range org.Labels {
			fmt.Printf("\nSearching for label: %s\n", label)
			ghIssues, err := c.SearchBountyIssues(org.Name, label)
			if err != nil {
				return nil, fmt.Errorf("searching %s for label %q: %w", org.Name, label, err)
			}
			want := labelSet([]string{label})
			stale := 0
			for _, gh := range ghIssues {
				// The search index can lag behind label removals; keep only
				// issues that still carry the label.
				if _, ok := firstMatchingLabel(gh.Labels, want); !ok {
					stale++
					continue
				}
				if _, err := add(gh, ""); err != nil {
					return nil, err
				}
			}
			if stale > 0 {
				fmt.Printf("  Skipped %d search hits that no longer carry %q\n", stale, label)
			}
		}

		for _, ts := range org.TextSearches {
			fmt.Printf("\nSearching %s for: %q\n", ts.In, ts.Phrase)
			ghIssues, err := c.SearchTextIssues(org.Name, ts)
			if err != nil {
				return nil, fmt.Errorf("searching %s %s for %q: %w", org.Name, ts.In, ts.Phrase, err)
			}
			for _, gh := range ghIssues {
				if !textMatches(gh, ts) {
					continue
				}
				if _, err := add(gh, ""); err != nil {
					return nil, err
				}
			}
		}
	}

	// Build org→labels lookup for filtering project items
	orgLabels := make(map[string]map[string]bool)
	for _, org := range config.Organizations {
		orgLabels[strings.ToLower(org.Name)] = labelSet(org.Labels)
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
		for _, gh := range ghIssues {
			if gh.PullRequest != nil || seen[gh.HTMLURL] {
				continue
			}
			// Filter: issue must have at least one bounty-related label
			if !hasBountyLabel(gh.Labels, acceptLabels) {
				skippedNoLabel++
				continue
			}
			added, err := add(gh, "[project] ")
			if err != nil {
				return nil, err
			}
			if added {
				newCount++
			}
		}
		fmt.Printf("  Project %s/%d: %d total items, %d new, %d skipped (no bounty label)\n",
			proj.OrgLogin, proj.ProjectNumber, len(ghIssues), newCount, skippedNoLabel)
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total bounty issues found: %d\n", len(allIssues))
	fmt.Printf("Skipped as unavailable (exclude_labels): %d\n", excludedCount)
	return allIssues, nil
}

// normalizeLabel folds a label for comparison: lower case, emoji variation
// selectors removed, and whitespace collapsed. "⏸️ Hold" and "⏸ hold" match.
func normalizeLabel(label string) string {
	label = strings.Map(func(r rune) rune {
		if r == '\uFE0E' || r == '\uFE0F' {
			return -1
		}
		return r
	}, label)
	return strings.ToLower(strings.Join(strings.Fields(label), " "))
}

// labelSet builds a normalized lookup set from labels.
func labelSet(labels []string) map[string]bool {
	set := make(map[string]bool, len(labels))
	for _, l := range labels {
		set[normalizeLabel(l)] = true
	}
	return set
}

// firstMatchingLabel returns the first issue label contained in set.
func firstMatchingLabel(issueLabels []GitHubLabel, set map[string]bool) (string, bool) {
	if len(set) == 0 {
		return "", false
	}
	for _, l := range issueLabels {
		if set[normalizeLabel(l.Name)] {
			return l.Name, true
		}
	}
	return "", false
}

// hasBountyLabel returns true if the issue has at least one label matching the
// accepted bounty labels (normalized with normalizeLabel). If acceptLabels is
// nil or empty, it falls back to checking if any label contains "bounty".
func hasBountyLabel(issueLabels []GitHubLabel, acceptLabels map[string]bool) bool {
	if len(acceptLabels) > 0 {
		normalized := make(map[string]bool, len(acceptLabels))
		for l := range acceptLabels {
			normalized[normalizeLabel(l)] = true
		}
		_, ok := firstMatchingLabel(issueLabels, normalized)
		return ok
	}
	for _, l := range issueLabels {
		if strings.Contains(normalizeLabel(l.Name), "bounty") {
			return true
		}
	}
	return false
}

// textMatches confirms a text-search hit really contains the phrase. GitHub
// search matches word stems, so "bounty" also returns "bounties".
func textMatches(gh GitHubIssue, ts TextSearch) bool {
	text := gh.Title
	if ts.In == "body" {
		text = gh.Body
	}
	fold := func(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
	return strings.Contains(fold(text), fold(ts.Phrase))
}

// truncate shortens s to at most n runes without splitting a character.
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// shorten collapses whitespace and truncates a message for log lines.
func shorten(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "..."
}

func (c *Client) convertIssue(gh GitHubIssue) (Issue, error) {
	labels := make([]string, len(gh.Labels))
	for i, l := range gh.Labels {
		labels[i] = l.Name
	}

	repoName := ExtractRepoName(gh.HTMLURL)

	openPRCount, err := c.GetOpenPRCount(repoName, gh.Number)
	if err != nil {
		return Issue{}, fmt.Errorf("counting open PRs for %s#%d: %w", repoName, gh.Number, err)
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

// ExtractRepoName extracts "owner/repo" from a GitHub issue URL.
func ExtractRepoName(issueURL string) string {
	parts := strings.Split(issueURL, "/")
	if len(parts) >= 5 {
		return parts[3] + "/" + parts[4]
	}
	return ""
}
