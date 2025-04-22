package vcs

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"project-integrity-calculator/internal/gh"
	"project-integrity-calculator/internal/io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-set/v3"
)

type CommitData struct {
	UnsignedCommits []io.Commit
	Hashs           *set.Set[string]
}

// getSignedCommitCount returns the number of commits and the number of verified commits
func GetCommitData(lc *git.Repository, repoDir string, targetBranch string) (*CommitData, error) {

	hash, err := lc.ResolveRevision(plumbing.Revision(targetBranch))
	if err != nil {
		return nil, err
	}

	c, _ := lc.CommitObject(*hash)
	iter, _ := lc.Log(&git.LogOptions{From: c.Hash})
	hashs := set.New[string](100)
	unsignedCommits := make([]io.Commit, 100)

	iter.ForEach(func(curr *object.Commit) error {
		if !curr.Hash.IsZero() {
			hashString := curr.Hash.String()
			patchId, err := GetPatchId(repoDir, hashString)
			if err != nil {
				return err
			}
			hashs.Insert(patchId)
			if curr.PGPSignature != "" {
				commit := io.Commit{
					Message: curr.Message,
					GitOID:  hashString,
					Date:    curr.Committer.When.String(),
					Signed:  false,
				}
				unsignedCommits = append(unsignedCommits, commit)
			}
		}
		return nil
	})

	return &CommitData{
		Hashs:           hashs,
		UnsignedCommits: unsignedCommits,
	}, nil
}

func GetMergedPrHashs(prs []gh.PR, lc *git.Repository, repoDir string) (*set.Set[string], error) {

	err := fetchAllRefs(prs, lc)
	if err != nil {
		return nil, err
	}

	allNewCommits := set.New[string](len(prs) * 10)

	for _, pr := range prs {

		newCommits, err := getNewCommitsFromPr(repoDir, pr, lc)
		if err != nil {
			continue
		}
		for nc := range newCommits.Items() {
			patchId, _ := GetPatchId(repoDir, nc)
			allNewCommits.Insert(patchId)
		}
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

func getNewCommitsFromPr(dir string, pr gh.PR, lc *git.Repository) (*set.Set[string], error) {

	slog.Default().Debug("processing pr", "pr number", pr.Number, "base ref", pr.BaseRefOid, "head ref", pr.HeadRefOid)
	newCommits, err := getRevList(dir, pr.BaseRefOid, pr.HeadRefOid)
	if err != nil {
		return nil, err
	}
	commitSet := set.From(newCommits)

	// this is used to identify squashed and merge commits
	if c, err := lc.CommitObject(plumbing.NewHash(pr.MergeCommit.Oid)); err == nil {
		commitSet.Insert(c.Hash.String())
	}
	return commitSet, nil
}

// getRevList executes the git rev-list command and returns commit hashes.
func getRevList(repoPath, baseRef, headRef string) ([]string, error) {
	cmd := exec.Command("git", "rev-list", fmt.Sprintf("%s..%s", baseRef, headRef))
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// Split by newline to get each commit hash.
	commits := strings.Split(strings.TrimSpace(string(out)), "\n")
	return commits, nil
}
