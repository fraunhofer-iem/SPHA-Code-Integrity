package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v69/github"
)

type CodeIntegrity struct {
	IntegrityConfig
	SignedCommit
}

var (
	ownerAndRepo = flag.String("ownerAndRepo", "", "GitHub repository link (e.g., https://github.com/owner/repo)")
	token        = flag.String("token", "", "GitHub access token")
	targetBranch = flag.String("branch", "", "Target branch to analyze. Defaults to the default branch of the repository")
	mode         = flag.String("mode", "local", "Mode: 'local' or 'clone'")
	localPath    = flag.String("localPath", "", "Path to the local repository (required if mode is 'local')")
	cloneTarget  = flag.String("cloneTarget", "", "Target to clone. Defaults to tmp")
)

func main() {
	// TODO: add input validation
	flag.Parse()

	if *ownerAndRepo == "" {
		panic("ownerAndRepo is required")
	}

	if *token == "" {
		panic("token is required")
	}

	if *mode == "local" && *localPath == "" {
		panic("localPath is required if mode is 'local'")
	}

	if *mode == "clone" && *cloneTarget == "" {
		*cloneTarget = path.Join(os.TempDir(), "codeintegrity")
	}

	client := github.NewClient(nil).WithAuthToken(*token)

	ownerAndRepoSplit := strings.Split(*ownerAndRepo, "/")
	// possible protection rules https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets
	r, _, err := client.Repositories.Get(context.Background(), ownerAndRepoSplit[0], ownerAndRepoSplit[1])
	if err != nil {
		log.Fatal(err)
	}

	if *targetBranch == "" {
		*targetBranch = r.GetDefaultBranch()
	}

	var lc *git.Repository
	if *mode == "clone" {
		// TODO: check auth to make this work on non public repos
		//	Auth: &http.BasicAuth{
		// Username: "abc123", // anything except an empty string
		// Password: "github_access_token",
		// },
		fmt.Printf("Cloning %s to %s\n", *r.CloneURL, *cloneTarget)
		lc, err = git.PlainClone(*cloneTarget, true, &git.CloneOptions{URL: *r.CloneURL})
		defer os.RemoveAll(*cloneTarget)
	} else {
		lc, err = git.PlainOpen(*localPath)
	}
	if err != nil {
		log.Fatal(err)
	}

	sc, err := getSignedCommitCount(lc, *targetBranch)
	fmt.Printf("Number of commits: %d\n", sc.NumberCommits)
	fmt.Printf("Number of verified commits: %d\n", sc.NumberVerified)
}

type IntegrityConfig struct {
	ApprovingCount       int
	SameAuthorCanApprove bool
	RequireSignatures    bool
	AllowForcePushes     bool
}

func getIntegrityConfig(owner string, repo string, targetBranch string, gh *github.Client) (*IntegrityConfig, error) {

	protection, _, err := gh.Repositories.GetBranchProtection(context.Background(), owner, repo, targetBranch)
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
func getSignedCommitCount(lc *git.Repository, targetBranch string) (*SignedCommit, error) {

	hash, err := lc.ResolveRevision(plumbing.Revision(targetBranch))
	if err != nil {
		return nil, err
	}

	fmt.Printf("Hash %s\n", hash.String())
	c, _ := lc.CommitObject(*hash)
	fmt.Printf("Commit %+v\n", c)
	iter, _ := lc.Log(&git.LogOptions{From: c.Hash})

	hashs := []string{}
	cc := 0
	csc := 0

	iter.ForEach(func(c *object.Commit) error {
		if !c.Hash.IsZero() {
			hashs = append(hashs, c.Hash.String())
			cc++
			if c.PGPSignature != "" {
				csc++
			}
		}
		return nil
	})

	return &SignedCommit{
		NumberCommits:  cc,
		NumberVerified: csc,
	}, nil
}
