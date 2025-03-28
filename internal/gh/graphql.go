package gh

import (
	"bytes"
	"encoding/json"
	"log"
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
				Nodes []struct {
					Number  int    `json:"number"`
					Title   string `json:"title"`
					State   string `json:"state"`
					Reviews struct {
						Nodes []struct {
							State string `json:"state"`
						} `json:"nodes"`
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							StartCursor string `json:"startCursor"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
					} `json:"reviews"`
				} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					StartCursor string `json:"startCursor"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
}

const initialQuery = `
query ($owner: String!, $name: String!, $branch: String!) {
	repository(owner: $owner, name: $name) {
		pullRequests(first: 100, states: MERGED, baseRefName: $branch) {
			nodes {
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

const paginatedQuery = `
query ($owner: String!, $name: String!, $branch: String!, $after: String!) {
repository(owner: $owner, name: $name) {
	pullRequests(first: 100, states: MERGED, baseRefName: $branch, after: $after) {
		nodes {
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
	NumberPRs               int
	NumberSufficientReviews int
	PRNumbers               []int
}

func GetPullRequestStats(owner string, repo string, branch string, token string) (*PullRequestStats, error) {

	// Initial query
	variables := map[string]any{
		"owner":  owner,
		"name":   repo,
		"branch": branch,
	}

	reqPayload := GraphQLRequest{
		Query:     initialQuery,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		log.Fatalf("Failed to marshal request: %v", err)
	}
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	var gqlResp PrReviewResponse
	decoder := json.NewDecoder(resp.Body)

	if err := decoder.Decode(&gqlResp); err != nil {
		log.Fatalf("Failed to decode JSON response: %v", err)
		return nil, err
	}

	// TODO: iterate cursor

	hasNext := gqlResp.Data.Repository.PullRequests.PageInfo.HasNextPage

	for hasNext {
		variables["after"] = gqlResp.Data.Repository.PullRequests.PageInfo.EndCursor
		reqPayload := GraphQLRequest{
			Query:     paginatedQuery,
			Variables: variables,
		}

		jsonData, err := json.Marshal(reqPayload)
		if err != nil {
			log.Fatalf("Failed to marshal request: %v", err)
		}
		req, err = http.NewRequest("POST", URL, bytes.NewBuffer(jsonData))

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Failed to create HTTP request: %v", err)
		}

		decoder := json.NewDecoder(resp.Body)

		if err := decoder.Decode(&gqlResp); err != nil {
			log.Fatalf("Failed to decode JSON response: %v", err)
			return nil, err
		}
		hasNext = gqlResp.Data.Repository.PullRequests.PageInfo.HasNextPage
	}

	return &PullRequestStats{}, nil
}
