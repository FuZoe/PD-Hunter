package scraper

// Config holds the list of organizations to scan.
type Config struct {
	Organizations []Organization `json:"organizations"`
	Projects      []ProjectBoard `json:"projects,omitempty"`
}

// Organization defines a GitHub org and its bounty labels.
type Organization struct {
	Name   string   `json:"name"`
	Labels []string `json:"labels"`
	Note   string   `json:"note"`
}

// ProjectBoard defines a GitHub Projects V2 board to scrape.
type ProjectBoard struct {
	OrgLogin      string `json:"org_login"`
	ProjectNumber int    `json:"project_number"`
	Note          string `json:"note"`
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

// HunterIntelligence contains AI-generated analysis and calculated scores.
type HunterIntelligence struct {
	FrictionLevel  string         `json:"friction_level"`
	TechnicalHint  string         `json:"technical_hint"`
	BountyTier     string         `json:"bounty_tier"`
	BountyAmount   int            `json:"bounty_amount"`
	IsHiddenGem    bool           `json:"is_hidden_gem"`
	BountyScore    *int           `json:"bounty_score,omitempty"`
	ScoreBreakdown map[string]int `json:"score_breakdown,omitempty"`
}

// EnrichedIssue represents the final output structure combining raw issue data and AI intelligence.
type EnrichedIssue struct {
	Issue
	HunterIntelligence HunterIntelligence `json:"hunter_intelligence"`
}
