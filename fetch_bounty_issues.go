package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Issue struct {
	Number       int      `json:"number"`
	Title        string   `json:"title"`
	URL          string   `json:"url"`
	State        string   `json:"state"`
	Labels       []string `json:"labels"`
	CommentCount int      `json:"comment_count"`
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
	org := "projectdiscovery"
	label := "💎 Bounty"
	outputFile := "bounty_issues.json"

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Note: GITHUB_TOKEN not set. Using unauthenticated requests (rate limited to 60/hour)")
	}

	fmt.Printf("Fetching repositories from organization: %s\n", org)

	repos, err := getOrgRepos(org, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching repos: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d repositories\n", len(repos))

	var allIssues []Issue

	for i, repo := range repos {
		repoName := repo.FullName
		fmt.Printf("[%d/%d] Checking %s...\n", i+1, len(repos), repoName)

		time.Sleep(requestDelay)
		issues, err := getBountyIssues(org, repo.Name, label, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error fetching issues from %s: %v\n", repoName, err)
			continue
		}

		for _, ghIssue := range issues {
			if ghIssue.PullRequest != nil {
				continue
			}
			labels := make([]string, len(ghIssue.Labels))
			for j, l := range ghIssue.Labels {
				labels[j] = l.Name
			}

			issue := Issue{
				Number:       ghIssue.Number,
				Title:        ghIssue.Title,
				URL:          ghIssue.HTMLURL,
				State:        ghIssue.State,
				Labels:       labels,
				CommentCount: ghIssue.Comments,
				Repository:   repoName,
				CreatedAt:    ghIssue.CreatedAt,
				UpdatedAt:    ghIssue.UpdatedAt,
				Author:       ghIssue.User.Login,
				Body:         ghIssue.Body,
			}
			allIssues = append(allIssues, issue)
		}
	}

	fmt.Printf("\nTotal bounty issues found: %d\n", len(allIssues))

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
