package vcs

import (
	"fmt"
	"log"
	"log/slog"
	"time"

	"project-integrity-calculator/internal/gh"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

func GetMergedPrHashs(prs []gh.PR, lc *git.Repository, repoDir string) map[int]map[string]*object.Commit {

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
		log.Fatal(err)
	}
	elapsed := time.Since(timer)
	slog.Default().Info("Fetching PR refs from git", "time", elapsed)

	allNewCommits := make(map[int]map[string]*object.Commit)

	for _, pr := range prs {

		newCommits := getNewCommitsFromPr(pr, lc, repoDir)
		patchIdCommits := make(map[string]*object.Commit, len(newCommits))
		for _, nc := range newCommits {
			patchId, _ := GetPatchId(repoDir, nc)
			patchIdCommits[patchId] = nc
		}
		allNewCommits[pr.Number] = patchIdCommits
	}

	return allNewCommits
}

func getNewCommitsFromPr(pr gh.PR, lc *git.Repository, repoDir string) map[string]*object.Commit {

	newCommits := make(map[string]*object.Commit)
	iter, _ := lc.Log(&git.LogOptions{From: plumbing.NewHash(pr.HeadRefOid), Order: git.LogOrderCommitterTime})

	end := false
	iter.ForEach(func(curr *object.Commit) error {
		if !end {
			newCommits[curr.Hash.String()] = curr
			if curr.Hash.String() == pr.BaseRefOid {
				end = true
			}
			return nil
		} else {
			return nil
		}
	})

	iterBase, _ := lc.Log(&git.LogOptions{From: plumbing.NewHash(pr.BaseRefOid), Order: git.LogOrderCommitterTime})
	iterBase.ForEach(func(curr *object.Commit) error {
		delete(newCommits, curr.Hash.String())
		return nil
	})

	return newCommits
}
