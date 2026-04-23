package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const projectItemsQuery = `
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

// graphQLRequest is the payload sent to the GitHub GraphQL API.
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

// graphQLResponse wraps the top-level GraphQL response.
type graphQLResponse struct {
	Data   graphQLData    `json:"data"`
	Errors []graphQLError `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLData struct {
	Organization graphQLOrg `json:"organization"`
}

type graphQLOrg struct {
	ProjectV2 *graphQLProject `json:"projectV2"`
}

type graphQLProject struct {
	Items graphQLItems `json:"items"`
}

type graphQLItems struct {
	PageInfo graphQLPageInfo     `json:"pageInfo"`
	Nodes    []graphQLItemNode   `json:"nodes"`
}

type graphQLPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type graphQLItemNode struct {
	Content *graphQLIssueContent `json:"content"`
}

type graphQLIssueContent struct {
	Number     int                `json:"number"`
	Title      string             `json:"title"`
	URL        string             `json:"url"`
	State      string             `json:"state"`
	CreatedAt  string             `json:"createdAt"`
	UpdatedAt  string             `json:"updatedAt"`
	Author     *graphQLAuthor     `json:"author"`
	Body       string             `json:"body"`
	Labels     graphQLLabels      `json:"labels"`
	Repository graphQLRepo        `json:"repository"`
	Comments   graphQLCommentCount `json:"comments"`
}

type graphQLAuthor struct {
	Login string `json:"login"`
}

type graphQLLabels struct {
	Nodes []graphQLLabelNode `json:"nodes"`
}

type graphQLLabelNode struct {
	Name string `json:"name"`
}

type graphQLRepo struct {
	NameWithOwner string `json:"nameWithOwner"`
}

type graphQLCommentCount struct {
	TotalCount int `json:"totalCount"`
}

// graphQLURL returns the GraphQL endpoint based on the client's BaseURL.
func (c *Client) graphQLURL() string {
	base := c.baseURL()
	if base == defaultBaseURL {
		return "https://api.github.com/graphql"
	}
	// For test servers, append /graphql to the base URL.
	return base + "/graphql"
}

// DoGraphQLRequest sends a GraphQL query to the GitHub API and returns the raw response body.
func (c *Client) DoGraphQLRequest(query string, variables map[string]interface{}) ([]byte, error) {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling GraphQL request: %w", err)
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(attempt*5) * time.Second
			fmt.Printf("  Retrying GraphQL in %v (attempt %d/%d)...\n", waitTime, attempt+1, maxRetries)
			time.Sleep(waitTime)
		}

		req, err := http.NewRequest("POST", c.graphQLURL(), bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
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

		return nil, fmt.Errorf("GraphQL HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("GraphQL max retries exceeded")
}

// FetchProjectItems queries a GitHub Projects V2 board and returns the issue-linked items as GitHubIssue slices.
// Draft items (cards without an associated issue) are skipped.
func (c *Client) FetchProjectItems(orgLogin string, projectNumber int) ([]GitHubIssue, error) {
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

		data, err := c.DoGraphQLRequest(projectItemsQuery, variables)
		if err != nil {
			return nil, fmt.Errorf("fetching project items: %w", err)
		}

		var resp graphQLResponse
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
			// Skip draft items — they have nil content
			if node.Content == nil {
				continue
			}

			// Skip items with no issue number (shouldn't happen for Issue content, but be safe)
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
