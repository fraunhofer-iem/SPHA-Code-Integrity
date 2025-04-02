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

func GetIntegrityConfig(owner string, repo string, branch string, gh *github.Client) (*IntegrityConfig, error) {

	slog.Default().Debug("Getting integrity config", "owner", owner, "repo", repo, "branch", branch)
	protection, _, err := gh.Repositories.GetBranchProtection(context.Background(), owner, repo, branch)
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
