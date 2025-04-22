package gh

import (
	"context"
	"log/slog"

	"github.com/google/go-github/v70/github"
)

type IntegrityConfig struct {
	ApprovingCount       int
	SameAuthorCanApprove bool
	RequireSignatures    bool
	AllowForcePushes     bool
}

type GhClient struct {
	client *github.Client
}

func NewClient(token string) *GhClient {
	return &GhClient{
		client: github.NewClient(nil).WithAuthToken(token),
	}
}

type RepoInfo struct {
	CloneUrl      string
	DefaultBranch string
}

func (client *GhClient) GetRepositoryInfo(owner, repo string) (*RepoInfo, error) {

	// possible protection rules https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets
	r, _, err := client.client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		return nil, err
	}

	defaultBranch := r.GetDefaultBranch()

	return &RepoInfo{CloneUrl: *r.CloneURL, DefaultBranch: defaultBranch}, nil
}

func (client *GhClient) GetIntegrityConfig(owner, repo, branch string) (*IntegrityConfig, error) {

	slog.Default().Debug("Getting integrity config", "owner", owner, "repo", repo, "branch", branch)
	protection, _, err := client.client.Repositories.GetBranchProtection(context.Background(), owner, repo, branch)
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
