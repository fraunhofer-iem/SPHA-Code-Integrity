package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v69/github"
)

type CodeIntegrity struct {
	IntegrityConfig
	SignedCommit
}

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
	// TODO: add input validation
	ownerAndRepoSplit := strings.Split(*ownerAndRepo, "/")
	// possible protection rules https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets
	r, _, err := client.Repositories.Get(context.Background(), ownerAndRepoSplit[0], ownerAndRepoSplit[1])
	if err != nil {
		log.Fatal(err)
	}

	defaultBranch := r.GetDefaultBranch()
	sc, err := getSignedCommitCount(ownerAndRepoSplit[0], ownerAndRepoSplit[1], defaultBranch, client)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Number of commits: %d\n", sc.NumberCommits)
	fmt.Printf("Number of verified commits: %d\n", sc.NumberVerified)

	ic, err := getIntegrityConfig(ownerAndRepoSplit[0], ownerAndRepoSplit[1], defaultBranch, client)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Number of required reviewers: %d\n", ic.ApprovingCount)
	fmt.Printf("Require last push approval: %t\n", ic.SameAuthorCanApprove)
	fmt.Printf("Require signatures: %t\n", ic.RequireSignatures)

	// how many PRs without review
	// get all PRs
	// filter by target branch (should be default (or protected if we want to be more precise))
	// filter by merged
	// get reviews for PR
}

type IntegrityConfig struct {
	ApprovingCount       int
	SameAuthorCanApprove bool
	RequireSignatures    bool
	AllowForcePushes     bool
}

func getIntegrityConfig(owner string, repo string, defaultBranch string, gh *github.Client) (*IntegrityConfig, error) {

	protection, _, err := gh.Repositories.GetBranchProtection(context.Background(), owner, repo, defaultBranch)
	if err != nil {
		return nil, err
	}

	pr := protection.GetRequiredPullRequestReviews()

	reviewerCount := 0
	sameAutor := true

	if pr != nil {
		reviewerCount = pr.RequiredApprovingReviewCount
		sameAutor = pr.RequireLastPushApproval
	}

	signaturesEnforced := false
	sigProtection := protection.GetRequiredSignatures()
	if sigProtection != nil {
		signaturesEnforced = *sigProtection.Enabled
	}

	allowForcePushes := true
	fp := protection.AllowForcePushes
	if fp != nil {
		allowForcePushes = fp.Enabled
	}

	return &IntegrityConfig{
		ApprovingCount:       reviewerCount,
		SameAuthorCanApprove: sameAutor,
		RequireSignatures:    signaturesEnforced,
		AllowForcePushes:     allowForcePushes,
	}, nil
}

type SignedCommit struct {
	NumberCommits  int
	NumberVerified int
}

// getSignedCommitCount returns the number of commits and the number of verified commits
func getSignedCommitCount(owner string, repo string, defaultBranch string, gh *github.Client) (*SignedCommit, error) {

	numberCommits := 0
	numberVerified := 0

	commitOpt := &github.CommitsListOptions{SHA: defaultBranch, ListOptions: github.ListOptions{Page: 1, PerPage: 100}}

	allCommitSHAs := make(map[string]struct{})

	for {

		commits, res, err := gh.Repositories.ListCommits(context.Background(), owner, repo, commitOpt)

		if err != nil {
			return nil, err
		}

		for _, commit := range commits {
			allCommitSHAs[commit.GetSHA()] = struct{}{}
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
		commitOpt.Page = res.NextPage
	}

	prOpt := &github.PullRequestListOptions{
		State:       "closed",
		Base:        defaultBranch,
		ListOptions: github.ListOptions{Page: 1, PerPage: 100},
	}

	for {
		prs, res, err := gh.PullRequests.List(context.Background(), owner, repo, prOpt)
		if err != nil {
			// TODO: we can still calculate part of the stats
			return nil, err
		}

		for _, pr := range prs {
			for {
				commits, commitRes, err := gh.PullRequests.ListCommits(context.Background(), owner, repo, pr.GetNumber(), &prOpt.ListOptions)
				if err != nil {
					// TODO:
				}

				for _, commit := range commits {
					delete(allCommitSHAs, *commit.SHA)
				}

				if commitRes.NextPage == 0 {
					break
				}
				prOpt.Page = commitRes.NextPage
			}
		}

		if res.NextPage == 0 {
			break
		}
		prOpt.Page = res.NextPage
	}
	fmt.Printf("%d Commits not from PRs\n", len(allCommitSHAs))

	return &SignedCommit{
		NumberCommits:  numberCommits,
		NumberVerified: numberVerified,
	}, nil
}
