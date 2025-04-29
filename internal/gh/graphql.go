package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
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

func GetPullRequests(owner, repo, branch, token string) iter.Seq[PR] {
	variables := map[string]any{
		"owner":  owner,
		"name":   repo,
		"branch": branch,
	}

	client := &http.Client{}

	var initialResp PrReviewResponse
	if err := executeGraphQLRequest(client, URL, token, initialPRQuery, variables, &initialResp); err != nil {
		// Return a nil sequence and the error
		return func(yield func(PR) bool) {}
	}

	// Define the iterator function
	return func(yield func(PR) bool) {
		slog.Default().Debug("Iterator started.")
		// Use the already fetched initial response
		currentResp := initialResp

		// Loop indefinitely, relying on break conditions
		for {
			// Yield items from the current page
			for _, pr := range currentResp.Data.Repository.PullRequests.Nodes {
				if !yield(pr) {
					slog.Default().Debug("Iterator stopping early due to yield returning false.")
					return // Stop iteration if yield returns false
				}
			}
			slog.Default().Debug("Finished yielding PRs from current page.")

			// Check if there's a next page
			if !currentResp.Data.Repository.PullRequests.PageInfo.HasNextPage {
				slog.Default().Debug("No next page. Iterator finished.")
				break // Exit loop if no more pages
			}

			// Prepare for the next paginated request
			slog.Default().Debug("Fetching next page...")
			variables["after"] = currentResp.Data.Repository.PullRequests.PageInfo.EndCursor
			var paginatedResp PrReviewResponse

			// Execute the paginated query
			if err := executeGraphQLRequest(client, URL, token, paginatedPRQuery, variables, &paginatedResp); err != nil {
				// Log the error (optional) and stop iteration
				fmt.Printf("Error fetching next page (cursor %v): %v. Iterator stopping.\n", variables["after"], err)
				break // Exit loop on error during pagination
			}
			slog.Default().Debug("Next page fetched successfully.")
			currentResp = paginatedResp // Update currentResp for the next iteration
		}
		slog.Default().Debug("Iterator function finished.")
	}
}
