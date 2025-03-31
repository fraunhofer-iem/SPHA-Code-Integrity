package gh

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v70/github"
)

type CodeIntegrity struct {
	IntegrityConfig
	CommitData
}

type IntegrityConfig struct {
	ApprovingCount       int
	SameAuthorCanApprove bool
	RequireSignatures    bool
	AllowForcePushes     bool
}

func GetIntegrityConfig(owner string, repo string, targetBranch string, gh *github.Client) (*IntegrityConfig, error) {

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

type CommitData struct {
	NumberCommits  int
	NumberVerified int
	Hashs          map[string]struct{}
}

// getSignedCommitCount returns the number of commits and the number of verified commits
func GetCommitData(lc *git.Repository, targetBranch string) (*CommitData, error) {

	hash, err := lc.ResolveRevision(plumbing.Revision(targetBranch))
	if err != nil {
		return nil, err
	}

	fmt.Printf("Hash %s\n", hash.String())
	c, _ := lc.CommitObject(*hash)
	fmt.Printf("Commit %+v\n", c)
	iter, _ := lc.Log(&git.LogOptions{From: c.Hash})

	hashs := make(map[string]struct{})
	cc := 0
	csc := 0

	iter.ForEach(func(curr *object.Commit) error {
		if !curr.Hash.IsZero() {
			hashs[curr.Hash.String()] = struct{}{}
			cc++
			if c.PGPSignature != "" {
				csc++
			}
		}
		return nil
	})

	return &CommitData{
		NumberCommits:  cc,
		NumberVerified: csc,
		Hashs:          hashs,
	}, nil
}
