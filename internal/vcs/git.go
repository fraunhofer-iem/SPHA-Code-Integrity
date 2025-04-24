package vcs

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"project-integrity-calculator/internal/gh"
	"project-integrity-calculator/internal/io"

	"github.com/hashicorp/go-set/v3"
)

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
		newCommits, err := getCommitHashsForPr(repoDir, pr)
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

func getCommitHashsForPr(dir string, pr gh.PR) (*set.Set[string], error) {

	slog.Default().Debug("processing pr", "pr number", pr.Number, "base ref", pr.BaseRefOid, "head ref", pr.HeadRefOid)
	newCommits, err := getRevList(dir, fmt.Sprintf("%s...%s", pr.BaseRefOid, pr.HeadRefOid))
	if err != nil {
		return nil, err
	}
	commitSet := set.From(newCommits)
	commitSet.Insert(pr.MergeCommit.Oid)

	return commitSet, nil
}

func GetCommitsFromHashs(repoPath string, hashs []string) ([]io.Commit, error) {
	return getCommit(show, repoPath, hashs)
}

func GetCommitsFromBrach(repoPath, branch string) ([]io.Commit, error) {
	return getCommit(log, repoPath, []string{branch})
}

type GitCmd string

const (
	show GitCmd = "show"
	log  GitCmd = "log"
)

func getCommit(gitCmd GitCmd, repoPath string, input []string) ([]io.Commit, error) {
	format := "--format={\"GitOID\":\"%H\", \"Message\":\"%B\", \"Date\":\"%cd\", \"Signed\":\"%G?\"}"

	args := append([]string{string(gitCmd), "--no-patch", format}, input...)

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		slog.Default().Error("error during get commit", "err", err, "input", input)
		return nil, err
	}
	// Split by newline to get each commit hash.
	rawCommits := strings.Split(strings.TrimSpace(string(out)), "\n")
	slog.Default().Info("commits", "c", rawCommits)

	return parseCommits(rawCommits), nil
}

func parseCommits(rawCommits []string) []io.Commit {
	commits := make([]io.Commit, len(rawCommits))
	for _, rc := range rawCommits {
		var c io.Commit
		err := json.Unmarshal([]byte(rc), &c)
		if err != nil {
			slog.Default().Error("unmarshall for commit failed", "commit", rc)
			continue
		}
		commits = append(commits, c)
	}

	return commits
}

// getRevList executes the git rev-list command and returns commit hashes.
func getRevList(repoPath, arg string) ([]string, error) {
	cmd := exec.Command("git", "rev-list")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// Split by newline to get each commit hash.
	commits := strings.Split(strings.TrimSpace(string(out)), "\n")
	return commits, nil
}

func CloneRepo(url, dir string) error {
	cmd := exec.Command("git", "clone", "--bare", url)
	cmd.Dir = dir
	_, err := cmd.Output()
	if err != nil {
		return err
	}

	return nil
}
