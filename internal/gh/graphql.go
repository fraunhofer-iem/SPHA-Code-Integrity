package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
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

const repoDetailsQuery = `
query ($owner: String!, $name: String!) {
	repository(owner: $owner, name: $name) {
    url
    stargazerCount
    languages(first:100,  orderBy: {field: SIZE, direction: DESC}){
      nodes{
        name
      }
    }
    defaultBranchRef{
      name
    }
  }
}
`

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

type RepoInfo struct {
	CloneUrl      string
	DefaultBranch string
	Languages     []string
	Stars         int
}

type RepoInfoResponse struct {
	Data struct {
		Repository struct {
			Url           string `json:"url"`
			Stars         int    `json:"stargazerCount"`
			DefaultBranch struct {
				Name string `json:"name"`
			} `json:"defaultBranchRef"`
			Languages struct {
				Nodes []struct {
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"languages"`
		} `json:"repository"`
	} `json:"data"`
}

func GetRepoInfo(owner, repo, token string) (*RepoInfo, error) {
	slog.Default().Info("Getting repo info")
	variables := map[string]any{
		"owner": owner,
		"name":  repo,
	}

	client := &http.Client{}
	var repoInfoRes RepoInfoResponse

	err := executeGraphQLRequest(client, URL, token, repoDetailsQuery, variables, &repoInfoRes)
	if err != nil {
		slog.Default().Error("Graphql request failed", "err", err)
		return nil, err
	}

	languages := make([]string, 0, len(repoInfoRes.Data.Repository.Languages.Nodes))
	for _, l := range repoInfoRes.Data.Repository.Languages.Nodes {
		languages = append(languages, l.Name)
	}

	return &RepoInfo{
		CloneUrl:      repoInfoRes.Data.Repository.Url,
		Stars:         repoInfoRes.Data.Repository.Stars,
		DefaultBranch: repoInfoRes.Data.Repository.DefaultBranch.Name,
		Languages:     languages,
	}, nil
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
