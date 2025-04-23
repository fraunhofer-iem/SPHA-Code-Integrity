package processor

import (
	"fmt"
	"log/slog"
	"os"
	"project-integrity-calculator/internal/gh"
	"project-integrity-calculator/internal/io"
	"project-integrity-calculator/internal/vcs"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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

	lc, err := getRepo(config, r.CloneUrl)
	if err != nil {
		return nil, err
	}

	var dir string
	if config.ClonePath != "" {
		dir = config.ClonePath
		defer os.RemoveAll(config.ClonePath)
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
	mergedPrResults, err := vcs.GetCommitShaForMergedPr(prs, lc, dir)
	if err != nil {
		return nil, err
	}
	elapsed = time.Since(methodTimer)
	logger.Info("Time to get pr hashes", "time", elapsed)

	methodTimer = time.Now()
	allCommits, err := vcs.GetCommitData(lc, dir, branch)
	if err != nil {
		return nil, err
	}
	elapsed = time.Since(methodTimer)
	logger.Info("Time to get commit hashes from target branch", "time", elapsed)

	ach := allCommits.Hashs
	logger.Info("Number all commits", branch, ach.Size())

	commitHashsWithoutPr := ach.Difference(mergedPrResults.PatchIds)

	logger.Info("Number commits from PRs", "number", mergedPrResults.PatchIds.Size())
	logger.Info("Number commits without PR", "number", commitHashsWithoutPr.Size())

	head := ""
	h, err := lc.Head()
	if err == nil {
		head = h.Hash().String()
	}

	// slog.Default().Info("map", "m", mergedPrResults.Hashs)
	commitsWithoutPr := make([]io.Commit, commitHashsWithoutPr.Size())
	for h := range commitHashsWithoutPr.Items() {
		originalHash := mergedPrResults.Hashs[h]
		hash := plumbing.NewHash(originalHash)
		// slog.Default().Info("Hashs", "h", h, "mapvalue", originalHash, "plumbing", hash)

		// todo: commit decoding is pretty expensive, we should add a commit cache
		c, err := lc.CommitObject(hash)
		// slog.Default().Info("resolved commit", "commit", c, "calculated hash", hash, "patchid", h)
		if err != nil {
			continue
		}
		ioc := io.Commit{
			GitOID:  h,
			Message: c.Message,
			Date:    c.Committer.When.String(),
			Signed:  c.PGPSignature != "",
		}
		commitsWithoutPr = append(commitsWithoutPr, ioc)
	}

	repo := io.Repo{
		Branch:           branch,
		Url:              r.CloneUrl,
		Head:             head,
		CommitsWithoutPR: commitsWithoutPr,
		UnsignedCommits:  allCommits.UnsignedCommits,
	}

	timerEnd := time.Since(timer)
	logger.Info("Processing of repo finished", "repo", config.Repo, "time", timerEnd)

	return &repo, nil
}

func getRepo(config RepoConfig, cloneUrl string) (*git.Repository, error) {
	switch {
	case config.ClonePath != "":
		return git.PlainClone(config.ClonePath, true, &git.CloneOptions{
			URL: cloneUrl,
		})
	case config.LocalPath != "":
		return git.PlainOpen(config.LocalPath)
	default:
		return nil, fmt.Errorf("invalid repo config: neither ClonePath nor LocalPath is set")
	}
}
