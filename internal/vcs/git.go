package vcs

import (
	"fmt"
	"log/slog"
	"time"

	"project-integrity-calculator/internal/gh"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

type CommitData struct {
	NumberCommits  int
	NumberVerified int
	Hashs          map[string]*object.Commit
}

// getSignedCommitCount returns the number of commits and the number of verified commits
func GetCommitData(lc *git.Repository, repoDir string, targetBranch string) (*CommitData, error) {

	hash, err := lc.ResolveRevision(plumbing.Revision(targetBranch))
	if err != nil {
		return nil, err
	}

	c, _ := lc.CommitObject(*hash)
	iter, _ := lc.Log(&git.LogOptions{From: c.Hash})
	hashs := make(map[string]*object.Commit)
	cc := 0
	csc := 0

	iter.ForEach(func(curr *object.Commit) error {
		if !curr.Hash.IsZero() {
			patchId, err := GetPatchId(repoDir, curr)
			if err != nil {
				return err
			}
			hashs[patchId] = curr
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

func GetMergedPrHashs(prs []gh.PR, lc *git.Repository, repoDir string) (map[int]map[string]*object.Commit, error) {

	err := fetchAllRefs(prs, lc)
	if err != nil {
		return nil, err
	}

	allNewCommits := make(map[int]map[string]*object.Commit)

	for _, pr := range prs {

		newCommits := getNewCommitsFromPr(pr, lc)
		patchIdCommits := make(map[string]*object.Commit, len(newCommits))
		for _, nc := range newCommits {
			patchId, _ := GetPatchId(repoDir, nc)
			patchIdCommits[patchId] = nc
		}
		allNewCommits[pr.Number] = patchIdCommits
	}

	return allNewCommits, nil
}

func fetchAllRefs(prs []gh.PR, lc *git.Repository) error {
	timer := time.Now()
	refspecs := []config.RefSpec{}
	for _, pr := range prs {
		if pr.State == "MERGED" {
			prn := pr.Number
			// git fetch origin pull/<pr_number>/head:<local_branch_name>
			refspec := fmt.Sprintf("+refs/pull/%d/head:pull/%d", prn, prn)
			refspecs = append(refspecs, config.RefSpec(refspec))
		}
	}

	err := lc.Fetch(&git.FetchOptions{
		RefSpecs: refspecs},
	)
	if err != nil && err.Error() != "already up-to-date" {
		slog.Default().Error("Fetch ran in an error", "error", err)
		return err
	}
	elapsed := time.Since(timer)
	slog.Default().Info("Fetching PR refs from git", "time", elapsed)

	return nil
}

func getNewCommitsFromPr(pr gh.PR, lc *git.Repository) map[string]*object.Commit {

	slog.Default().Info("processing pr", "pr number", pr.Number, "base ref", pr.BaseRefOid, "head ref", pr.HeadRefOid)
	newCommits := make(map[string]*object.Commit)
	iter, _ := lc.Log(&git.LogOptions{From: plumbing.NewHash(pr.HeadRefOid), Order: git.LogOrderCommitterTime})

	iter.ForEach(func(curr *object.Commit) error {
		h := curr.Hash.String()
		slog.Default().Info("first it", "commit", curr)
		newCommits[h] = curr
		if h == pr.BaseRefOid {
			slog.Default().Info("should stop now")
			return storer.ErrStop
		}

		return nil
	})

	iterBase, _ := lc.Log(&git.LogOptions{From: plumbing.NewHash(pr.BaseRefOid), Order: git.LogOrderCommitterTime})
	iterBase.ForEach(func(curr *object.Commit) error {
		slog.Default().Info("second it", "commit", curr)
		delete(newCommits, curr.Hash.String())
		return nil
	})

	// get commit from merge getNewCommit
	// this is used to identify squashed commits
	c, err := lc.CommitObject(plumbing.NewHash(pr.MergeCommit.Oid))
	if err != nil {
		slog.Default().Error("Get commit object failed for merge commit", "id", pr.MergeCommit.Oid, "error", err)
		return newCommits
	}
	newCommits[c.Hash.String()] = c

	return newCommits
}
