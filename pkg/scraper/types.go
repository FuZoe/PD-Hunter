package scraper

// Config holds the list of organizations to scan.
type Config struct {
	Organizations []Organization `json:"organizations"`
}

// Organization defines a GitHub org and its bounty labels.
type Organization struct {
	Name   string   `json:"name"`
	Labels []string `json:"labels"`
	Note   string   `json:"note"`
}

// Issue is the normalized bounty issue stored in bounty_issues.json.
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

// GitHubSearchResult is the response from GitHub Search API.
type GitHubSearchResult struct {
	TotalCount int           `json:"total_count"`
	Items      []GitHubIssue `json:"items"`
}

// GitHubIssue represents a single issue from the GitHub API.
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

// GitHubLabel is a label attached to a GitHub issue.
type GitHubLabel struct {
	Name string `json:"name"`
}

// GitHubUser is the author of a GitHub issue.
type GitHubUser struct {
	Login string `json:"login"`
}

// GitHubRepo represents a GitHub repository.
type GitHubRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}
