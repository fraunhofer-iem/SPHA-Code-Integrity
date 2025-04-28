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

	r, err := gh.GetRepoInfo(config.Owner, config.Repo, config.Token)
	if err != nil {
		return nil, err
	}

	// TODO: fix me !
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
			if err != nil || pi == "" {
				slog.Default().Warn("Get patch id failed. Setting patch id to original commit id", "err", err)
				pi = c
			}
			allCommitsFromPrs.Insert(pi)
		}
	}
	logger.Info("Commits from prs", "prs", commitsFromPrs)
	elapsed = time.Since(methodTimer)
	logger.Info("Time to get pr hashes", "time", elapsed)

	methodTimer = time.Now()
	allCommits, err := vcs.GetCommitsFromBrach(dir, branch)
	logger.Info("All commits", "commits", allCommits)
	if err != nil {
		return nil, err
	}

	allCommitShas := set.New[string](len(allCommits))
	patchIdToCommit := make(map[string]*io.Commit, len(allCommits))
	unsignedCommits := make([]io.Commit, 0, len(allCommits)/3)
	for i := range allCommits {
		c := &allCommits[i]
		pi, err := vcs.GetPatchId(dir, c.GitOID)
		logger.Info("getting patch id", "commit", c, "patch id", pi)
		if err != nil || pi == "" {
			slog.Default().Warn("Get patch id failed or is empty. Setting patch id to original commit id", "err", err)
			pi = c.GitOID
		}
		allCommitShas.Insert(pi)
		patchIdToCommit[pi] = c
		if c.Signed == "N" || c.Signed == "B" {
			unsignedCommits = append(unsignedCommits, *c)
		}
	}

	logger.Info("all commits sha", "shas", allCommitShas)

	commitsWithoutPrShas := allCommitShas.Difference(allCommitsFromPrs)
	commitsWithoutPr := make([]io.Commit, 0, commitsWithoutPrShas.Size())

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

	heads, err := vcs.GetCommitsFromHashs(dir, []string{branch})
	head := ""
	if err == nil || len(heads) == 1 {
		head = heads[0].GitOID
	}

	repo := io.Repo{
		Branch:           branch,
		Url:              r.CloneUrl,
		Head:             head,
		CommitsWithoutPR: commitsWithoutPr,
		UnsignedCommits:  unsignedCommits,
		Stats: io.Stats{
			NumberCommits: allCommitShas.Size(),
			NumberPRs:     len(*commitsFromPrs),
			Stars:         r.Stars,
			Languages:     r.Languages,
		},
	}

	timerEnd := time.Since(timer)
	logger.Info("Processing of repo finished", "repo", config.Repo, "time", timerEnd)

	return &repo, nil
}
