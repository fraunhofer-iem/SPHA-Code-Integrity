package vcs

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"project-integrity-calculator/internal/gh"
	"project-integrity-calculator/internal/io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-set/v3"
)

type CommitData struct {
	UnsignedCommits []io.Commit
	Hashs           *set.Set[string]
}

// getSignedCommitCount returns the number of commits and the number of verified commits
func GetCommitData(repoDir string, targetBranch string) (*CommitData, error) {

	// do external process call to git log or git show to get this info
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

func GetCommitShaForMergedPr(prs []gh.PR, repoDir string) (*map[int]*set.Set[string], error) {

	if len(prs) == 0 {
		slog.Default().Warn("No PRs provided to GetCommitShaForMergedPr", "repoDir", repoDir)
		return &map[int]*set.Set[string]{}, nil
	}

	err := fetchAllRefs(prs, repoDir)
	if err != nil {
		return nil, err
	}

	res := make(map[int]*set.Set[string], len(prs))

	for _, pr := range prs {
		newCommits, err := getNewCommitsFromPr(repoDir, pr)
		if err != nil {
			continue
		}
		res[pr.Number] = newCommits
	}

	return &res, nil
}

func fetchAllRefs(prs []gh.PR, dir string) error {
	if len(prs) == 0 {
		return nil
	}

	refs := "origin "
	for _, pr := range prs {
		if pr.State == "MERGED" {
			prn := pr.Number
			refs += fmt.Sprintf("pull/%d/head:pull/%d ", prn, prn)
		}
	}

	// git fetch origin pull/<pr_number>/head:<local_branch_name> ...
	cmd := exec.Command("git", "fetch", refs)
	cmd.Dir = dir
	_, err := cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

func getNewCommitsFromPr(dir string, pr gh.PR) (*set.Set[string], error) {

	slog.Default().Debug("processing pr", "pr number", pr.Number, "base ref", pr.BaseRefOid, "head ref", pr.HeadRefOid)
	newCommits, err := getRevList(dir, pr.BaseRefOid, pr.HeadRefOid)
	if err != nil {
		return nil, err
	}
	commitSet := set.From(newCommits)

	commitSet.Insert(pr.MergeCommit.Oid)
	GetCommit(dir, commitSet.Slice())

	return commitSet, nil
}

func GetCommit(repoPath string, hashs []string) { //(*[]io.Commit, error) {
	format := "--format={\"GitOID\":\"%H\", \"Message\":\"%B\", \"Date\":\"%cd\", \"Signed\":\"%G?\"}"
	args := append([]string{"show", "--no-patch", format}, hashs...)

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		slog.Default().Error("error during get commit", "err", err, "hashs", hashs)
		// return nil, err
	}
	// Split by newline to get each commit hash.
	commits := strings.Split(strings.TrimSpace(string(out)), "\n")
	slog.Default().Info("commits", "c", commits)
	// return commits, nil
}

// getRevList executes the git rev-list command and returns commit hashes.
func getRevList(repoPath, baseRef, headRef string) ([]string, error) {
	cmd := exec.Command("git", "rev-list", fmt.Sprintf("%s...%s", baseRef, headRef))
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// Split by newline to get each commit hash.
	commits := strings.Split(strings.TrimSpace(string(out)), "\n")
	return commits, nil
}
