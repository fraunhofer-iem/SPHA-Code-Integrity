package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type GraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
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
	BaseRefOid  string      `json:"baseRefOid"`
	HeadRefOid  string      `json:"headRefOid"`
	Number      int         `json:"number"`
	Title       string      `json:"title"`
	State       string      `json:"state"`
	MergeCommit MergeCommit `json:"mergeCommit"`
	Reviews     struct {
		Nodes []struct {
			State string `json:"state"`
		} `json:"nodes"`
		PageInfo Pagination `json:"pageInfo"`
	} `json:"reviews"`
}

type MergeCommit struct {
	Oid     string `json:"oid"`
	Message string `json:"message"`
}

const initialPRQuery = `
query ($owner: String!, $name: String!, $branch: String!) {
	repository(owner: $owner, name: $name) {
		pullRequests(first: 100, states: MERGED, baseRefName: $branch) {
			nodes {
				mergeCommit {
		            id
		            oid
		            message
		        }
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
			mergeCommit {
	            id
	            oid
	            message
	        }
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

var RequestCounter = 0

// Helper function to execute a GraphQL request.
func executeGraphQLRequest(client *http.Client, url, token, query string, variables map[string]any, result *PrReviewResponse) error {
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
	RequestCounter++
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

func GetPullRequests(owner, repo, branch, token string) ([]PR, error) {
	variables := map[string]any{
		"owner":  owner,
		"name":   repo,
		"branch": branch,
	}

	client := &http.Client{}
	var gqlResp PrReviewResponse
	prs := []PR{}

	// Execute the initial query.
	if err := executeGraphQLRequest(client, URL, token, initialPRQuery, variables, &gqlResp); err != nil {
		return prs, err
	}

	prs = append(prs, gqlResp.Data.Repository.PullRequests.Nodes...)

	// Cursor iteration for paginated results.
	for gqlResp.Data.Repository.PullRequests.PageInfo.HasNextPage {
		// Set the cursor for the next page.
		variables["after"] = gqlResp.Data.Repository.PullRequests.PageInfo.EndCursor

		if err := executeGraphQLRequest(client, URL, token, paginatedPRQuery, variables, &gqlResp); err != nil {
			break
		}

		prs = append(prs, gqlResp.Data.Repository.PullRequests.Nodes...)
	}

	return prs, nil
}
