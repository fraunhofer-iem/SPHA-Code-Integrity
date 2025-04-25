package processor

import (
	"log/slog"
	"os"
	"project-integrity-calculator/internal/gh"
	"project-integrity-calculator/internal/io"
	"project-integrity-calculator/internal/vcs"
	"time"

	"github.com/hashicorp/go-set/v3"
)

type RepoConfig struct {
	Owner, Repo, Branch, Token, ClonePath, LocalPath, Out string
}

func ProcessRepo(config RepoConfig) (*io.Repo, error) {

	logger := slog.Default()
	timer := time.Now()
	logger.Info("Started processing of", "repo with config", config)
	client := gh.NewClient(config.Token)

	r, err := client.GetRepositoryInfo(config.Owner, config.Repo)
	if err != nil {
		return nil, err
	}

	var dir string
	if config.ClonePath != "" {
		dir = config.ClonePath
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return nil, err
		}
		err = vcs.CloneRepo(r.CloneUrl, dir)
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(dir)
	} else {
		dir = config.LocalPath
	}

	var branch string
	if config.Branch != "" {
		branch = config.Branch
	} else {
		branch = r.DefaultBranch
	}

	methodTimer := time.Now()
	prs, err := gh.GetPullRequests(config.Owner, config.Repo, branch, config.Token)
	if err != nil {
		return nil, err
	}
	elapsed := time.Since(methodTimer)
	logger.Info("Time to query all Pull requests", "time", elapsed)

	methodTimer = time.Now()
	commitsFromPrs, err := vcs.GetCommitShaForMergedPr(prs, dir)
	if err != nil {
		return nil, err
	}
	allCommitsFromPrs := set.New[string](len(*commitsFromPrs) * 5)
	for _, cs := range *commitsFromPrs {
		for c := range cs.Items() {
			pi, err := vcs.GetPatchId(dir, c)
			if err != nil {
				slog.Default().Warn("Get patch id failed", "err", err)
				continue
			}
			allCommitsFromPrs.Insert(pi)
		}
	}

	elapsed = time.Since(methodTimer)
	logger.Info("Time to get pr hashes", "time", elapsed)

	methodTimer = time.Now()
	allCommits, err := vcs.GetCommitsFromBrach(dir, branch)
	if err != nil {
		return nil, err
	}

	allCommitShas := set.New[string](len(allCommits))
	patchIdToCommit := make(map[string]*io.Commit, len(allCommits))
	for _, c := range allCommits {
		pi, err := vcs.GetPatchId(dir, c.GitOID)
		if err != nil {
			slog.Default().Warn("Get patch id failed", "err", err)
			continue
		}
		allCommitShas.Insert(pi)
		patchIdToCommit[pi] = &c
	}

	commitsWithoutPrShas := allCommitShas.Difference(allCommitsFromPrs)
	commitsWithoutPr := make([]io.Commit, commitsWithoutPrShas.Size())

	for h := range commitsWithoutPrShas.Items() {
		c, ok := patchIdToCommit[h]
		if !ok {
			continue
		}

		commitsWithoutPr = append(commitsWithoutPr, *c)
	}

	elapsed = time.Since(methodTimer)
	logger.Info("Time to get commit hashes from target branch", "time", elapsed)

	logger.Info("Number all commits", branch, allCommitShas.Size())

	logger.Info("Number commits from PRs", "number", allCommitsFromPrs.Size())
	logger.Info("Number commits without PR", "number", commitsWithoutPrShas.Size())

	repo := io.Repo{
		Branch: branch,
		Url:    r.CloneUrl,
		// Head:             head,
		CommitsWithoutPR: commitsWithoutPr,
		// UnsignedCommits:  allCommits.UnsignedCommits,
	}

	timerEnd := time.Since(timer)
	logger.Info("Processing of repo finished", "repo", config.Repo, "time", timerEnd)

	return &repo, nil
}
