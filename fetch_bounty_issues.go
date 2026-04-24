package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Configuration structures
type Config struct {
	Organizations []Organization `json:"organizations"`
	Projects      []ProjectBoard `json:"projects,omitempty"`
}

type Organization struct {
	Name   string   `json:"name"`
	Labels []string `json:"labels"`
	Note   string   `json:"note"`
}

type ProjectBoard struct {
	OrgLogin      string `json:"org_login"`
	ProjectNumber int    `json:"project_number"`
	Note          string `json:"note"`
}

type Issue struct {
	Number       int      `json:"number"`
	Title        string   `json:"title"`
	URL          string   `json:"url"`
	State        string   `json:"state"`
	Labels       []string `json:"labels"`
	CommentCount int      `json:"comment_count"`
	OpenPRCount  int      `json:"open_pr_count"`
	Repository   string   `json:"repository"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	Author       string   `json:"author"`
	Body         string   `json:"body"`
}

type GitHubRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type GitHubLabel struct {
	Name string `json:"name"`
}

type GitHubUser struct {
	Login string `json:"login"`
}

type GitHubSearchResult struct {
	TotalCount int           `json:"total_count"`
	Items      []GitHubIssue `json:"items"`
}

type GitHubIssue struct {
	Number      int           `json:"number"`
	Title       string        `json:"title"`
	HTMLURL     string        `json:"html_url"`
	State       string        `json:"state"`
	Labels      []GitHubLabel `json:"labels"`
	Comments    int           `json:"comments"`
	CreatedAt   string        `json:"created_at"`
	UpdatedAt   string        `json:"updated_at"`
	User        GitHubUser    `json:"user"`
	Body        string        `json:"body"`
	PullRequest *struct{}     `json:"pull_request,omitempty"`
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

const requestDelay = 500 * time.Millisecond

func main() {
	configFile := "mapping.json"
	outputFile := "bounty_issues.json"

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Note: GITHUB_TOKEN not set. Using unauthenticated requests (rate limited to 60/hour)")
	}

	// Load configuration
	config, err := loadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d organizations from config\n", len(config.Organizations))

	var allIssues []Issue
	seenIssues := make(map[string]bool) // Deduplicate by URL

	for _, org := range config.Organizations {
		fmt.Printf("\n=== Scanning organization: %s ===\n", org.Name)
		fmt.Printf("Note: %s\n", org.Note)

		for _, label := range org.Labels {
			fmt.Printf("\nSearching for label: %s\n", label)
			time.Sleep(requestDelay)

			issues, err := searchBountyIssues(org.Name, label, token)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Error searching %s with label '%s': %v\n", org.Name, label, err)
				continue
			}

			for _, ghIssue := range issues {
				// Skip PRs and duplicates
				if ghIssue.PullRequest != nil || seenIssues[ghIssue.HTMLURL] {
					continue
				}
				seenIssues[ghIssue.HTMLURL] = true

				labels := make([]string, len(ghIssue.Labels))
				for j, l := range ghIssue.Labels {
					labels[j] = l.Name
				}

				// Extract repo name from URL
				repoName := extractRepoName(ghIssue.HTMLURL)

				time.Sleep(requestDelay)
				openPRCount := getOpenPRCount(repoName, ghIssue.Number, token)
				fmt.Printf("  Issue #%d: %d open PRs, %d comments - %s\n", ghIssue.Number, openPRCount, ghIssue.Comments, ghIssue.Title[:min(50, len(ghIssue.Title))])

				issue := Issue{
					Number:       ghIssue.Number,
					Title:        ghIssue.Title,
					URL:          ghIssue.HTMLURL,
					State:        ghIssue.State,
					Labels:       labels,
					CommentCount: ghIssue.Comments,
					OpenPRCount:  openPRCount,
					Repository:   repoName,
					CreatedAt:    ghIssue.CreatedAt,
					UpdatedAt:    ghIssue.UpdatedAt,
					Author:       ghIssue.User.Login,
					Body:         ghIssue.Body,
				}
				allIssues = append(allIssues, issue)
			}
		}
	}

	// Build org→labels lookup for filtering project items
	orgLabelsMap := make(map[string]map[string]bool)
	for _, org := range config.Organizations {
		lower := strings.ToLower(org.Name)
		orgLabelsMap[lower] = make(map[string]bool)
		for _, l := range org.Labels {
			orgLabelsMap[lower][strings.ToLower(l)] = true
		}
	}

	// Scan GitHub Projects V2 boards
	for _, proj := range config.Projects {
		fmt.Printf("\n=== Scanning project: %s/%d ===\n", proj.OrgLogin, proj.ProjectNumber)
		fmt.Printf("Note: %s\n", proj.Note)

		projIssues, err := fetchProjectItems(proj.OrgLogin, proj.ProjectNumber, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error fetching project %s/%d: %v\n", proj.OrgLogin, proj.ProjectNumber, err)
			continue
		}

		acceptLabels := orgLabelsMap[strings.ToLower(proj.OrgLogin)]

		newCount := 0
		skippedNoLabel := 0
		for _, ghIssue := range projIssues {
			if ghIssue.PullRequest != nil || seenIssues[ghIssue.HTMLURL] {
				continue
			}

			if !hasBountyLabelLegacy(ghIssue.Labels, acceptLabels) {
				skippedNoLabel++
				continue
			}

			seenIssues[ghIssue.HTMLURL] = true
			newCount++

			labels := make([]string, len(ghIssue.Labels))
			for j, l := range ghIssue.Labels {
				labels[j] = l.Name
			}

			repoName := extractRepoName(ghIssue.HTMLURL)

			time.Sleep(requestDelay)
			openPRCount := getOpenPRCount(repoName, ghIssue.Number, token)
			fmt.Printf("  [project] Issue #%d: %d open PRs, %d comments - %s\n",
				ghIssue.Number, openPRCount, ghIssue.Comments, ghIssue.Title[:min(50, len(ghIssue.Title))])

			issue := Issue{
				Number:       ghIssue.Number,
				Title:        ghIssue.Title,
				URL:          ghIssue.HTMLURL,
				State:        ghIssue.State,
				Labels:       labels,
				CommentCount: ghIssue.Comments,
				OpenPRCount:  openPRCount,
				Repository:   repoName,
				CreatedAt:    ghIssue.CreatedAt,
				UpdatedAt:    ghIssue.UpdatedAt,
				Author:       ghIssue.User.Login,
				Body:         ghIssue.Body,
			}
			allIssues = append(allIssues, issue)
		}
		fmt.Printf("  Project %s/%d: %d total items, %d new, %d skipped (no bounty label)\n",
			proj.OrgLogin, proj.ProjectNumber, len(projIssues), newCount, skippedNoLabel)
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total bounty issues found: %d\n", len(allIssues))

	output, err := json.MarshalIndent(allIssues, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(outputFile, output, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Results saved to: %s\n", outputFile)
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func extractRepoName(issueURL string) string {
	// Extract "owner/repo" from "https://github.com/owner/repo/issues/123"
	parts := strings.Split(issueURL, "/")
	if len(parts) >= 5 {
		return parts[3] + "/" + parts[4]
	}
	return ""
}

func searchBountyIssues(org, label, token string) ([]GitHubIssue, error) {
	var allIssues []GitHubIssue
	page := 1

	for {
		// Use GitHub Search API: is:open is:issue org:ORG label:"LABEL"
		query := fmt.Sprintf("is:open is:issue org:%s label:\"%s\"", org, label)
		apiURL := fmt.Sprintf("https://api.github.com/search/issues?q=%s&per_page=100&page=%d", url.QueryEscape(query), page)

		data, err := doRequest(apiURL, token)
		if err != nil {
			return nil, err
		}

		var result GitHubSearchResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}

		if len(result.Items) == 0 {
			break
		}

		allIssues = append(allIssues, result.Items...)

		// GitHub Search API returns max 1000 results
		if len(allIssues) >= result.TotalCount || page >= 10 {
			break
		}
		page++
		time.Sleep(requestDelay) // Rate limit for search API
	}

	return allIssues, nil
}

func doRequest(reqURL, token string) ([]byte, error) {
	maxRetries := 3

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
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
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

		if resp.StatusCode == 429 || resp.StatusCode == 403 {
			if attempt < maxRetries-1 {
				continue
			}
		}

		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("max retries exceeded")
}

func getOrgRepos(org, token string) ([]GitHubRepo, error) {
	var allRepos []GitHubRepo
	page := 1

	for {
		apiURL := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&page=%d", org, page)
		data, err := doRequest(apiURL, token)
		if err != nil {
			return nil, err
		}

		var repos []GitHubRepo
		if err := json.Unmarshal(data, &repos); err != nil {
			return nil, err
		}

		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
		page++
	}

	return allRepos, nil
}

func getOpenPRCount(repoFullName string, issueNumber int, token string) int {
	// Search for open PRs that mention this issue number in body/title (e.g., "Fixes #2063", "#2063")
	query := fmt.Sprintf("is:pr is:open repo:%s %d", repoFullName, issueNumber)
	apiURL := fmt.Sprintf("https://api.github.com/search/issues?q=%s", url.QueryEscape(query))

	data, err := doRequest(apiURL, token)
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

func hasBountyLabelLegacy(issueLabels []GitHubLabel, acceptLabels map[string]bool) bool {
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

// GraphQL types for GitHub Projects V2
type gqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type gqlResponse struct {
	Data   gqlData    `json:"data"`
	Errors []gqlError `json:"errors,omitempty"`
}

type gqlError struct {
	Message string `json:"message"`
}

type gqlData struct {
	Organization gqlOrg `json:"organization"`
}

type gqlOrg struct {
	ProjectV2 *gqlProject `json:"projectV2"`
}

type gqlProject struct {
	Items gqlItems `json:"items"`
}

type gqlItems struct {
	PageInfo gqlPageInfo   `json:"pageInfo"`
	Nodes    []gqlItemNode `json:"nodes"`
}

type gqlPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type gqlItemNode struct {
	Content *gqlIssueContent `json:"content"`
}

type gqlIssueContent struct {
	Number     int             `json:"number"`
	Title      string          `json:"title"`
	URL        string          `json:"url"`
	State      string          `json:"state"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
	Author     *gqlAuthor      `json:"author"`
	Body       string          `json:"body"`
	Labels     gqlLabels       `json:"labels"`
	Repository gqlRepo         `json:"repository"`
	Comments   gqlCommentCount `json:"comments"`
}

type gqlAuthor struct {
	Login string `json:"login"`
}

type gqlLabels struct {
	Nodes []gqlLabelNode `json:"nodes"`
}

type gqlLabelNode struct {
	Name string `json:"name"`
}

type gqlRepo struct {
	NameWithOwner string `json:"nameWithOwner"`
}

type gqlCommentCount struct {
	TotalCount int `json:"totalCount"`
}

const projectItemsQueryLegacy = `
query($org: String!, $number: Int!, $cursor: String) {
  organization(login: $org) {
    projectV2(number: $number) {
      items(first: 100, after: $cursor) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          content {
            ... on Issue {
              number
              title
              url
              state
              createdAt
              updatedAt
              author { login }
              body
              labels(first: 20) {
                nodes { name }
              }
              repository {
                nameWithOwner
              }
              comments { totalCount }
            }
          }
        }
      }
    }
  }
}
`

func doGraphQLRequest(query string, variables map[string]interface{}, token string) ([]byte, error) {
	reqBody := gqlRequest{
		Query:     query,
		Variables: variables,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling GraphQL request: %w", err)
	}

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(attempt*5) * time.Second
			fmt.Printf("  Retrying GraphQL in %v (attempt %d/%d)...\n", waitTime, attempt+1, maxRetries)
			time.Sleep(waitTime)
		}

		req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
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

		if resp.StatusCode == 429 || resp.StatusCode == 403 {
			if attempt < maxRetries-1 {
				continue
			}
		}

		return nil, fmt.Errorf("GraphQL HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("GraphQL max retries exceeded")
}

func fetchProjectItems(orgLogin string, projectNumber int, token string) ([]GitHubIssue, error) {
	var allIssues []GitHubIssue
	var cursor *string

	for {
		variables := map[string]interface{}{
			"org":    orgLogin,
			"number": projectNumber,
		}
		if cursor != nil {
			variables["cursor"] = *cursor
		}

		time.Sleep(requestDelay)

		data, err := doGraphQLRequest(projectItemsQueryLegacy, variables, token)
		if err != nil {
			return nil, fmt.Errorf("fetching project items: %w", err)
		}

		var resp gqlResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parsing GraphQL response: %w", err)
		}

		if len(resp.Errors) > 0 {
			return nil, fmt.Errorf("GraphQL error: %s", resp.Errors[0].Message)
		}

		if resp.Data.Organization.ProjectV2 == nil {
			return nil, fmt.Errorf("project %s/%d not found or not accessible", orgLogin, projectNumber)
		}

		items := resp.Data.Organization.ProjectV2.Items

		for _, node := range items.Nodes {
			if node.Content == nil {
				continue
			}
			if node.Content.Number == 0 {
				continue
			}

			labels := make([]GitHubLabel, len(node.Content.Labels.Nodes))
			for i, l := range node.Content.Labels.Nodes {
				labels[i] = GitHubLabel{Name: l.Name}
			}

			author := ""
			if node.Content.Author != nil {
				author = node.Content.Author.Login
			}

			ghIssue := GitHubIssue{
				Number:    node.Content.Number,
				Title:     node.Content.Title,
				HTMLURL:   node.Content.URL,
				State:     node.Content.State,
				Labels:    labels,
				Comments:  node.Content.Comments.TotalCount,
				CreatedAt: node.Content.CreatedAt,
				UpdatedAt: node.Content.UpdatedAt,
				User:      GitHubUser{Login: author},
				Body:      node.Content.Body,
			}
			allIssues = append(allIssues, ghIssue)
		}

		if !items.PageInfo.HasNextPage {
			break
		}
		cursor = &items.PageInfo.EndCursor
	}

	return allIssues, nil
}

func getBountyIssues(org, repo, label, token string) ([]GitHubIssue, error) {
	var allIssues []GitHubIssue
	page := 1

	for {
		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=open&labels=%s&per_page=100&page=%d",
			org, repo, url.QueryEscape(label), page)

		data, err := doRequest(apiURL, token)
		if err != nil {
			return nil, err
		}

		var issues []GitHubIssue
		if err := json.Unmarshal(data, &issues); err != nil {
			return nil, err
		}

		if len(issues) == 0 {
			break
		}

		allIssues = append(allIssues, issues...)
		page++
	}

	return allIssues, nil
}
