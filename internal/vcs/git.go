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
	"github.com/hashicorp/go-set/v3"
)

type CommitData struct {
	NumberCommits  int
	NumberVerified int
	Hashs          *set.Set[string]
}

// getSignedCommitCount returns the number of commits and the number of verified commits
func GetCommitData(lc *git.Repository, repoDir string, targetBranch string) (*CommitData, error) {

	hash, err := lc.ResolveRevision(plumbing.Revision(targetBranch))
	if err != nil {
		return nil, err
	}

	c, _ := lc.CommitObject(*hash)
	iter, _ := lc.Log(&git.LogOptions{From: c.Hash})
	hashs := set.New[string](100) //make(map[string]*object.Commit)
	cc := 0
	csc := 0

	iter.ForEach(func(curr *object.Commit) error {
		if !curr.Hash.IsZero() {
			patchId, err := GetPatchId(repoDir, curr.Hash.String())
			if err != nil {
				return err
			}
			hashs.Insert(patchId)
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

func GetMergedPrHashs(prs []gh.PR, lc *git.Repository, repoDir string) (*set.Set[string], error) {

	err := fetchAllRefs(prs, lc)
	if err != nil {
		return nil, err
	}

	allNewCommits := set.New[string](len(prs) * 10) //make(map[int]map[string]*object.Commit)

	for _, pr := range prs {

		newCommits := getNewCommitsFromPr(pr, lc)
		// patchIdCommits := make(map[string]*object.Commit, len(newCommits))
		for nc := range newCommits.Items() {
			patchId, _ := GetPatchId(repoDir, nc)
			allNewCommits.Insert(patchId)
			// patchIdCommits[patchId] = nc
		}
		// allNewCommits[pr.Number] = patchIdCommits
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

func getNewCommitsFromPr(pr gh.PR, lc *git.Repository) *set.Set[string] {

	slog.Default().Debug("processing pr", "pr number", pr.Number, "base ref", pr.BaseRefOid, "head ref", pr.HeadRefOid)
	newCommits := set.New[string](100) //make(map[string]*object.Commit)
	iter, _ := lc.Log(&git.LogOptions{From: plumbing.NewHash(pr.HeadRefOid), Order: git.LogOrderCommitterTime})

	iter.ForEach(func(curr *object.Commit) error {
		h := curr.Hash.String()
		// slog.Default().Info("first it", "commit", curr)
		newCommits.Insert(h) //[h] = curr
		if h == pr.BaseRefOid {
			slog.Default().Info("Found base ref and stopping loop")
			return storer.ErrStop
		}

		return nil
	})

	iterBase, _ := lc.Log(&git.LogOptions{From: plumbing.NewHash(pr.BaseRefOid), Order: git.LogOrderCommitterTime})
	iterBase.ForEach(func(curr *object.Commit) error {
		// slog.Default().Info("second it", "commit", curr)
		newCommits.Remove(curr.Hash.String())
		// delete(newCommits, curr.Hash.String())
		return nil
	})

	// get commit from merge getNewCommit
	// this is used to identify squashed commits
	c, err := lc.CommitObject(plumbing.NewHash(pr.MergeCommit.Oid))
	if err != nil {
		slog.Default().Error("Get commit object failed for merge commit", "id", pr.MergeCommit.Oid, "error", err)
		return newCommits
	}
	newCommits.Insert(c.Hash.String())

	return newCommits
}
