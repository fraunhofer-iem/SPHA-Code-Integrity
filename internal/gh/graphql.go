package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type PrReviewResponse struct {
	Data struct {
		Repository struct {
			PullRequests struct {
				Nodes    []PR       `json:"nodes"`
				PageInfo Pagination `json:"pageInfo"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
}

type Pagination struct {
	HasNextPage bool   `json:"hasNextPage"`
	StartCursor string `json:"startCursor"`
	EndCursor   string `json:"endCursor"`
}

type PR struct {
	BaseRefOid string `json:"baseRefOid"`
	HeadRefOid string `json:"headRefOid"`
	Number     int    `json:"number"`
	Title      string `json:"title"`
	State      string `json:"state"`
	Reviews    struct {
		Nodes []struct {
			State string `json:"state"`
		} `json:"nodes"`
		PageInfo Pagination `json:"pageInfo"`
	} `json:"reviews"`
}

const initialPRQuery = `
query ($owner: String!, $name: String!, $branch: String!) {
	repository(owner: $owner, name: $name) {
		pullRequests(first: 100, states: MERGED, baseRefName: $branch) {
			nodes {
			    baseRefOid
        		headRefOid
				number
				title
				state
				reviews(first: 100) {
				nodes {
					state
				}
				pageInfo {
					hasNextPage
					startCursor
					endCursor
				}
				}
			}
			pageInfo {
				hasNextPage
				startCursor
				endCursor
			}
		}
	}
}
`

const paginatedPRQuery = `
query ($owner: String!, $name: String!, $branch: String!, $after: String!) {
repository(owner: $owner, name: $name) {
	pullRequests(first: 100, states: MERGED, baseRefName: $branch, after: $after) {
		nodes {
			baseRefOid
   		    headRefOid
			number
			title
			state
			reviews(first: 100) {
			nodes {
				state
			}
			pageInfo {
				hasNextPage
				startCursor
				endCursor
			}
			}
		}
		pageInfo {
			hasNextPage
			startCursor
			endCursor
		}
	}
}
}
`

const URL = "https://api.github.com/graphql"

type PullRequestStats struct {
	NumberSufficientReviews int
	PRNumbers               []int
}

// Helper function to execute a GraphQL request.
func executeGraphQLRequest(client *http.Client, url, token, query string, variables map[string]any, result any) error {
	reqPayload := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(result); err != nil {
		return fmt.Errorf("failed to decode JSON response: %v", err)
	}
	return nil
}

func GetPullRequestStats(owner, repo, branch, token string, threshold int) (*PullRequestStats, error) {
	variables := map[string]any{
		"owner":  owner,
		"name":   repo,
		"branch": branch,
	}

	client := &http.Client{}
	var gqlResp PrReviewResponse

	// Execute the initial query.
	if err := executeGraphQLRequest(client, URL, token, initialPRQuery, variables, &gqlResp); err != nil {
		return nil, err
	}

	stats := &PullRequestStats{
		PRNumbers: []int{},
	}

	fillPrStats(gqlResp.Data.Repository.PullRequests.Nodes, threshold, stats)

	// Cursor iteration for paginated results.
	for gqlResp.Data.Repository.PullRequests.PageInfo.HasNextPage {
		// Set the cursor for the next page.
		variables["after"] = gqlResp.Data.Repository.PullRequests.PageInfo.EndCursor

		if err := executeGraphQLRequest(client, URL, token, paginatedPRQuery, variables, &gqlResp); err != nil {
			break
		}

		fillPrStats(gqlResp.Data.Repository.PullRequests.Nodes, threshold, stats)
	}

	return stats, nil
}

func fillPrStats(prs []PR, threshold int, stats *PullRequestStats) {

	// We only take the first 100 reviews into account. However, if you have
	// more than 100 reviews, you have different problems.
	for _, pr := range prs {
		stats.PRNumbers = append(stats.PRNumbers, pr.Number)
		ac := 0
		for _, r := range pr.Reviews.Nodes {
			if r.State == "APPROVED" {
				ac++
			}

			if ac >= threshold {
				stats.NumberSufficientReviews++
				break
			}
		}
	}
}
