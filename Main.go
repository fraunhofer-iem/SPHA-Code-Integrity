package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v69/github"
)

// read github token from CLI or environment variable
// query all commits (remember pagination)
// create counter for signed / verified commits and overall counter
// url /repos/{owner}/{repo}/commits // this is stored in $GITHUB_REPOSITORY

var ownerAndRepo = flag.String("ownerAndRepo", "", "owner/repo to query")
var token = flag.String("token", "", "github token to use")

func main() {
	flag.Parse()

	if *ownerAndRepo == "" {
		panic("ownerAndRepo is required")
	}

	if *token == "" {
		panic("token is required")
	}

	client := github.NewClient(nil).WithAuthToken(*token)

	ownerAndRepoSplit := strings.Split(*ownerAndRepo, "/")

	sc, err := getSignedCommitCount(ownerAndRepoSplit[0], ownerAndRepoSplit[1], client)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Number of commits: %d\n", sc.NumberCommits)
	fmt.Printf("Number of verified commits: %d\n", sc.NumberVerified)

	// TODO: check PRs
	// get PR author (we might need to validate that reviews are not from the author)
	// get reviews for PR? do we want to check whether they do reviews or if they enforce reviews through branch protection?
	// get (number of) PRs which were merged with admin rights (without checks)
	// https://api.github.com/repos/OWNER/REPO/branches/BRANCH/protection/required_pull_request_reviews
	// /repos/{owner}/{repo}/branches/{branch}/protection
	// possible protection rules https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets
	// require commits signed
	// require pull request reviews before merging
	// check for bypasses (how often, are they allowed?)
	// force push?
	// require PR, require approval, Require approval of the most recent reviewable push.
	// /repos/{owner}/{repo} --> default_branch

}

type SignedCommit struct {
	NumberCommits  int
	NumberVerified int
}

// getSignedCommitCount returns the number of commits and the number of verified commits
func getSignedCommitCount(owner string, repo string, gh *github.Client) (*SignedCommit, error) {

	numberCommits := 0
	numberVerified := 0
	opt := &github.CommitsListOptions{ListOptions: github.ListOptions{Page: 1, PerPage: 100}}

	for {

		commits, res, err := gh.Repositories.ListCommits(context.Background(), owner, repo, opt)

		if err != nil {
			return nil, err
		}

		numberCommits += len(commits)
		for _, commitResponse := range commits {
			if commitResponse.Commit.Verification.Verified != nil && *commitResponse.Commit.Verification.Verified == true {
				numberVerified++
			}
		}

		if res.NextPage == 0 {
			break
		}
		opt.Page = res.NextPage
	}

	return &SignedCommit{
		NumberCommits:  numberCommits,
		NumberVerified: numberVerified,
	}, nil
}
